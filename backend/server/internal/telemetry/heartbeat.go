package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/szplanner/snakerule/internal/licensing"
	"github.com/szplanner/snakerule/internal/pulseagent"
	"github.com/szplanner/snakerule/internal/repo"
	appversion "github.com/szplanner/snakerule/internal/version"
)

type Config struct {
	Environment string
	Endpoint    string
	Interval    time.Duration
	Timeout     time.Duration
}

type Stores struct {
	License       *licensing.Manager
	Organizations repo.OrganizationRepository
	Projects      repo.ProjectRepository
	Rulesets      repo.RulesetRepository
}

func Start(ctx context.Context, cfg Config, stores Stores) {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://pulse.szplanner.com/api/v1/snakerule/v1/telemetry/heartbeat"
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 4 * time.Minute
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	go run(ctx, cfg, stores)
}

func run(ctx context.Context, cfg Config, stores Stores) {
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := send(ctx, cfg, stores); err != nil {
				slog.Debug("Pulse heartbeat failed", "error", err)
			}
			timer.Reset(cfg.Interval)
		}
	}
}

func send(parent context.Context, cfg Config, stores Stores) error {
	if stores.License == nil {
		return nil
	}
	licenseKey, installationID, licensed, err := stores.License.TelemetryCredentials()
	if err != nil {
		return fmt.Errorf("load Pulse telemetry credentials: %w", err)
	}
	if !licensed {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, cfg.Timeout)
	defer cancel()
	response, err := pulseagent.SubmitTelemetry(ctx, cfg.Timeout, map[string]any{
		"product":         "snakerule",
		"license_key":     licenseKey,
		"installation_id": installationID,
		"endpoint":        cfg.Endpoint,
		"timeout_seconds": int(cfg.Timeout.Seconds()),
		"telemetry":       collect(ctx, cfg, installationID, stores),
	})
	if err != nil {
		return err
	}
	if response.Status != "accepted" {
		return fmt.Errorf("Pulse Agent rejected heartbeat: %s", response.Error)
	}
	slog.Info("Pulse heartbeat accepted", "product", "snakerule")
	return nil
}

func collect(ctx context.Context, cfg Config, installationID string, stores Stores) map[string]any {
	hostname, _ := os.Hostname()
	organizations, projects, rulesets := 0, 0, 0
	if stores.Organizations != nil && stores.Projects != nil && stores.Rulesets != nil {
		items, err := stores.Organizations.List(ctx)
		if err == nil {
			organizations = len(items)
			for _, organization := range items {
				organizationCtx := repo.WithOrganization(ctx, organization.ID)
				projectItems, err := stores.Projects.List(organizationCtx)
				if err != nil {
					continue
				}
				projects += len(projectItems)
				for _, project := range projectItems {
					ruleItems, err := stores.Rulesets.List(repo.WithProject(organizationCtx, project.ID))
					if err == nil {
						rulesets += len(ruleItems)
					}
				}
			}
		}
	}
	environment := strings.ToLower(strings.TrimSpace(cfg.Environment))
	switch environment {
	case "local", "development", "staging", "production", "test":
	case "dev":
		environment = "development"
	default:
		environment = "development"
	}
	return map[string]any{
		"schema_version": 1,
		"app_version":    appversion.Version,
		"environment":    environment,
		"product":        "snakerule",
		"instance":       map[string]any{"deployment_id": installationID, "hostname": hostname},
		"runtime": map[string]any{
			"os_name": runtime.GOOS, "arch": runtime.GOARCH,
			"language": "go", "language_version": strings.TrimPrefix(runtime.Version(), "go"),
		},
		"resources": map[string]any{"cpu_cores": runtime.NumCPU()},
		"health":    map[string]any{"boot_ok": true, "license_valid": true},
		"metrics": map[string]any{
			"organizations": organizations,
			"projects":      projects,
			"rulesets":      rulesets,
		},
		"uptime_seconds": int64(time.Since(startedAt).Seconds()),
	}
}

var startedAt = time.Now()
