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

// QuizHandler 暴露知识点题集接口。
type QuizHandler struct {
	svc    *service.QuizService
	logger *slog.Logger
}

// NewQuizHandler creates a QuizHandler.
func NewQuizHandler(svc *service.QuizService, logger *slog.Logger) *QuizHandler {
	return &QuizHandler{svc: svc, logger: logger}
}

// ListSets 处理 GET /quiz/sets（登录）。
func (h *QuizHandler) ListSets(c *gin.Context) {
	items, err := h.svc.ListSets()
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(items))
}

// StartAttempt 处理 POST /quiz/attempts（登录）。
func (h *QuizHandler) StartAttempt(c *gin.Context) {
	var req dto.StartAttemptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	view, err := h.svc.StartAttempt(middleware.GetUserID(c), &req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(view))
}

// GetAttempt 处理 GET /quiz/attempts/:attemptNo（登录）。
func (h *QuizHandler) GetAttempt(c *gin.Context) {
	view, err := h.svc.GetAttempt(middleware.GetUserID(c), c.Param("attemptNo"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(view))
}

// SubmitAttempt 处理 POST /quiz/attempts/submit（登录）。
// 同一尝试号只认首次交卷结果；写失败按同一尝试号重试。
func (h *QuizHandler) SubmitAttempt(c *gin.Context) {
	var req dto.SubmitAttemptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	result, err := h.svc.SubmitAttempt(middleware.GetUserID(c), &req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(result))
}

// ListReview 处理 GET /quiz/review（登录）。
func (h *QuizHandler) ListReview(c *gin.Context) {
	items, err := h.svc.ListReview(middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(items))
}

// AnswerReview 处理 POST /quiz/review/answer（登录）。
// answer_no 为幂等键，重试不重复累计连对次数。
func (h *QuizHandler) AnswerReview(c *gin.Context) {
	var req dto.ReviewAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	result, err := h.svc.AnswerReview(middleware.GetUserID(c), &req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(result))
}

// Progress 处理 GET /quiz/progress（登录）。复习页与首页进度读取同一结果。
func (h *QuizHandler) Progress(c *gin.Context) {
	progress, err := h.svc.Progress(middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(progress))
}

// ---- 管理员题库维护 ----

// ListQuestions 处理 GET /quiz/admin/questions（管理员）。
func (h *QuizHandler) ListQuestions(c *gin.Context) {
	items, err := h.svc.ListQuestions(c.Query("category"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(items))
}

// CreateQuestion 处理 POST /quiz/admin/questions（管理员，限流）。
func (h *QuizHandler) CreateQuestion(c *gin.Context) {
	var req dto.QuizQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	item, err := h.svc.CreateQuestion(&req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(item))
}

// UpdateQuestion 处理 PUT /quiz/admin/questions/:id（管理员）。
func (h *QuizHandler) UpdateQuestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid question id"))
		return
	}
	var req dto.QuizQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	item, err := h.svc.UpdateQuestion(uint(id), &req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(item))
}
