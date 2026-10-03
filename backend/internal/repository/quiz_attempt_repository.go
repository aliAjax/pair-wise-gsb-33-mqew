package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// QuizAttemptRepository handles persistence of quiz attempts.
type QuizAttemptRepository struct {
	db *gorm.DB
}

// NewQuizAttemptRepository creates a QuizAttemptRepository.
func NewQuizAttemptRepository(db *gorm.DB) *QuizAttemptRepository {
	return &QuizAttemptRepository{db: db}
}

// Create inserts an attempt. A unique key violation on
// (user_id, quiz_set_id, attempt_no) is reported as ErrDuplicate so the
// service can fall back to the attempt that won the race.
func (r *QuizAttemptRepository) Create(m *model.QuizAttempt) error {
	if err := r.db.Create(m).Error; err != nil {
		if isDuplicate(err) || strings.Contains(err.Error(), "uk_quiz_attempt_user_set_no") {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

// FindByID locates an attempt by id.
func (r *QuizAttemptRepository) FindByID(id uint) (*model.QuizAttempt, error) {
	var m model.QuizAttempt
	if err := r.db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// FindByIDForUpdate locates an attempt by id and locks the row, so
// concurrent submissions of the same attempt are serialized.
func (r *QuizAttemptRepository) FindByIDForUpdate(tx *gorm.DB, id uint) (*model.QuizAttempt, error) {
	var m model.QuizAttempt
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// FindInProgress returns the in-progress attempt of a user for a quiz set.
func (r *QuizAttemptRepository) FindInProgress(userID, setID uint) (*model.QuizAttempt, error) {
	var m model.QuizAttempt
	if err := r.db.Where("user_id = ? AND quiz_set_id = ? AND status = ?", userID, setID, model.QuizAttemptInProgress).
		Order("id DESC").First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// FindBestSubmitted returns the best submitted attempt of a user for a set.
func (r *QuizAttemptRepository) FindBestSubmitted(userID, setID uint) (*model.QuizAttempt, error) {
	var m model.QuizAttempt
	if err := r.db.Where("user_id = ? AND quiz_set_id = ? AND status = ?", userID, setID, model.QuizAttemptSubmitted).
		Order("score DESC, submitted_at DESC").First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// MaxAttemptNo returns the highest attempt number of a user for a quiz set.
func (r *QuizAttemptRepository) MaxAttemptNo(userID, setID uint) (int, error) {
	var maxNo *int
	if err := r.db.Model(&model.QuizAttempt{}).
		Where("user_id = ? AND quiz_set_id = ?", userID, setID).
		Select("MAX(attempt_no)").Scan(&maxNo).Error; err != nil {
		return 0, err
	}
	if maxNo == nil {
		return 0, nil
	}
	return *maxNo, nil
}

// UpdateTx persists an attempt inside a transaction.
func (r *QuizAttemptRepository) UpdateTx(tx *gorm.DB, m *model.QuizAttempt) error {
	return tx.Save(m).Error
}
