package service

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newQuizTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.QuizQuestion{}, &model.QuizSet{}, &model.QuizSetVersion{},
		&model.QuizAttempt{}, &model.QuizAttemptAnswer{},
		&model.QuizReviewItem{}, &model.QuizReviewAnswer{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// SQLite 内存库在事务期间禁止多连接并发访问。
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Exec("DROP TABLE quiz_attempt_answers; DROP TABLE quiz_attempts; DROP TABLE quiz_set_versions; DROP TABLE quiz_sets; DROP TABLE quiz_review_answers; DROP TABLE quiz_review_items; DROP TABLE quiz_questions;").Error
	})
	return db
}

func newQuizService(db *gorm.DB) *QuizService {
	return NewQuizService(repository.NewQuizRepository(db),
		slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})))
}

func createTestQuestion(t *testing.T, db *gorm.DB, category, title string, answer int) *model.QuizQuestion {
	t.Helper()
	opts, _ := json.Marshal([]string{"A选项", "B选项", "C选项", "D选项"})
	q := &model.QuizQuestion{
		Category:    category,
		Title:       title,
		Options:     string(opts),
		AnswerIndex: answer,
		Explanation: "解析-" + title,
	}
	if err := db.Create(q).Error; err != nil {
		t.Fatalf("create question: %v", err)
	}
	return q
}

func createTestSet(t *testing.T, db *gorm.DB, category string) *model.QuizSet {
	t.Helper()
	s := &model.QuizSet{Name: category + "题集", Category: category, Status: model.QuizSetActive}
	if err := db.Create(s).Error; err != nil {
		t.Fatalf("create set: %v", err)
	}
	return s
}

// publishV1 直接发布初始版本（复用播种路径的等价逻辑）。
func publishV1(t *testing.T, svc *QuizService, category string) {
	t.Helper()
	if err := svc.PublishCategoryVersion(category); err != nil {
		t.Fatalf("publish v1: %v", err)
	}
}

func TestQuizVersionSnapshotFreezeAndNewVersion(t *testing.T) {
	db := newQuizTestDB(t)
	svc := newQuizService(db)
	q := createTestQuestion(t, db, constants.QuizCategoryPlant, "原题面", 1)
	createTestSet(t, db, constants.QuizCategoryPlant)
	publishV1(t, svc, constants.QuizCategoryPlant)

	// 用户开始第一次尝试，锁定 v1。
	first, err := svc.StartAttempt(100, &dto.StartAttemptRequest{SetID: 1, AttemptNo: "try-1"})
	if err != nil {
		t.Fatalf("start attempt: %v", err)
	}
	if first.Version != 1 || first.TotalCount != 1 {
		t.Fatalf("unexpected v1 attempt: %+v", first)
	}
	if first.Questions[0].AnswerIndex != nil {
		t.Fatalf("未交卷不应返回正确答案")
	}

	// 管理员修改题面与答案 -> 自动级联发布 v2。
	updated, err := svc.UpdateQuestion(q.ID, &dto.QuizQuestionRequest{
		Category: constants.QuizCategoryPlant, Title: "新题面",
		Options: []string{"A选项", "B选项", "C选项", "D选项"}, AnswerIndex: 2, Explanation: "新解析",
	})
	if err != nil {
		t.Fatalf("update question: %v", err)
	}
	if updated.Title != "新题面" || updated.AnswerIndex != 2 {
		t.Fatalf("题目未更新: %+v", updated)
	}

	// 进行中的尝试仍按 v1 查看（题面、答案都是旧版本）。
	view, err := svc.GetAttempt(100, "try-1")
	if err != nil {
		t.Fatalf("get attempt: %v", err)
	}
	if view.Version != 1 || view.Questions[0].Title != "原题面" {
		t.Fatalf("进行中的尝试应锁定 v1，实际: version=%d title=%s", view.Version, view.Questions[0].Title)
	}

	// 未开始的新尝试号，但题集已有进行中尝试：返回同一进行中尝试以便继续。
	second, err := svc.StartAttempt(100, &dto.StartAttemptRequest{SetID: 1, AttemptNo: "try-2"})
	if err != nil || second.ID != first.ID || second.Status != model.QuizAttemptOpen {
		t.Fatalf("存在进行中尝试时新开始请求应幂等返回该尝试，err=%v attempt=%+v", err, second)
	}

	// 先完成第一次尝试。
	res1, err := svc.SubmitAttempt(100, &dto.SubmitAttemptRequest{
		AttemptNo: "try-1",
		Answers:   map[uint]int{q.ID: 2}, // v1 正确答案是 1，选 2 判错
	})
	if err != nil {
		t.Fatalf("submit try-1: %v", err)
	}
	if !res1.FirstSubmitted || res1.Attempt.Score != 0 || res1.Attempt.Version != 1 {
		t.Fatalf("v1 作答判分异常: %+v", res1)
	}
	if res1.Attempt.Questions[0].AnswerIndex == nil || *res1.Attempt.Questions[0].AnswerIndex != 1 {
		t.Fatalf("交卷后应返回 v1 的正确答案 1")
	}

	// 现在可以开始第二次尝试，应当拿到 v2。
	second, err = svc.StartAttempt(100, &dto.StartAttemptRequest{SetID: 1, AttemptNo: "try-2"})
	if err != nil {
		t.Fatalf("start attempt 2: %v", err)
	}
	if second.Version != 2 || second.Questions[0].Title != "新题面" {
		t.Fatalf("未开始的题集应使用 v2，实际 version=%d", second.Version)
	}
}

func TestQuizSubmitIdempotentByAttemptNo(t *testing.T) {
	db := newQuizTestDB(t)
	svc := newQuizService(db)
	q := createTestQuestion(t, db, constants.QuizCategoryPest, "错题", 0)
	createTestSet(t, db, constants.QuizCategoryPest)
	publishV1(t, svc, constants.QuizCategoryPest)

	if _, err := svc.StartAttempt(7, &dto.StartAttemptRequest{SetID: 1, AttemptNo: "net-retry"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	// 答错（选 3，正确为 0），模拟写入后客户端未收到响应、按同一尝试号重发。
	payload := &dto.SubmitAttemptRequest{AttemptNo: "net-retry", Answers: map[uint]int{q.ID: 3}}
	first, err := svc.SubmitAttempt(7, payload)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !first.FirstSubmitted || first.Attempt.Score != 0 {
		t.Fatalf("首次交卷结果异常: %+v", first)
	}
	retry, err := svc.SubmitAttempt(7, payload)
	if err != nil {
		t.Fatalf("retry submit: %v", err)
	}
	if retry.FirstSubmitted {
		t.Fatalf("同一尝试号重复交卷应标记 first_submitted=false")
	}

	// 作答明细只写入一份，错题清单只进入一次。
	if n, err := svc.repo.CountAttemptAnswers(first.Attempt.ID); err != nil || n != 1 {
		t.Fatalf("作答明细应只有 1 行，实际 n=%d err=%v", n, err)
	}
	review, err := svc.ListReview(7)
	if err != nil {
		t.Fatalf("list review: %v", err)
	}
	if len(review) != 1 || review[0].QuestionID != q.ID {
		t.Fatalf("错题应进入复习清单 1 次，实际: %+v", review)
	}

	// 同一尝试号开始接口也必须幂等（开始请求本身可能写失败重试）。
	replay, err := svc.StartAttempt(7, &dto.StartAttemptRequest{SetID: 1, AttemptNo: "net-retry"})
	if err != nil {
		t.Fatalf("start replay: %v", err)
	}
	if replay.ID != first.Attempt.ID || replay.Status != model.QuizAttemptDone {
		t.Fatalf("开始接口幂等回放应返回同一已完成尝试")
	}
}

func TestQuizReviewWrongExitsAfterTwoCorrect(t *testing.T) {
	db := newQuizTestDB(t)
	svc := newQuizService(db)
	q := createTestQuestion(t, db, constants.QuizCategoryArticle, "复习题", 2)
	createTestSet(t, db, constants.QuizCategoryArticle)
	publishV1(t, svc, constants.QuizCategoryArticle)
	if _, err := svc.StartAttempt(9, &dto.StartAttemptRequest{SetID: 1, AttemptNo: "a1"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := svc.SubmitAttempt(9, &dto.SubmitAttemptRequest{AttemptNo: "a1", Answers: map[uint]int{q.ID: 0}}); err != nil {
		t.Fatalf("submit: %v", err)
	}

	// 第一次答对：连对 1，不退出。
	r1, err := svc.AnswerReview(9, &dto.ReviewAnswerRequest{AnswerNo: "ans-1", QuestionID: q.ID, Selected: 2})
	if err != nil {
		t.Fatalf("review answer 1: %v", err)
	}
	if !r1.Correct || r1.CorrectStreak != 1 || r1.Resolved {
		t.Fatalf("第一次答对不应退出: %+v", r1)
	}

	// 答错一次：连对清零。
	r2, err := svc.AnswerReview(9, &dto.ReviewAnswerRequest{AnswerNo: "ans-2", QuestionID: q.ID, Selected: 1})
	if err != nil {
		t.Fatalf("review answer 2: %v", err)
	}
	if r2.Correct || r2.CorrectStreak != 0 || r2.Resolved {
		t.Fatalf("答错应清零连对: %+v", r2)
	}

	// 同一 answer_no 重试：结果回放，不重复累计。
	r2Retry, err := svc.AnswerReview(9, &dto.ReviewAnswerRequest{AnswerNo: "ans-2", QuestionID: q.ID, Selected: 1})
	if err != nil {
		t.Fatalf("review answer 2 retry: %v", err)
	}
	if r2Retry.CorrectStreak != 0 {
		t.Fatalf("重试不应改变连对次数: %+v", r2Retry)
	}

	// 再连续答对两次才退出。
	if r, _ := svc.AnswerReview(9, &dto.ReviewAnswerRequest{AnswerNo: "ans-3", QuestionID: q.ID, Selected: 2}); r.CorrectStreak != 1 || r.Resolved {
		t.Fatalf("第 3 次作答后连对应为 1: %+v", r)
	}
	r4, err := svc.AnswerReview(9, &dto.ReviewAnswerRequest{AnswerNo: "ans-4", QuestionID: q.ID, Selected: 2})
	if err != nil {
		t.Fatalf("review answer 4: %v", err)
	}
	if !r4.Resolved || r4.CorrectStreak != constants.QuizReviewPassStreak {
		t.Fatalf("连续答对两次应退出复习清单: %+v", r4)
	}
	review, _ := svc.ListReview(9)
	if len(review) != 0 {
		t.Fatalf("退出后复习清单应为空，实际 %d 条", len(review))
	}
}

func TestQuizProgressCountsOnlyFirstCompletion(t *testing.T) {
	db := newQuizTestDB(t)
	svc := newQuizService(db)
	q := createTestQuestion(t, db, constants.QuizCategoryPlant, "唯一题", 0)
	createTestSet(t, db, constants.QuizCategoryPlant)
	publishV1(t, svc, constants.QuizCategoryPlant)

	// 未开始。
	p, err := svc.Progress(55)
	if err != nil {
		t.Fatalf("progress: %v", err)
	}
	if p.Sets[0].Status != "not_started" || p.TotalScore != 0 || p.ReviewCount != 0 {
		t.Fatalf("初始进度异常: %+v", p)
	}

	// 进行中：首页进度应能读到 open 状态和同一尝试号。
	if _, err := svc.StartAttempt(55, &dto.StartAttemptRequest{SetID: 1, AttemptNo: "open-no"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	p, _ = svc.Progress(55)
	if p.Sets[0].Status != model.QuizAttemptOpen || p.Sets[0].AttemptNo != "open-no" {
		t.Fatalf("进行中进度异常: %+v", p.Sets[0])
	}

	// 首次完成，答错 -> 成绩 0，错题 1。
	if _, err := svc.SubmitAttempt(55, &dto.SubmitAttemptRequest{AttemptNo: "open-no", Answers: map[uint]int{q.ID: 1}}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	p, _ = svc.Progress(55)
	if p.Sets[0].Status != model.QuizAttemptDone || p.Sets[0].Score != 0 || p.TotalScore != 0 || p.ReviewCount != 1 {
		t.Fatalf("首次完成进度异常: %+v", p)
	}

	// 重考一次并答对：总成绩仍以首次完成为准（0 分），不重复累计。
	if _, err := svc.StartAttempt(55, &dto.StartAttemptRequest{SetID: 1, AttemptNo: "retake-no"}); err != nil {
		t.Fatalf("start retake: %v", err)
	}
	if _, err := svc.SubmitAttempt(55, &dto.SubmitAttemptRequest{AttemptNo: "retake-no", Answers: map[uint]int{q.ID: 0}}); err != nil {
		t.Fatalf("submit retake: %v", err)
	}
	p, _ = svc.Progress(55)
	if p.TotalScore != 0 || p.Sets[0].Score != 0 || p.Sets[0].AttemptNo != "open-no" {
		t.Fatalf("重考不应改变首次完成成绩，实际: %+v", p)
	}
}

func TestQuizCompletedAttemptKeepsOldVersionAfterUpdate(t *testing.T) {
	db := newQuizTestDB(t)
	svc := newQuizService(db)
	q := createTestQuestion(t, db, constants.QuizCategoryPest, "旧版题", 0)
	createTestSet(t, db, constants.QuizCategoryPest)
	publishV1(t, svc, constants.QuizCategoryPest)

	if _, err := svc.StartAttempt(88, &dto.StartAttemptRequest{SetID: 1, AttemptNo: "done-1"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := svc.SubmitAttempt(88, &dto.SubmitAttemptRequest{AttemptNo: "done-1", Answers: map[uint]int{q.ID: 0}}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	// 题库改答案后级联 v2。
	if _, err := svc.UpdateQuestion(q.ID, &dto.QuizQuestionRequest{
		Category: constants.QuizCategoryPest, Title: "新版题",
		Options: []string{"A选项", "B选项", "C选项", "D选项"}, AnswerIndex: 3, Explanation: "新解析",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	// 已完成尝试查看时仍按 v1 快照展示判分。
	view, err := svc.GetAttempt(88, "done-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.Version != 1 || view.Questions[0].Title != "旧版题" || *view.Questions[0].AnswerIndex != 0 {
		t.Fatalf("已完成尝试必须仍按原版本查看: %+v", view)
	}
}
