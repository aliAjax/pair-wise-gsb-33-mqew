package service

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// QuizAdminService maintains quiz sets and versioned question banks.
type QuizAdminService struct {
	db           *gorm.DB
	setRepo      *repository.QuizSetRepository
	questionRepo *repository.QuizQuestionRepository
	logger       *slog.Logger
}

// NewQuizAdminService creates a QuizAdminService.
func NewQuizAdminService(db *gorm.DB, setRepo *repository.QuizSetRepository, questionRepo *repository.QuizQuestionRepository, logger *slog.Logger) *QuizAdminService {
	return &QuizAdminService{db: db, setRepo: setRepo, questionRepo: questionRepo, logger: logger}
}

// CreateSet creates a quiz set with its first bank version.
func (s *QuizAdminService) CreateSet(req *dto.QuizSetCreateRequest) (*model.QuizSet, error) {
	set := &model.QuizSet{
		Title: req.Title, Category: req.Category, Description: req.Description,
		Version: 1, Status: model.QuizSetStatusPublished,
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.setRepo.CreateTx(tx, set); err != nil {
			return fmt.Errorf("quiz set create: %w", err)
		}
		questions, err := buildQuestionRows(set.ID, 1, req.Questions)
		if err != nil {
			return err
		}
		if err := s.questionRepo.BatchCreateTx(tx, questions); err != nil {
			return fmt.Errorf("quiz set create questions: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(fmt.Sprintf(constants.LogQuizSetCreated, set.Title, set.Category), "id", set.ID)
	return set, nil
}

// ReplaceQuestions bumps the bank version and inserts the new questions. Old
// version rows stay untouched so in-progress and submitted attempts keep
// viewing their original snapshot; only sets not yet started pick up the new
// version.
func (s *QuizAdminService) ReplaceQuestions(setID uint, inputs []dto.QuizQuestionInput) (*model.QuizSet, error) {
	var set *model.QuizSet
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		set, err = s.setRepo.FindByIDForUpdate(tx, setID)
		if err != nil {
			return err
		}
		newVersion := set.Version + 1
		questions, err := buildQuestionRows(setID, newVersion, inputs)
		if err != nil {
			return err
		}
		if err := s.questionRepo.BatchCreateTx(tx, questions); err != nil {
			return fmt.Errorf("quiz bank replace questions: %w", err)
		}
		set.Version = newVersion
		if err := s.setRepo.UpdateTx(tx, set); err != nil {
			return fmt.Errorf("quiz bank replace version: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(fmt.Sprintf(constants.LogQuizBankUpdated, setID, set.Version))
	return set, nil
}

// buildQuestionRows converts question inputs into bank rows of one version.
func buildQuestionRows(setID uint, version int, inputs []dto.QuizQuestionInput) ([]model.QuizQuestion, error) {
	rows := make([]model.QuizQuestion, 0, len(inputs))
	for i, in := range inputs {
		if in.Answer >= len(in.Options) {
			return nil, util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("QuizQuestion[sort=%d] invalid: answer index %d out of %d options", i, in.Answer, len(in.Options)))
		}
		optionsJSON, err := json.Marshal(in.Options)
		if err != nil {
			return nil, fmt.Errorf("quiz question options encode: %w", err)
		}
		rows = append(rows, model.QuizQuestion{
			QuizSetID: setID, Version: version, Question: in.Question,
			Options: string(optionsJSON), Answer: in.Answer,
			Explanation: in.Explanation, Sort: i,
		})
	}
	return rows, nil
}
