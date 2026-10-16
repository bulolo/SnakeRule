package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

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
	database       *sql.DB
}

func (a *App) StopBackground() {
	if a.stopBackground != nil {
		a.stopBackground()
	}
	if a.database != nil {
		_ = a.database.Close()
	}
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	logger := newLogger(cfg.Log.Level, cfg.Log.Format)
	db, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect PostgreSQL: %w", err)
	}
	if err := repo.OpenDatabase(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize PostgreSQL storage: %w", err)
	}
	repo.SetDatabase(db)
	licenseManager, err := licensing.New(cfg.License.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("initialize license manager: %w", err)
	}
	store, err := repo.NewPersistentRulesetRepository("rulesets")
	if err != nil {
		return nil, fmt.Errorf("load rulesets: %w", err)
	}
	organizationStore, err := repo.NewPersistentOrganizationRepository("organizations")
	if err != nil {
		return nil, fmt.Errorf("load organizations: %w", err)
	}
	projectStore, err := repo.NewPersistentProjectRepository("projects")
	if err != nil {
		return nil, fmt.Errorf("load projects: %w", err)
	}
	projectKeyStore, err := repo.NewPersistentProjectKeyRepository("project-keys")
	if err != nil {
		return nil, fmt.Errorf("load project keys: %w", err)
	}
	userStore, err := repo.NewPersistentUserRepository("users")
	if err != nil {
		return nil, fmt.Errorf("load users: %w", err)
	}
	if err = userStore.SeedSuperAdmin(cfg.Auth.Email, cfg.Auth.Name, cfg.Auth.Password); err != nil {
		return nil, fmt.Errorf("seed superadmin: %w", err)
	}
	if err = userStore.SeedDefaultOrganizationAdmin(); err != nil {
		return nil, fmt.Errorf("seed default organization admin: %w", err)
	}
	alertChannelStore, err := repo.NewPersistentAlertChannelRepository("alert-channels")
	if err != nil {
		return nil, fmt.Errorf("load alert channels: %w", err)
	}
	platformSettingsStore, err := repo.NewPersistentPlatformSettingsRepository("platform-settings")
	if err != nil {
		return nil, fmt.Errorf("load platform settings: %w", err)
	}
	notificationStore, err := repo.NewPersistentNotificationRepository("notifications")
	if err != nil {
		return nil, fmt.Errorf("load notifications: %w", err)
	}
	invocationLogStore, err := repo.NewPersistentInvocationLogRepository("invocations")
	if err != nil {
		return nil, fmt.Errorf("load invocation logs: %w", err)
	}
	jwtSigningKey, err := loadOrCreateJWTSigningKey()
	if err != nil {
		return nil, fmt.Errorf("load JWT signing key: %w", err)
	}
	if _, err := alertChannelStore.Get(context.Background(), "internal-inbox"); err == repo.ErrNotFound {
		if err := alertChannelStore.Save(context.Background(), domain.AlertChannel{ID: "internal-inbox", Name: "站内通知", Type: "internal", Enabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
			return nil, fmt.Errorf("seed internal notification channel: %w", err)
		}
	}
	organizations := service.NewOrganizationService(organizationStore)
	rulesets := service.NewRulesetService(store)
	engine := service.NewEngine(store)
	handler := httpapi.NewServer(rulesets, engine, logger, httpapi.AuthConfig{Email: cfg.Auth.Email, Name: cfg.Auth.Name, Password: cfg.Auth.Password, SessionTTL: cfg.Auth.SessionTTL, SessionStoreKey: "sessions", JWTSigningKey: jwtSigningKey}, licenseManager, organizations)
	handler.SetAlertChannelRepository(alertChannelStore)
	handler.SetRuntimeInfo(db, cfg.Server.Env)
	handler.SetPlatformSettingsRepository(platformSettingsStore)
	handler.SetNotificationRepository(notificationStore)
	handler.SetInvocationLogRepository(invocationLogStore)
	handler.SetProjectRepositories(projectStore, projectKeyStore)
	handler.SetUserRepository(userStore)
	server := &http.Server{Addr: ":" + cfg.Server.Port, Handler: handler.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	backgroundCtx, stopBackground := context.WithCancel(context.Background())
	startInvocationLogCleanup(backgroundCtx, invocationLogStore, cfg.Invocations.RetentionDays, cfg.Invocations.CleanupIntervalSecs, cfg.Invocations.CleanupBatchSize, logger)
	telemetry.Start(backgroundCtx, telemetry.Config{
		Environment: cfg.Server.Env,
		Endpoint:    cfg.Pulse.HeartbeatURL,
		Interval:    time.Duration(cfg.Pulse.HeartbeatIntervalSecond) * time.Second,
		Timeout:     time.Duration(cfg.Pulse.HeartbeatTimeoutSecond) * time.Second,
	}, telemetry.Stores{License: licenseManager, Organizations: organizationStore, Projects: projectStore, Rulesets: store})
	return &App{Config: cfg, Log: logger, Server: server, stopBackground: stopBackground, database: db}, nil
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
