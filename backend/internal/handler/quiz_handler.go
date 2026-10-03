package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
	"github.com/gbplantwiki/gbplantwiki/internal/service"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// QuizHandler exposes quiz set, attempt and review endpoints.
type QuizHandler struct {
	svc    *service.QuizService
	logger *slog.Logger
}

// NewQuizHandler creates a QuizHandler.
func NewQuizHandler(svc *service.QuizService, logger *slog.Logger) *QuizHandler {
	return &QuizHandler{svc: svc, logger: logger}
}

// ListSets handles GET /quiz/sets.
func (h *QuizHandler) ListSets(c *gin.Context) {
	items, err := h.svc.ListSets(middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(items))
}

// StartAttempt handles POST /quiz/sets/:id/attempts.
func (h *QuizHandler) StartAttempt(c *gin.Context) {
	setID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid quiz set id"))
		return
	}
	detail, err := h.svc.StartAttempt(middleware.GetUserID(c), uint(setID))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(detail))
}

// GetAttempt handles GET /quiz/attempts/:id.
func (h *QuizHandler) GetAttempt(c *gin.Context) {
	attemptID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid quiz attempt id"))
		return
	}
	detail, err := h.svc.GetAttempt(middleware.GetUserID(c), uint(attemptID))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(detail))
}

// Submit handles POST /quiz/attempts/:id/submit. The attempt number makes the
// submission idempotent: retries after a write failure never double-count.
func (h *QuizHandler) Submit(c *gin.Context) {
	attemptID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid quiz attempt id"))
		return
	}
	var req dto.QuizSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	detail, err := h.svc.Submit(middleware.GetUserID(c), uint(attemptID), req.Answers)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"message": constants.MsgQuizSubmitted, "attempt": detail}))
}

// ListReview handles GET /quiz/review.
func (h *QuizHandler) ListReview(c *gin.Context) {
	items, err := h.svc.ListReview(middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(items))
}

// AnswerReview handles POST /quiz/review/answer.
func (h *QuizHandler) AnswerReview(c *gin.Context) {
	var req dto.QuizReviewAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	result, err := h.svc.AnswerReview(middleware.GetUserID(c), req.QuestionID, req.Selected)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(result))
}

// ReviewProgress handles GET /quiz/review/progress. The review page and the
// home page both read this single result.
func (h *QuizHandler) ReviewProgress(c *gin.Context) {
	progress, err := h.svc.ReviewProgress(middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(progress))
}
