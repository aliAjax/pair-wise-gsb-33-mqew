package repository

import (
	"gorm.io/gorm"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// QuizQuestionRepository handles persistence of quiz bank questions.
type QuizQuestionRepository struct {
	db *gorm.DB
}

// NewQuizQuestionRepository creates a QuizQuestionRepository.
func NewQuizQuestionRepository(db *gorm.DB) *QuizQuestionRepository {
	return &QuizQuestionRepository{db: db}
}

// BatchCreateTx inserts questions inside a transaction.
func (r *QuizQuestionRepository) BatchCreateTx(tx *gorm.DB, items []model.QuizQuestion) error {
	if len(items) == 0 {
		return nil
	}
	return tx.Create(&items).Error
}

// ListBySetVersion returns questions of a quiz set at a given version.
func (r *QuizQuestionRepository) ListBySetVersion(setID uint, version int) ([]model.QuizQuestion, error) {
	var items []model.QuizQuestion
	if err := r.db.Where("quiz_set_id = ? AND version = ?", setID, version).
		Order("sort ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CountBySetVersion counts questions of a quiz set at a given version.
func (r *QuizQuestionRepository) CountBySetVersion(setID uint, version int) (int64, error) {
	var count int64
	if err := r.db.Model(&model.QuizQuestion{}).
		Where("quiz_set_id = ? AND version = ?", setID, version).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
