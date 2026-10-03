package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/service"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// QuizAdminHandler exposes quiz bank maintenance endpoints.
type QuizAdminHandler struct {
	svc    *service.QuizAdminService
	logger *slog.Logger
}

// NewQuizAdminHandler creates a QuizAdminHandler.
func NewQuizAdminHandler(svc *service.QuizAdminService, logger *slog.Logger) *QuizAdminHandler {
	return &QuizAdminHandler{svc: svc, logger: logger}
}

// CreateSet handles POST /quiz/admin/sets.
func (h *QuizAdminHandler) CreateSet(c *gin.Context) {
	var req dto.QuizSetCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	set, err := h.svc.CreateSet(&req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(gin.H{"message": constants.MsgQuizSetCreated, "set": set}))
}

// ReplaceQuestions handles PUT /quiz/admin/sets/:id/questions. The bank
// version is bumped; unfinished and finished attempts keep their version.
func (h *QuizAdminHandler) ReplaceQuestions(c *gin.Context) {
	setID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid quiz set id"))
		return
	}
	var req dto.QuizSetQuestionsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	set, err := h.svc.ReplaceQuestions(uint(setID), req.Questions)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.Error(util.NewAppError(http.StatusNotFound, constants.CodeNotFound,
				"QuizSet[id="+c.Param("id")+"] not found for bank update"))
			return
		}
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"message": constants.MsgQuizBankUpdated, "set": set}))
}
