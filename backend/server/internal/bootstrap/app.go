package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/szplanner/snakerule/internal/config"
	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/licensing"
	"github.com/szplanner/snakerule/internal/repo"
	"github.com/szplanner/snakerule/internal/service"
	"github.com/szplanner/snakerule/internal/telemetry"
	httpapi "github.com/szplanner/snakerule/internal/transport/http"
)

type App struct {
	Config         *config.Config
	Log            *slog.Logger
	Server         *http.Server
	stopBackground context.CancelFunc
}

func (a *App) StopBackground() {
	if a.stopBackground != nil {
		a.stopBackground()
	}
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	logger := newLogger(cfg.Log.Level, cfg.Log.Format)
	licenseManager, err := licensing.New(cfg.License.StorePath, cfg.License.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("initialize license manager: %w", err)
	}
	stateDir := filepath.Join(filepath.Dir(cfg.License.StorePath), "state")
	store, err := repo.NewPersistentRulesetRepository(filepath.Join(stateDir, "rulesets.json"))
	if err != nil {
		return nil, fmt.Errorf("load rulesets: %w", err)
	}
	workspaceStore, err := repo.NewPersistentWorkspaceRepository(filepath.Join(stateDir, "workspaces.json"))
	if err != nil {
		return nil, fmt.Errorf("load workspaces: %w", err)
	}
	projectStore, err := repo.NewPersistentProjectRepository(filepath.Join(stateDir, "projects.json"))
	if err != nil {
		return nil, fmt.Errorf("load projects: %w", err)
	}
	projectKeyStore, err := repo.NewPersistentProjectKeyRepository(filepath.Join(stateDir, "project-keys.json"))
	if err != nil {
		return nil, fmt.Errorf("load project keys: %w", err)
	}
	alertChannelStore, err := repo.NewPersistentAlertChannelRepository(filepath.Join(stateDir, "alert-channels.json"))
	if err != nil {
		return nil, fmt.Errorf("load alert channels: %w", err)
	}
	notificationStore, err := repo.NewPersistentNotificationRepository(filepath.Join(stateDir, "notifications.json"))
	if err != nil {
		return nil, fmt.Errorf("load notifications: %w", err)
	}
	invocationLogStore, err := repo.NewPersistentInvocationLogRepository(filepath.Join(stateDir, "invocations.json"))
	if err != nil {
		return nil, fmt.Errorf("load invocation logs: %w", err)
	}
	workspaceItems, err := workspaceStore.List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("list workspaces for project migration: %w", err)
	}
	for _, workspace := range workspaceItems {
		if err := store.DeleteProjectScope(workspace.ID, "default"); err != nil {
			return nil, fmt.Errorf("remove legacy default project rules: %w", err)
		}
		if err := projectKeyStore.DeleteProject(workspace.ID, "default"); err != nil {
			return nil, fmt.Errorf("remove legacy default project credentials: %w", err)
		}
		if err := invocationLogStore.DeleteProject(workspace.ID, "default"); err != nil {
			return nil, fmt.Errorf("remove legacy default project logs: %w", err)
		}
	}
	jwtSigningKey, err := loadOrCreateJWTSigningKey(filepath.Join(stateDir, "jwt-signing.key"))
	if err != nil {
		return nil, fmt.Errorf("load JWT signing key: %w", err)
	}
	if _, err := alertChannelStore.Get(context.Background(), "internal-inbox"); err == repo.ErrNotFound {
		if err := alertChannelStore.Save(context.Background(), domain.AlertChannel{ID: "internal-inbox", Name: "站内通知", Type: "internal", Enabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
			return nil, fmt.Errorf("seed internal notification channel: %w", err)
		}
	}
	workspaces := service.NewWorkspaces(workspaceStore)
	rulesets := service.NewRulesets(store)
	engine := service.NewEngine(store)
	handler := httpapi.New(rulesets, engine, logger, httpapi.AuthConfig{Email: cfg.Auth.Email, Name: cfg.Auth.Name, Password: cfg.Auth.Password, SessionTTL: cfg.Auth.SessionTTL, SessionStorePath: filepath.Join(stateDir, "sessions.json"), JWTSigningKey: jwtSigningKey}, licenseManager, workspaces)
	handler.SetAlertChannelRepository(alertChannelStore)
	handler.SetNotificationRepository(notificationStore)
	handler.SetInvocationLogRepository(invocationLogStore)
	handler.SetProjectRepositories(projectStore, projectKeyStore)
	server := &http.Server{Addr: ":" + cfg.Server.Port, Handler: handler.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	backgroundCtx, stopBackground := context.WithCancel(context.Background())
	telemetry.Start(backgroundCtx, telemetry.Config{
		Environment: cfg.Server.Env,
		Endpoint:    cfg.Pulse.HeartbeatURL,
		Interval:    time.Duration(cfg.Pulse.HeartbeatIntervalSecond) * time.Second,
		Timeout:     time.Duration(cfg.Pulse.HeartbeatTimeoutSecond) * time.Second,
	}, telemetry.Stores{License: licenseManager, Workspaces: workspaceStore, Projects: projectStore, Rulesets: store})
	return &App{Config: cfg, Log: logger, Server: server, stopBackground: stopBackground}, nil
}

func newLogger(level, format string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn", "warning":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: l}
	if strings.EqualFold(format, "text") {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
