package pulseagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var agentMu sync.Mutex
var agentCmd *exec.Cmd

const (
	defaultSocket = "/run/pulse/agent.sock"
	defaultBin    = "/opt/pulse/bin/pulse-agent"
)

type LicenseSnapshot struct {
	Edition      string         `json:"edition"`
	IsLicensed   bool           `json:"is_licensed"`
	Customer     string         `json:"customer"`
	LicenseType  string         `json:"license_type"`
	ExpiresAt    string         `json:"expires_at"`
	Features     map[string]any `json:"features"`
	FeatureKeys  []string       `json:"feature_keys"`
	LicenseHash  string         `json:"license_hash"`
	Error        string         `json:"error"`
	AgentVersion string         `json:"agent_version"`
}

type TelemetryResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

func Evaluate(ctx context.Context, product, edition, key, installationID string, timeout time.Duration) (*LicenseSnapshot, error) {
	var result LicenseSnapshot
	err := post(ctx, timeout, "/v1/license/evaluate", map[string]string{
		"product": product, "edition": edition, "license_key": key, "installation_id": installationID,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func SubmitTelemetry(ctx context.Context, timeout time.Duration, payload map[string]any) (*TelemetryResponse, error) {
	var result TelemetryResponse
	err := post(ctx, timeout, "/v1/telemetry/submit", payload, &result)
	return &result, err
}

func post(ctx context.Context, timeout time.Duration, path string, payload, out any) error {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	bin := os.Getenv("PULSE_AGENT_BIN")
	if bin == "" {
		bin = defaultBin
	}
	socket := os.Getenv("PULSE_AGENT_SOCKET")
	if socket == "" {
		socket = defaultSocket
	}
	if err := ensureAgent(ctx, bin, socket); err != nil {
		return err
	}
	client := &http.Client{Timeout: timeout, Transport: &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://pulse-agent"+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call Pulse Agent: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Pulse Agent returned HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func ensureAgent(ctx context.Context, bin, socket string) error {
	agentMu.Lock()
	defer agentMu.Unlock()
	if conn, err := net.DialTimeout("unix", socket, 250*time.Millisecond); err == nil {
		_ = conn.Close()
		return nil
	}
	if agentCmd != nil && agentCmd.Process != nil {
		return errors.New("Pulse Agent is starting; retry shortly")
	}
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("Pulse Agent not found at %s", bin)
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o750); err != nil {
		return err
	}
	_ = os.Remove(socket)
	cmd := exec.Command(bin) // #nosec G204 -- operator-configured executable; never sourced from a request.
	cmd.Env = append(os.Environ(), "PULSE_AGENT_SOCKET="+socket)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start Pulse Agent: %w", err)
	}
	agentCmd = cmd
	go func() {
		_ = cmd.Wait()
		agentMu.Lock()
		if agentCmd == cmd {
			agentCmd = nil
			_ = os.Remove(socket)
		}
		agentMu.Unlock()
	}()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if conn, err := net.DialTimeout("unix", socket, 100*time.Millisecond); err == nil {
			_ = conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return errors.New("Pulse Agent did not start its local socket")
}
