package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// QuizService implements quiz set listing, attempt lifecycle with versioned
// snapshots, idempotent submission and the wrong-question review list.
type QuizService struct {
	db           *gorm.DB
	setRepo      *repository.QuizSetRepository
	questionRepo *repository.QuizQuestionRepository
	attemptRepo  *repository.QuizAttemptRepository
	wrongRepo    *repository.QuizWrongQuestionRepository
	logger       *slog.Logger
}

// NewQuizService creates a QuizService.
func NewQuizService(db *gorm.DB, setRepo *repository.QuizSetRepository, questionRepo *repository.QuizQuestionRepository,
	attemptRepo *repository.QuizAttemptRepository, wrongRepo *repository.QuizWrongQuestionRepository, logger *slog.Logger) *QuizService {
	return &QuizService{db: db, setRepo: setRepo, questionRepo: questionRepo, attemptRepo: attemptRepo, wrongRepo: wrongRepo, logger: logger}
}

// ListSets returns published quiz sets with the current user's status. Sets
// never started always reflect the latest bank version; in-progress and
// completed attempts keep the version they were started with.
func (s *QuizService) ListSets(userID uint) ([]dto.QuizSetItem, error) {
	sets, err := s.setRepo.ListPublished()
	if err != nil {
		return nil, fmt.Errorf("quiz set list: %w", err)
	}
	items := make([]dto.QuizSetItem, 0, len(sets))
	for _, set := range sets {
		item := dto.QuizSetItem{
			ID: set.ID, Title: set.Title, Category: set.Category,
			CategoryText: util.QuizCategoryText(set.Category),
			Description:  set.Description, Version: set.Version,
			MyStatus: constants.QuizMyStatusNotStarted,
		}
		count, err := s.questionRepo.CountBySetVersion(set.ID, set.Version)
		if err != nil {
			return nil, fmt.Errorf("quiz set question count: %w", err)
		}
		item.QuestionCount = count
		if inProgress, err := s.attemptRepo.FindInProgress(userID, set.ID); err == nil {
			item.MyStatus = constants.QuizMyStatusInProgress
			item.MyAttemptID = inProgress.ID
		} else if !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("quiz set in-progress find: %w", err)
		} else if best, err := s.attemptRepo.FindBestSubmitted(userID, set.ID); err == nil {
			item.MyStatus = constants.QuizMyStatusCompleted
			item.MyAttemptID = best.ID
			item.BestScore = best.Score
			item.BestTotal = best.Total
		} else if !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("quiz set best find: %w", err)
		}
		item.MyStatusText = util.QuizMyStatusText(item.MyStatus)
		items = append(items, item)
	}
	return items, nil
}

// StartAttempt returns the in-progress attempt if one exists, otherwise
// snapshots the current bank version into a new attempt. Concurrent starts
// race on the unique (user_id, quiz_set_id, attempt_no) key and the loser
// falls back to the attempt that won, so only the first result counts.
func (s *QuizService) StartAttempt(userID, setID uint) (*dto.QuizAttemptDetail, error) {
	set, err := s.setRepo.FindByID(setID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("QuizSet[id=%d] not found", setID))
		}
		return nil, fmt.Errorf("quiz attempt start set find: %w", err)
	}
	if set.Status != model.QuizSetStatusPublished {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("QuizSet[id=%d] start failed: status=%s not published", setID, set.Status))
	}
	if existing, err := s.attemptRepo.FindInProgress(userID, setID); err == nil {
		return s.buildAttemptDetail(existing, false)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("quiz attempt start in-progress find: %w", err)
	}
	questions, err := s.questionRepo.ListBySetVersion(setID, set.Version)
	if err != nil {
		return nil, fmt.Errorf("quiz attempt start questions: %w", err)
	}
	if len(questions) == 0 {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("QuizSet[id=%d] start failed: version=%d has no questions", setID, set.Version))
	}
	snapshot := make([]model.QuizSnapshotQuestion, 0, len(questions))
	for _, q := range questions {
		var options []string
		if err := json.Unmarshal([]byte(q.Options), &options); err != nil {
			return nil, fmt.Errorf("quiz attempt start options decode: %w", err)
		}
		snapshot = append(snapshot, model.QuizSnapshotQuestion{
			QuestionID: q.ID, Question: q.Question, Options: options,
			Answer: q.Answer, Explanation: q.Explanation,
		})
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("quiz attempt start snapshot encode: %w", err)
	}
	maxNo, err := s.attemptRepo.MaxAttemptNo(userID, setID)
	if err != nil {
		return nil, fmt.Errorf("quiz attempt start max no: %w", err)
	}
	attempt := &model.QuizAttempt{
		UserID: userID, QuizSetID: setID, AttemptNo: maxNo + 1,
		SetVersion: set.Version, Snapshot: string(snapshotJSON), Answers: "{}",
		Total: len(snapshot), Status: model.QuizAttemptInProgress,
	}
	if err := s.attemptRepo.Create(attempt); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			existing, findErr := s.attemptRepo.FindInProgress(userID, setID)
			if findErr == nil {
				return s.buildAttemptDetail(existing, false)
			}
			return nil, fmt.Errorf("quiz attempt start duplicate refetch: %w", findErr)
		}
		s.logger.Error(fmt.Sprintf(constants.LogQuizAttemptStartFailed, userID, setID), "error", err)
		return nil, fmt.Errorf("quiz attempt create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogQuizAttemptStartSuccess, userID, setID, attempt.AttemptNo, attempt.SetVersion), "attempt_id", attempt.ID)
	return s.buildAttemptDetail(attempt, false)
}

// GetAttempt returns an attempt owned by the user. In-progress attempts hide
// answers; submitted ones reveal the frozen answers and explanations.
func (s *QuizService) GetAttempt(userID, attemptID uint) (*dto.QuizAttemptDetail, error) {
	attempt, err := s.attemptRepo.FindByID(attemptID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("QuizAttempt[id=%d] not found", attemptID))
		}
		return nil, fmt.Errorf("quiz attempt find: %w", err)
	}
	if attempt.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("QuizAttempt[id=%d] view failed: user_id=%d not owner", attemptID, userID))
	}
	return s.buildAttemptDetail(attempt, attempt.Status == model.QuizAttemptSubmitted)
}

// Submit grades an attempt inside a transaction. The attempt row is locked,
// so simultaneous submissions of the same set resolve to the first result;
// a retry with the same attempt number after a write failure returns the
// stored result without accumulating the score or review entries twice.
func (s *QuizService) Submit(userID, attemptID uint, answers map[string]int) (*dto.QuizAttemptDetail, error) {
	var detail *dto.QuizAttemptDetail
	err := s.db.Transaction(func(tx *gorm.DB) error {
		attempt, err := s.attemptRepo.FindByIDForUpdate(tx, attemptID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("QuizAttempt[id=%d] not found", attemptID))
			}
			return fmt.Errorf("quiz submit find: %w", err)
		}
		if attempt.UserID != userID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("QuizAttempt[id=%d] submit failed: user_id=%d not owner", attemptID, userID))
		}
		snapshot, err := decodeSnapshot(attempt.Snapshot)
		if err != nil {
			return fmt.Errorf("quiz submit snapshot decode: %w", err)
		}
		if attempt.Status == model.QuizAttemptSubmitted {
			s.logger.Info(fmt.Sprintf(constants.LogQuizSubmitDuplicate, attempt.ID, attempt.AttemptNo))
			detail, err = s.buildAttemptDetail(attempt, true)
			return err
		}
		score := 0
		for _, q := range snapshot {
			selected, ok := answers[strconv.FormatUint(uint64(q.QuestionID), 10)]
			if ok && selected == q.Answer {
				score++
			}
		}
		answersJSON, err := json.Marshal(answers)
		if err != nil {
			return fmt.Errorf("quiz submit answers encode: %w", err)
		}
		now := time.Now()
		attempt.Answers = string(answersJSON)
		attempt.Score = score
		attempt.Status = model.QuizAttemptSubmitted
		attempt.SubmittedAt = &now
		if err := s.attemptRepo.UpdateTx(tx, attempt); err != nil {
			s.logger.Error(fmt.Sprintf(constants.LogQuizSubmitFailed, attempt.ID), "error", err)
			return fmt.Errorf("quiz submit update: %w", err)
		}
		for _, q := range snapshot {
			selected, ok := answers[strconv.FormatUint(uint64(q.QuestionID), 10)]
			if !ok || selected != q.Answer {
				if err := s.upsertWrongQuestion(tx, userID, attempt.QuizSetID, q); err != nil {
					return err
				}
			} else if err := s.markQuestionCorrect(tx, userID, q.QuestionID); err != nil {
				return err
			}
		}
		s.logger.Info(fmt.Sprintf(constants.LogQuizSubmitSuccess, attempt.ID, attempt.AttemptNo, score, attempt.Total))
		detail, err = s.buildAttemptDetail(attempt, true)
		return err
	})
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// ListReview returns the pending wrong questions of a user, replaying the
// snapshot of the version that was failed.
func (s *QuizService) ListReview(userID uint) ([]dto.QuizReviewItem, error) {
	items, err := s.wrongRepo.ListPendingByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("quiz review list: %w", err)
	}
	out := make([]dto.QuizReviewItem, 0, len(items))
	for _, w := range items {
		snap, err := decodeSnapshotOne(w.Snapshot)
		if err != nil {
			return nil, fmt.Errorf("quiz review snapshot decode: %w", err)
		}
		out = append(out, dto.QuizReviewItem{
			QuestionID: w.QuestionID, QuizSetID: w.QuizSetID,
			Question: snap.Question, Options: snap.Options,
			ConsecutiveCorrect: w.ConsecutiveCorrect,
		})
	}
	return out, nil
}

// AnswerReview grades one review answer against the frozen snapshot. A wrong
// question leaves the review list only after WrongQuestionResolveThreshold
// consecutive correct answers; a wrong answer resets the streak.
func (s *QuizService) AnswerReview(userID, questionID uint, selected int) (*dto.QuizReviewAnswerResult, error) {
	var result *dto.QuizReviewAnswerResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		wrong, err := s.wrongRepo.FindByUserQuestionForUpdate(tx, userID, questionID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("QuizWrongQuestion[question_id=%d] not in review list of user_id=%d", questionID, userID))
			}
			return fmt.Errorf("quiz review answer find: %w", err)
		}
		if wrong.Status != model.WrongQuestionPending {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("QuizWrongQuestion[question_id=%d] review failed: status=%s not pending", questionID, wrong.Status))
		}
		snap, err := decodeSnapshotOne(wrong.Snapshot)
		if err != nil {
			return fmt.Errorf("quiz review answer snapshot decode: %w", err)
		}
		correct := selected == snap.Answer
		if correct {
			wrong.ConsecutiveCorrect++
			if wrong.ConsecutiveCorrect >= model.WrongQuestionResolveThreshold {
				wrong.Status = model.WrongQuestionResolved
			}
		} else {
			wrong.ConsecutiveCorrect = 0
		}
		if err := s.wrongRepo.UpdateTx(tx, wrong); err != nil {
			return fmt.Errorf("quiz review answer update: %w", err)
		}
		s.logger.Info(fmt.Sprintf(constants.LogQuizReviewAnswered, userID, questionID, correct, wrong.ConsecutiveCorrect))
		result = &dto.QuizReviewAnswerResult{
			Correct: correct, ConsecutiveCorrect: wrong.ConsecutiveCorrect,
			Resolved: wrong.Status == model.WrongQuestionResolved,
			Answer:   snap.Answer, Explanation: snap.Explanation,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ReviewProgress is the single progress source shared by the review page and
// the home page.
func (s *QuizService) ReviewProgress(userID uint) (*dto.QuizReviewProgress, error) {
	pending, err := s.wrongRepo.CountByStatus(userID, model.WrongQuestionPending)
	if err != nil {
		return nil, fmt.Errorf("quiz review progress pending: %w", err)
	}
	resolved, err := s.wrongRepo.CountByStatus(userID, model.WrongQuestionResolved)
	if err != nil {
		return nil, fmt.Errorf("quiz review progress resolved: %w", err)
	}
	return &dto.QuizReviewProgress{Pending: pending, Resolved: resolved}, nil
}

// upsertWrongQuestion inserts or resets a review entry for a failed question.
func (s *QuizService) upsertWrongQuestion(tx *gorm.DB, userID, setID uint, q model.QuizSnapshotQuestion) error {
	snapJSON, err := json.Marshal(q)
	if err != nil {
		return fmt.Errorf("quiz wrong snapshot encode: %w", err)
	}
	existing, err := s.wrongRepo.FindByUserQuestionForUpdate(tx, userID, q.QuestionID)
	if errors.Is(err, repository.ErrNotFound) {
		return s.wrongRepo.CreateTx(tx, &model.QuizWrongQuestion{
			UserID: userID, QuizSetID: setID, QuestionID: q.QuestionID,
			Snapshot: string(snapJSON), ConsecutiveCorrect: 0, Status: model.WrongQuestionPending,
		})
	}
	if err != nil {
		return fmt.Errorf("quiz wrong find: %w", err)
	}
	existing.QuizSetID = setID
	existing.Snapshot = string(snapJSON)
	existing.ConsecutiveCorrect = 0
	existing.Status = model.WrongQuestionPending
	return s.wrongRepo.UpdateTx(tx, existing)
}

// markQuestionCorrect advances the review streak of a pending entry.
func (s *QuizService) markQuestionCorrect(tx *gorm.DB, userID, questionID uint) error {
	existing, err := s.wrongRepo.FindByUserQuestionForUpdate(tx, userID, questionID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("quiz wrong correct find: %w", err)
	}
	if existing.Status != model.WrongQuestionPending {
		return nil
	}
	existing.ConsecutiveCorrect++
	if existing.ConsecutiveCorrect >= model.WrongQuestionResolveThreshold {
		existing.Status = model.WrongQuestionResolved
	}
	return s.wrongRepo.UpdateTx(tx, existing)
}

// buildAttemptDetail renders an attempt. While in progress the frozen answers
// stay hidden; after submission they are revealed together with the choices.
func (s *QuizService) buildAttemptDetail(attempt *model.QuizAttempt, reveal bool) (*dto.QuizAttemptDetail, error) {
	snapshot, err := decodeSnapshot(attempt.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("quiz attempt detail snapshot decode: %w", err)
	}
	answers := map[string]int{}
	if attempt.Answers != "" {
		if err := json.Unmarshal([]byte(attempt.Answers), &answers); err != nil {
			return nil, fmt.Errorf("quiz attempt detail answers decode: %w", err)
		}
	}
	detail := &dto.QuizAttemptDetail{
		AttemptID: attempt.ID, QuizSetID: attempt.QuizSetID, AttemptNo: attempt.AttemptNo,
		SetVersion: attempt.SetVersion, Status: attempt.Status,
		StatusText: util.QuizAttemptStatusText(attempt.Status),
		Score:      attempt.Score, Total: attempt.Total, SubmittedAt: attempt.SubmittedAt,
		Questions: make([]dto.QuizQuestionResultView, 0, len(snapshot)),
	}
	for _, q := range snapshot {
		view := dto.QuizQuestionResultView{QuestionID: q.QuestionID, Question: q.Question, Options: q.Options}
		if reveal {
			answer := q.Answer
			view.Answer = &answer
			view.Explanation = q.Explanation
			if selected, ok := answers[strconv.FormatUint(uint64(q.QuestionID), 10)]; ok {
				view.Selected = &selected
				view.Correct = selected == q.Answer
			}
		}
		detail.Questions = append(detail.Questions, view)
	}
	return detail, nil
}

func decodeSnapshot(raw string) ([]model.QuizSnapshotQuestion, error) {
	var snapshot []model.QuizSnapshotQuestion
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func decodeSnapshotOne(raw string) (*model.QuizSnapshotQuestion, error) {
	var snap model.QuizSnapshotQuestion
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}
