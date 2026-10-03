package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// QuizWrongQuestionRepository handles persistence of the review list.
type QuizWrongQuestionRepository struct {
	db *gorm.DB
}

// NewQuizWrongQuestionRepository creates a QuizWrongQuestionRepository.
func NewQuizWrongQuestionRepository(db *gorm.DB) *QuizWrongQuestionRepository {
	return &QuizWrongQuestionRepository{db: db}
}

// FindByUserQuestionForUpdate locates a review entry and locks the row.
func (r *QuizWrongQuestionRepository) FindByUserQuestionForUpdate(tx *gorm.DB, userID, questionID uint) (*model.QuizWrongQuestion, error) {
	var m model.QuizWrongQuestion
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND question_id = ?", userID, questionID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// CreateTx inserts a review entry inside a transaction.
func (r *QuizWrongQuestionRepository) CreateTx(tx *gorm.DB, m *model.QuizWrongQuestion) error {
	return tx.Create(m).Error
}

// UpdateTx persists a review entry inside a transaction.
func (r *QuizWrongQuestionRepository) UpdateTx(tx *gorm.DB, m *model.QuizWrongQuestion) error {
	return tx.Save(m).Error
}

// ListPendingByUser returns pending review entries of a user, newest first.
func (r *QuizWrongQuestionRepository) ListPendingByUser(userID uint) ([]model.QuizWrongQuestion, error) {
	var items []model.QuizWrongQuestion
	if err := r.db.Where("user_id = ? AND status = ?", userID, model.WrongQuestionPending).
		Order("updated_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CountByStatus counts review entries of a user in a given status.
func (r *QuizWrongQuestionRepository) CountByStatus(userID uint, status string) (int64, error) {
	var count int64
	if err := r.db.Model(&model.QuizWrongQuestion{}).
		Where("user_id = ? AND status = ?", userID, status).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
