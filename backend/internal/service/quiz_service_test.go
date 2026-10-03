package service

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
)

func newQuizService(t *testing.T) (*QuizService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := newServiceDB(t)
	return NewQuizService(db,
		repository.NewQuizSetRepository(db),
		repository.NewQuizQuestionRepository(db),
		repository.NewQuizAttemptRepository(db),
		repository.NewQuizWrongQuestionRepository(db),
		newTestLogger()), mock
}

func quizSetRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "title", "category", "description", "version", "status", "created_at", "updated_at"})
}

func TestQuizServiceListSetsStatuses(t *testing.T) {
	svc, mock := newQuizService(t)
	// three published sets
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `quiz_sets` WHERE status = ? ORDER BY id ASC")).
		WithArgs("published").
		WillReturnRows(quizSetRows().
			AddRow(1, "品种知识速览", "plant", "", 2, "published", time.Now(), time.Now()).
			AddRow(2, "养护文章要点", "article", "", 1, "published", time.Now(), time.Now()).
			AddRow(3, "病虫害防治手册", "pest", "", 1, "published", time.Now(), time.Now()))
	attemptCols := []string{"id", "user_id", "quiz_set_id", "attempt_no", "set_version", "snapshot", "answers", "score", "total", "status", "created_at", "submitted_at"}
	// set 1: in-progress attempt exists
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `quiz_questions`").WithArgs(uint(1), 2).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery("SELECT \\* FROM `quiz_attempts` WHERE .*status = ?").
		WillReturnRows(sqlmock.NewRows(attemptCols).
			AddRow(9, 7, 1, 1, 1, "[]", "", 0, 5, model.QuizAttemptInProgress, time.Now(), nil))
	// set 2: no in-progress, best submitted exists
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `quiz_questions`").WithArgs(uint(2), 1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery("SELECT \\* FROM `quiz_attempts` WHERE .*status = ?").
		WillReturnRows(sqlmock.NewRows(attemptCols))
	mock.ExpectQuery("SELECT \\* FROM `quiz_attempts` WHERE .*status = ?").
		WillReturnRows(sqlmock.NewRows(attemptCols).
			AddRow(5, 7, 2, 1, 1, "[]", "{}", 4, 5, model.QuizAttemptSubmitted, time.Now(), time.Now()))
	// set 3: never started
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `quiz_questions`").WithArgs(uint(3), 1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery("SELECT \\* FROM `quiz_attempts` WHERE .*status = ?").
		WillReturnRows(sqlmock.NewRows(attemptCols))
	mock.ExpectQuery("SELECT \\* FROM `quiz_attempts` WHERE .*status = ?").
		WillReturnRows(sqlmock.NewRows(attemptCols))

	items, err := svc.ListSets(7)
	if err != nil {
		t.Fatalf("ListSets: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 sets, got %d", len(items))
	}
	if items[0].MyStatus != "in_progress" || items[0].MyAttemptID != 9 {
		t.Errorf("set 1: expected in_progress attempt 9, got %+v", items[0])
	}
	if items[1].MyStatus != "completed" || items[1].BestScore != 4 || items[1].BestTotal != 5 {
		t.Errorf("set 2: expected completed best 4/5, got %+v", items[1])
	}
	if items[2].MyStatus != "not_started" {
		t.Errorf("set 3: expected not_started, got %+v", items[2])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestQuizServiceSubmitGradesAndUpsertsWrong(t *testing.T) {
	svc, mock := newQuizService(t)
	snapshot := `[{"question_id":11,"question":"q1","options":["a","b"],"answer":1,"explanation":"e1"},` +
		`{"question_id":12,"question":"q2","options":["a","b"],"answer":0,"explanation":"e2"}]`
	attemptCols := []string{"id", "user_id", "quiz_set_id", "attempt_no", "set_version", "snapshot", "answers", "score", "total", "status", "created_at", "submitted_at"}
	wrongCols := []string{"id", "user_id", "quiz_set_id", "question_id", "snapshot", "consecutive_correct", "status", "created_at", "updated_at"}

	mock.ExpectBegin()
	// lock the in-progress attempt
	mock.ExpectQuery("SELECT \\* FROM `quiz_attempts` WHERE .*FOR UPDATE").
		WithArgs(uint(50), 1).
		WillReturnRows(sqlmock.NewRows(attemptCols).
			AddRow(50, 7, 1, 1, 1, snapshot, "", 0, 2, model.QuizAttemptInProgress, time.Now(), nil))
	// grade: q11 correct, q12 wrong -> score 1, persist attempt
	mock.ExpectExec("UPDATE `quiz_attempts` SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// q11 correct: review entry lookup misses -> nothing to advance
	mock.ExpectQuery("SELECT \\* FROM `quiz_wrong_questions` WHERE .*FOR UPDATE").
		WillReturnRows(sqlmock.NewRows(wrongCols))
	// q12 wrong: review entry lookup misses -> insert new pending entry
	mock.ExpectQuery("SELECT \\* FROM `quiz_wrong_questions` WHERE .*FOR UPDATE").
		WillReturnRows(sqlmock.NewRows(wrongCols))
	mock.ExpectExec("INSERT INTO `quiz_wrong_questions`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	detail, err := svc.Submit(7, 50, map[string]int{"11": 1, "12": 1})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if detail.Score != 1 || detail.Total != 2 || detail.Status != model.QuizAttemptSubmitted {
		t.Errorf("unexpected detail: %+v", detail)
	}
	if len(detail.Questions) != 2 || !detail.Questions[0].Correct || detail.Questions[1].Correct {
		t.Errorf("unexpected per-question results: %+v", detail.Questions)
	}
	if detail.Questions[0].Answer == nil || *detail.Questions[0].Answer != 1 || detail.Questions[0].Explanation != "e1" {
		t.Errorf("submitted attempt must reveal frozen answers: %+v", detail.Questions[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestQuizServiceSubmitDuplicateReturnsFirstResult(t *testing.T) {
	svc, mock := newQuizService(t)
	snapshot := `[{"question_id":11,"question":"q1","options":["a","b"],"answer":1,"explanation":"e1"}]`
	attemptCols := []string{"id", "user_id", "quiz_set_id", "attempt_no", "set_version", "snapshot", "answers", "score", "total", "status", "created_at", "submitted_at"}
	now := time.Now()

	mock.ExpectBegin()
	// attempt already submitted: retry with the same attempt number must not
	// re-grade or touch the review list, only return the stored first result.
	mock.ExpectQuery("SELECT \\* FROM `quiz_attempts` WHERE .*FOR UPDATE").
		WithArgs(uint(50), 1).
		WillReturnRows(sqlmock.NewRows(attemptCols).
			AddRow(50, 7, 1, 3, 2, snapshot, `{"11":1}`, 1, 1, model.QuizAttemptSubmitted, now, now))
	mock.ExpectCommit()

	detail, err := svc.Submit(7, 50, map[string]int{"11": 0})
	if err != nil {
		t.Fatalf("Submit duplicate: %v", err)
	}
	if detail.Score != 1 || detail.AttemptNo != 3 || detail.SetVersion != 2 {
		t.Errorf("duplicate submit must return the first stored result, got %+v", detail)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations (score must not be accumulated twice): %v", err)
	}
}

func TestQuizServiceAnswerReviewStreak(t *testing.T) {
	wrongCols := []string{"id", "user_id", "quiz_set_id", "question_id", "snapshot", "consecutive_correct", "status", "created_at", "updated_at"}
	snap := `{"question_id":11,"question":"q1","options":["a","b"],"answer":1,"explanation":"e1"}`

	tests := []struct {
		name         string
		streak       int
		status       string
		selected     int
		wantCorrect  bool
		wantStreak   int
		wantResolved bool
	}{
		{"first correct advances streak", 0, model.WrongQuestionPending, 1, true, 1, false},
		{"second consecutive correct resolves", 1, model.WrongQuestionPending, 1, true, 2, true},
		{"wrong answer resets streak", 1, model.WrongQuestionPending, 0, false, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, mock := newQuizService(t)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT \\* FROM `quiz_wrong_questions` WHERE .*FOR UPDATE").
				WillReturnRows(sqlmock.NewRows(wrongCols).
					AddRow(3, 7, 1, 11, snap, tt.streak, tt.status, time.Now(), time.Now()))
			mock.ExpectExec("UPDATE `quiz_wrong_questions` SET").
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			res, err := svc.AnswerReview(7, 11, tt.selected)
			if err != nil {
				t.Fatalf("AnswerReview: %v", err)
			}
			if res.Correct != tt.wantCorrect || res.ConsecutiveCorrect != tt.wantStreak || res.Resolved != tt.wantResolved {
				t.Errorf("got %+v, want correct=%v streak=%d resolved=%v",
					res, tt.wantCorrect, tt.wantStreak, tt.wantResolved)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestQuizServiceAnswerReviewNotInList(t *testing.T) {
	svc, mock := newQuizService(t)
	wrongCols := []string{"id", "user_id", "quiz_set_id", "question_id", "snapshot", "consecutive_correct", "status", "created_at", "updated_at"}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM `quiz_wrong_questions` WHERE .*FOR UPDATE").
		WillReturnRows(sqlmock.NewRows(wrongCols))
	mock.ExpectRollback()
	if _, err := svc.AnswerReview(7, 99, 0); err == nil {
		t.Error("expected error for question not in review list")
	}
}
