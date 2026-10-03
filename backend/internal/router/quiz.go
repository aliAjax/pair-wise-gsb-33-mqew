package router

import (
	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/config"
	"github.com/gbplantwiki/gbplantwiki/internal/handler"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
)

func registerQuizRoutes(v1 *gin.RouterGroup, cfg *config.Config, qh *handler.QuizHandler, ah *handler.QuizAdminHandler, limiter *middleware.RateLimiter) {
	quiz := v1.Group("/quiz", middleware.AuthRequired(cfg))
	quiz.GET("/sets", qh.ListSets)
	quiz.POST("/sets/:id/attempts", limiter.Limit(), qh.StartAttempt)
	quiz.GET("/attempts/:id", qh.GetAttempt)
	quiz.POST("/attempts/:id/submit", limiter.Limit(), qh.Submit)
	quiz.GET("/review", qh.ListReview)
	quiz.GET("/review/progress", qh.ReviewProgress)
	quiz.POST("/review/answer", qh.AnswerReview)
	admin := quiz.Group("/admin", middleware.RequireRole("admin"))
	admin.POST("/sets", ah.CreateSet)
	admin.PUT("/sets/:id/questions", ah.ReplaceQuestions)
}
