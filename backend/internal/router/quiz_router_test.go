package router

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gbplantwiki/gbplantwiki/internal/config"
)

func TestSetupRegistersQuizRoutes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:routes_setup?mode=memory"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	cfg := &config.Config{
		JWTSecret:    "test-secret-test-secret-test-secret",
		RateLimitReq: 100,
		RateLimitWin: time.Minute,
	}
	r := Setup(cfg, db, slog.New(slog.NewTextHandler(io.Discard, nil)))

	want := map[string]bool{
		"GET-/api/v1/quiz/sets":                false,
		"GET-/api/v1/quiz/progress":            false,
		"GET-/api/v1/quiz/review":              false,
		"POST-/api/v1/quiz/review/answer":      false,
		"POST-/api/v1/quiz/attempts":           false,
		"GET-/api/v1/quiz/attempts/:attemptNo": false,
		"POST-/api/v1/quiz/attempts/submit":    false,
		"GET-/api/v1/quiz/admin/questions":     false,
		"POST-/api/v1/quiz/admin/questions":    false,
		"PUT-/api/v1/quiz/admin/questions/:id": false,
	}
	for _, ri := range r.Routes() {
		key := ri.Method + "-" + ri.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("quiz 路由未注册: %s", route)
		}
	}
}
