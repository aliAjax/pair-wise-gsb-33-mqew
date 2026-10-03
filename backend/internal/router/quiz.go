package router

import (
	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/config"
	"github.com/gbplantwiki/gbplantwiki/internal/handler"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
)

func registerQuizRoutes(v1 *gin.RouterGroup, cfg *config.Config, h *handler.QuizHandler, limiter *middleware.RateLimiter) {
	quiz := v1.Group("/quiz", middleware.AuthRequired(cfg))

	// 普通用户：题集浏览、作答、错题复习、进度
	quiz.GET("/sets", h.ListSets)
	quiz.GET("/progress", h.Progress)
	quiz.GET("/review", h.ListReview)
	quiz.POST("/review/answer", h.AnswerReview)
	quiz.POST("/attempts", h.StartAttempt)
	quiz.GET("/attempts/:attemptNo", h.GetAttempt)
	quiz.POST("/attempts/submit", h.SubmitAttempt)

	// 管理员：题库维护（新增/修改题目后自动级联发布新版本）
	admin := quiz.Group("/admin", middleware.RequireRole("admin"))
	admin.GET("/questions", h.ListQuestions)
	admin.POST("/questions", limiter.Limit(), h.CreateQuestion)
	admin.PUT("/questions/:id", h.UpdateQuestion)
}
