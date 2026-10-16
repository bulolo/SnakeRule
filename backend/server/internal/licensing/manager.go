package licensing

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/szplanner/snakerule/internal/pulseagent"
	"github.com/szplanner/snakerule/internal/repo"
)

const productID = "snakerule"

var canonicalUUIDv4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type Status struct {
	Configured     bool           `json:"configured"`
	Edition        string         `json:"edition"`
	IsLicensed     bool           `json:"is_licensed"`
	Customer       string         `json:"customer,omitempty"`
	LicenseType    string         `json:"license_type,omitempty"`
	ExpiresAt      string         `json:"expires_at,omitempty"`
	Features       map[string]any `json:"features,omitempty"`
	FeatureKeys    []string       `json:"feature_keys,omitempty"`
	InstallationID string         `json:"installation_id"`
	Error          string         `json:"error,omitempty"`
}

type diskState struct {
	InstallationID string `json:"installation_id"`
	LicenseCipher  string `json:"license_cipher,omitempty"`
}

type Evaluator func(context.Context, string, string, string, string, time.Duration) (*pulseagent.LicenseSnapshot, error)

type Manager struct {
	mu       sync.RWMutex
	opMu     sync.Mutex
	key      []byte
	state    diskState
	snapshot *pulseagent.LicenseSnapshot
	errText  string
	evaluate Evaluator
}

func New(encryptionKey string) (*Manager, error) {
	m := &Manager{evaluate: pulseagent.Evaluate}
	if encryptionKey != "" {
		decoded, err := hex.DecodeString(strings.TrimSpace(encryptionKey))
		if err != nil || len(decoded) != 32 {
			return nil, errors.New("LICENSE_ENCRYPTION_KEY must be 64 hexadecimal characters")
		}
		m.key = decoded
	}
	if err := repo.LoadEntity("license", &m.state); err != nil {
		return nil, err
	}
	if !canonicalUUIDv4.MatchString(m.state.InstallationID) {
		var id [16]byte
		if _, err := rand.Read(id[:]); err != nil {
			return nil, err
		}
		id[6] = id[6]&0x0f | 0x40
		id[8] = id[8]&0x3f | 0x80
		m.state.InstallationID = formatUUID(id[:])
		if err := m.persist(m.state); err != nil {
			return nil, err
		}
	}
	if m.state.LicenseCipher != "" {
		if len(m.key) == 0 {
			return nil, errors.New("LICENSE_ENCRYPTION_KEY is required to load the saved license")
		}
		key, err := m.decrypt(m.state.LicenseCipher)
		if err != nil {
			return nil, fmt.Errorf("decrypt saved license: %w", err)
		}
		m.refresh(context.Background(), key)
	}
	return m, nil
}

func formatUUID(id []byte) string {
	encoded := hex.EncodeToString(id)
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:]
}

func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := Status{Edition: "community", InstallationID: m.state.InstallationID, Configured: m.state.LicenseCipher != ""}
	if m.snapshot != nil {
		s.Edition, s.IsLicensed = m.snapshot.Edition, m.snapshot.IsLicensed
		s.Customer, s.LicenseType, s.ExpiresAt = m.snapshot.Customer, m.snapshot.LicenseType, m.snapshot.ExpiresAt
		s.Features, s.FeatureKeys = m.snapshot.Features, m.snapshot.FeatureKeys
		if s.IsLicensed && s.ExpiresAt != "" {
			expires, err := time.Parse(time.RFC3339, s.ExpiresAt)
			if err != nil || !expires.After(time.Now()) {
				s.IsLicensed = false
				s.Error = "授权已过期"
			}
		}
	}
	if m.errText != "" {
		s.Error = m.errText
	}
	return s
}

// TelemetryCredentials returns the active license and deployment ID only to the
// internal Pulse heartbeat sender. The API never exposes the license token.
func (m *Manager) TelemetryCredentials() (string, string, bool, error) {
	status := m.GetStatus()
	if !status.IsLicensed {
		return "", "", false, nil
	}
	m.mu.RLock()
	state := m.state
	m.mu.RUnlock()
	if state.LicenseCipher == "" {
		return "", "", false, nil
	}
	key, err := m.decrypt(state.LicenseCipher)
	if err != nil {
		return "", "", false, err
	}
	return key, state.InstallationID, true, nil
}

func (m *Manager) Upload(ctx context.Context, key string) error {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("授权文件为空")
	}
	if len(m.key) == 0 {
		return errors.New("未配置 LICENSE_ENCRYPTION_KEY，不能安全保存授权")
	}
	snapshot, err := m.evaluate(ctx, productID, "enterprise", key, m.state.InstallationID, 10*time.Second)
	if err != nil {
		return fmt.Errorf("Pulse 授权校验失败：%w", err)
	}
	if snapshot == nil || !snapshot.IsLicensed {
		if snapshot != nil && snapshot.Error != "" {
			return fmt.Errorf("Pulse 拒绝该授权：%s", snapshot.Error)
		}
		return errors.New("Pulse 未确认该授权有效")
	}
	ciphertext, err := m.encrypt(key)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	next := m.state
	next.LicenseCipher = ciphertext
	if err := m.persist(next); err != nil {
		return err
	}
	m.state, m.snapshot, m.errText = next, snapshot, ""
	return nil
}

func (m *Manager) Clear() error {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	next := m.state
	next.LicenseCipher = ""
	if err := m.persist(next); err != nil {
		return err
	}
	m.state, m.snapshot, m.errText = next, nil, ""
	return nil
}

func (m *Manager) refresh(ctx context.Context, key string) {
	snapshot, err := m.evaluate(ctx, productID, "enterprise", key, m.state.InstallationID, 10*time.Second)
	m.snapshot = snapshot
	if err != nil {
		m.errText = "Pulse 授权服务暂不可用，请刷新重试"
		return
	}
	if snapshot == nil || !snapshot.IsLicensed {
		m.errText = "授权无效或已过期"
	}
}

func (m *Manager) persist(state diskState) error {
	return repo.SaveEntity("license", state)
}

func (m *Manager) encrypt(value string) (string, error) {
	h := sha256.Sum256(m.key)
	block, err := aes.NewCipher(h[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), []byte(productID))
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (m *Manager) decrypt(value string) (string, error) {
	h := sha256.Sum256(m.key)
	block, err := aes.NewCipher(h[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	sealed, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	if len(sealed) < gcm.NonceSize() {
		return "", errors.New("saved license ciphertext is truncated")
	}
	plain, err := gcm.Open(nil, sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():], []byte(productID))
	return string(plain), err
}
