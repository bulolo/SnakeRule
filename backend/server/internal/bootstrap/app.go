package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/szplanner/snakerule/internal/config"
	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
	"github.com/szplanner/snakerule/internal/service"
	httpapi "github.com/szplanner/snakerule/internal/transport/http"
)

type App struct {
	Config *config.Config
	Log    *slog.Logger
	Server *http.Server
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	logger := newLogger(cfg.Log.Level, cfg.Log.Format)
	store := repo.NewMemoryRulesetRepository()
	rulesets := service.NewRulesets(store)
	engine := service.NewEngine(store)
	if err := rulesets.Save(context.Background(), demoRuleset()); err != nil {
		return nil, fmt.Errorf("seed demo ruleset: %w", err)
	}
	handler := httpapi.New(rulesets, engine, logger)
	server := &http.Server{Addr: ":" + cfg.Server.Port, Handler: handler.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	return &App{Config: cfg, Log: logger, Server: server}, nil
}

func demoRuleset() domain.Ruleset {
	return domain.Ruleset{
		ID: "order-risk", Name: "订单风险规则", Version: "1.0.0", Description: "演示 all 条件组和 facts 路径", Enabled: true,
		Rules: []domain.Rule{
			{ID: "large-new-order", Name: "大额新客订单", Priority: 100, Enabled: true,
				Conditions: domain.Condition{All: []domain.Condition{
					{Fact: "order.amount", Operator: "greaterThan", Value: 10000},
					{Fact: "customer.isNew", Operator: "equal", Value: true},
				}},
				Actions: []domain.Action{{Type: "tag", Config: map[string]any{"label": "MANUAL_REVIEW"}}},
			},
			{ID: "vip-order", Name: "VIP 订单标记", Priority: 50, Enabled: true,
				Conditions: domain.Condition{Fact: "customer.level", Operator: "equal", Value: "VIP"},
				Actions: []domain.Action{{Type: "tag", Config: map[string]any{"label": "VIP"}}},
			},
		},
	}
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
