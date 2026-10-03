package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// QuizSetRepository handles persistence of quiz sets.
type QuizSetRepository struct {
	db *gorm.DB
}

// NewQuizSetRepository creates a QuizSetRepository.
func NewQuizSetRepository(db *gorm.DB) *QuizSetRepository {
	return &QuizSetRepository{db: db}
}

// Create inserts a quiz set.
func (r *QuizSetRepository) Create(m *model.QuizSet) error {
	return r.db.Create(m).Error
}

// CreateTx inserts a quiz set inside a transaction.
func (r *QuizSetRepository) CreateTx(tx *gorm.DB, m *model.QuizSet) error {
	return tx.Create(m).Error
}

// FindByID locates a quiz set by id.
func (r *QuizSetRepository) FindByID(id uint) (*model.QuizSet, error) {
	var m model.QuizSet
	if err := r.db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// FindByIDForUpdate locates a quiz set by id and locks the row.
func (r *QuizSetRepository) FindByIDForUpdate(tx *gorm.DB, id uint) (*model.QuizSet, error) {
	var m model.QuizSet
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// ListPublished returns all published quiz sets ordered by id.
func (r *QuizSetRepository) ListPublished() ([]model.QuizSet, error) {
	var items []model.QuizSet
	if err := r.db.Where("status = ?", model.QuizSetStatusPublished).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// UpdateTx persists a quiz set inside a transaction.
func (r *QuizSetRepository) UpdateTx(tx *gorm.DB, m *model.QuizSet) error {
	return tx.Save(m).Error
}
