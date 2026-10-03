package router

import (
	"testing"

	"github.com/gbplantwiki/gbplantwiki/internal/config"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// TestSetupRegistersRoutes ensures the whole route tree (including the quiz
// routes) registers without gin wildcard conflicts.
func TestSetupRegistersRoutes(t *testing.T) {
	cfg := config.Load()
	r := Setup(cfg, nil, util.NewLogger())
	want := map[string]bool{
		"GET /api/v1/quiz/sets":                     false,
		"POST /api/v1/quiz/sets/:id/attempts":       false,
		"GET /api/v1/quiz/attempts/:id":             false,
		"POST /api/v1/quiz/attempts/:id/submit":     false,
		"GET /api/v1/quiz/review":                   false,
		"GET /api/v1/quiz/review/progress":          false,
		"POST /api/v1/quiz/review/answer":           false,
		"POST /api/v1/quiz/admin/sets":              false,
		"PUT /api/v1/quiz/admin/sets/:id/questions": false,
	}
	for _, route := range r.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	if len(want) > 0 {
		t.Errorf("missing routes: %v", want)
	}
}
