package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// QuizRepository handles persistence of quiz questions, sets, versions,
// attempts and review items.
type QuizRepository struct {
	db *gorm.DB
}

// NewQuizRepository creates a QuizRepository.
func NewQuizRepository(db *gorm.DB) *QuizRepository {
	return &QuizRepository{db: db}
}

// ---- 题库题目 ----

// CreateQuestion inserts a quiz question.
func (r *QuizRepository) CreateQuestion(q *model.QuizQuestion) error {
	return r.CreateQuestionTx(r.db, q)
}

// CreateQuestionTx inserts a quiz question within a transaction.
func (r *QuizRepository) CreateQuestionTx(tx *gorm.DB, q *model.QuizQuestion) error {
	return tx.Create(q).Error
}

// UpdateQuestion persists a quiz question.
func (r *QuizRepository) UpdateQuestion(q *model.QuizQuestion) error {
	return r.UpdateQuestionTx(r.db, q)
}

// UpdateQuestionTx persists a quiz question within a transaction.
func (r *QuizRepository) UpdateQuestionTx(tx *gorm.DB, q *model.QuizQuestion) error {
	return tx.Save(q).Error
}

// FindQuestion locates a quiz question by id.
func (r *QuizRepository) FindQuestion(id uint) (*model.QuizQuestion, error) {
	return r.FindQuestionTx(r.db, id)
}

// FindQuestionTx locates a quiz question within a transaction.
func (r *QuizRepository) FindQuestionTx(tx *gorm.DB, id uint) (*model.QuizQuestion, error) {
	var q model.QuizQuestion
	if err := tx.First(&q, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &q, nil
}

// ListQuestions returns quiz questions optionally filtered by category.
func (r *QuizRepository) ListQuestions(category string) ([]model.QuizQuestion, error) {
	return r.ListQuestionsTx(r.db, category)
}

// ListQuestionsTx returns quiz questions within a transaction.
func (r *QuizRepository) ListQuestionsTx(tx *gorm.DB, category string) ([]model.QuizQuestion, error) {
	var items []model.QuizQuestion
	q := tx.Model(&model.QuizQuestion{})
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if err := q.Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListQuestionsByID returns questions matching the given ids.
func (r *QuizRepository) ListQuestionsByID(ids []uint) ([]model.QuizQuestion, error) {
	var items []model.QuizQuestion
	if len(ids) == 0 {
		return items, nil
	}
	if err := r.db.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ---- 题集 ----

// CreateSet inserts a quiz set.
func (r *QuizRepository) CreateSet(s *model.QuizSet) error {
	return r.db.Create(s).Error
}

// ListSets returns all active quiz sets ordered by id.
func (r *QuizRepository) ListSets() ([]model.QuizSet, error) {
	var items []model.QuizSet
	if err := r.db.Where("status = ?", model.QuizSetActive).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindSet locates a quiz set by id.
func (r *QuizRepository) FindSet(id uint) (*model.QuizSet, error) {
	var s model.QuizSet
	if err := r.db.First(&s, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// FindSetByCategory locates the (single) active quiz set for a category.
func (r *QuizRepository) FindSetByCategory(category string) (*model.QuizSet, error) {
	return r.FindSetByCategoryTx(r.db, category)
}

// FindSetByCategoryTx locates the active quiz set for a category within a transaction.
func (r *QuizRepository) FindSetByCategoryTx(tx *gorm.DB, category string) (*model.QuizSet, error) {
	var s model.QuizSet
	if err := tx.Where("category = ? AND status = ?", category, model.QuizSetActive).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// ---- 版本 ----

// CreateVersionTx inserts a quiz set version within an outer transaction.
func (r *QuizRepository) CreateVersionTx(tx *gorm.DB, v *model.QuizSetVersion) error {
	return tx.Create(v).Error
}

// LatestVersion returns the highest version of a set.
func (r *QuizRepository) LatestVersion(setID uint) (*model.QuizSetVersion, error) {
	var v model.QuizSetVersion
	if err := r.db.Where("set_id = ?", setID).Order("version DESC").First(&v).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &v, nil
}

// FindVersion locates a specific set version.
func (r *QuizRepository) FindVersion(id uint) (*model.QuizSetVersion, error) {
	var v model.QuizSetVersion
	if err := r.db.First(&v, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &v, nil
}

// PublishVersionForSetTx snapshots all questions of the set's category and
// appends a new immutable version within an outer transaction.
func (r *QuizRepository) PublishVersionForSetTx(tx *gorm.DB, s *model.QuizSet, snapshot string, questionIDs string) error {
	var last model.QuizSetVersion
	nextVersion := 1
	if err := tx.Where("set_id = ?", s.ID).Order("version DESC").First(&last).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		nextVersion = last.Version + 1
	}
	v := &model.QuizSetVersion{
		SetID:       s.ID,
		Version:     nextVersion,
		QuestionIDs: questionIDs,
		Snapshot:    snapshot,
	}
	return tx.Create(v).Error
}

// ---- 尝试 ----

// FindAttemptByNo locates an attempt by user and client-generated attempt number.
func (r *QuizRepository) FindAttemptByNo(userID uint, attemptNo string) (*model.QuizAttempt, error) {
	return r.FindAttemptByNoTx(r.db, userID, attemptNo)
}

// FindAttemptByNoTx locates an attempt within a transaction.
func (r *QuizRepository) FindAttemptByNoTx(tx *gorm.DB, userID uint, attemptNo string) (*model.QuizAttempt, error) {
	var a model.QuizAttempt
	if err := tx.Where("user_id = ? AND attempt_no = ?", userID, attemptNo).First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// FindOpenAttempt returns the user's in-progress attempt for a set.
func (r *QuizRepository) FindOpenAttempt(userID, setID uint) (*model.QuizAttempt, error) {
	return r.FindOpenAttemptTx(r.db, userID, setID)
}

// FindOpenAttemptTx returns the user's in-progress attempt within a transaction.
func (r *QuizRepository) FindOpenAttemptTx(tx *gorm.DB, userID, setID uint) (*model.QuizAttempt, error) {
	var a model.QuizAttempt
	if err := tx.Where("user_id = ? AND set_id = ? AND status = ?", userID, setID, model.QuizAttemptOpen).
		Order("id DESC").First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// LockOpenAttemptTx 锁住该用户该题集的 open 尝试行（若存在），用于并发互斥。
// 即使没有行也返回 nil，调用方随后在同一事务内插入以消除竞态。
func (r *QuizRepository) LockOpenAttemptTx(tx *gorm.DB, userID, setID uint) (*model.QuizAttempt, error) {
	var a model.QuizAttempt
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND set_id = ? AND status = ?", userID, setID, model.QuizAttemptOpen).
		Order("id DESC").First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// CreateAttemptTx inserts an attempt within an outer transaction.
func (r *QuizRepository) CreateAttemptTx(tx *gorm.DB, a *model.QuizAttempt) error {
	return tx.Create(a).Error
}

// SaveAttempt persists an attempt (used for status/score transitions).
func (r *QuizRepository) SaveAttempt(a *model.QuizAttempt) error {
	return r.db.Save(a).Error
}

// CountAttemptAnswers returns how many answer rows an attempt currently has.
func (r *QuizRepository) CountAttemptAnswers(attemptID uint) (int64, error) {
	var n int64
	if err := r.db.Model(&model.QuizAttemptAnswer{}).Where("attempt_id = ?", attemptID).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// CreateAttemptAnswersTx inserts the graded answer rows of an attempt.
func (r *QuizRepository) CreateAttemptAnswersTx(tx *gorm.DB, rows []model.QuizAttemptAnswer) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// ListAttemptAnswers returns the answer rows of an attempt.
func (r *QuizRepository) ListAttemptAnswers(attemptID uint) ([]model.QuizAttemptAnswer, error) {
	var rows []model.QuizAttemptAnswer
	if err := r.db.Where("attempt_id = ?", attemptID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// FirstDoneAttempt returns the user's earliest completed attempt for a set.
// 同一题集只有首次完成的结果计入成绩统计，重考不重复累计。
func (r *QuizRepository) FirstDoneAttempt(userID, setID uint) (*model.QuizAttempt, error) {
	var a model.QuizAttempt
	if err := r.db.Where("user_id = ? AND set_id = ? AND status = ?", userID, setID, model.QuizAttemptDone).
		Order("submitted_at ASC, id ASC").First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// ---- 错题复习清单 ----

// UpsertReviewItemOnWrongTx marks a question as wrong in the user's review list,
// resetting the correct streak. Re-created after being resolved.
func (r *QuizRepository) UpsertReviewItemOnWrongTx(tx *gorm.DB, userID, questionID uint) error {
	var item model.QuizReviewItem
	err := tx.Where("user_id = ? AND question_id = ?", userID, questionID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&model.QuizReviewItem{
			UserID:        userID,
			QuestionID:    questionID,
			CorrectStreak: 0,
			Resolved:      false,
		}).Error
	}
	if err != nil {
		return err
	}
	item.CorrectStreak = 0
	item.Resolved = false
	return tx.Save(&item).Error
}

// FindReviewItem locates an unresolved review item for a user/question.
func (r *QuizRepository) FindReviewItem(userID, questionID uint) (*model.QuizReviewItem, error) {
	return r.FindReviewItemTx(r.db, userID, questionID)
}

// FindReviewItemTx locates an unresolved review item within a transaction.
func (r *QuizRepository) FindReviewItemTx(tx *gorm.DB, userID, questionID uint) (*model.QuizReviewItem, error) {
	var item model.QuizReviewItem
	if err := tx.Where("user_id = ? AND question_id = ? AND resolved = ?", userID, questionID, false).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// SaveReviewItem persists a review item.
func (r *QuizRepository) SaveReviewItem(item *model.QuizReviewItem) error {
	return r.SaveReviewItemTx(r.db, item)
}

// SaveReviewItemTx persists a review item within an outer transaction.
func (r *QuizRepository) SaveReviewItemTx(tx *gorm.DB, item *model.QuizReviewItem) error {
	return tx.Save(item).Error
}

// ListReviewItems returns the user's unresolved review items.
func (r *QuizRepository) ListReviewItems(userID uint) ([]model.QuizReviewItem, error) {
	var items []model.QuizReviewItem
	if err := r.db.Where("user_id = ? AND resolved = ?", userID, false).
		Order("updated_at DESC, id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindReviewAnswerByNo locates a review answer record by the idempotency key.
func (r *QuizRepository) FindReviewAnswerByNo(userID uint, answerNo string) (*model.QuizReviewAnswer, error) {
	return r.FindReviewAnswerByNoTx(r.db, userID, answerNo)
}

// FindReviewAnswerByNoTx locates a review answer record within a transaction.
func (r *QuizRepository) FindReviewAnswerByNoTx(tx *gorm.DB, userID uint, answerNo string) (*model.QuizReviewAnswer, error) {
	var a model.QuizReviewAnswer
	if err := tx.Where("user_id = ? AND answer_no = ?", userID, answerNo).First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// CreateReviewAnswerTx inserts a review answer record within a transaction.
func (r *QuizRepository) CreateReviewAnswerTx(tx *gorm.DB, a *model.QuizReviewAnswer) error {
	return tx.Create(a).Error
}

// DB exposes the underlying handle for transactional service logic.
func (r *QuizRepository) DB() *gorm.DB {
	return r.db
}
