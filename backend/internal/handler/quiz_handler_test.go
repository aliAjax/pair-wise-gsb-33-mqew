package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gbplantwiki/gbplantwiki/internal/config"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/service"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

const testJWTSecret = "test-secret-test-secret-test-secret"

func newQuizRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:quiz_http_"+t.Name()+"?mode=memory"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.QuizQuestion{}, &model.QuizSet{}, &model.QuizSetVersion{},
		&model.QuizAttempt{}, &model.QuizAttemptAnswer{},
		&model.QuizReviewItem{}, &model.QuizReviewAnswer{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := repository.NewQuizRepository(db)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := service.NewQuizService(repo, logger)
	h := NewQuizHandler(svc, logger)

	cfg := &config.Config{JWTSecret: testJWTSecret, RateLimitReq: 1000, RateLimitWin: time.Minute}
	r := gin.New()
	v1 := r.Group("/api/v1")
	quiz := v1.Group("/quiz", middleware.AuthRequired(cfg))
	quiz.GET("/sets", h.ListSets)
	quiz.GET("/progress", h.Progress)
	quiz.GET("/review", h.ListReview)
	quiz.POST("/review/answer", h.AnswerReview)
	quiz.POST("/attempts", h.StartAttempt)
	quiz.GET("/attempts/:attemptNo", h.GetAttempt)
	quiz.POST("/attempts/submit", h.SubmitAttempt)
	admin := quiz.Group("/admin", middleware.RequireRole("admin"))
	admin.GET("/questions", h.ListQuestions)
	admin.POST("/questions", h.CreateQuestion)
	admin.PUT("/questions/:id", h.UpdateQuestion)
	return r, db
}

func seedHTTPQuiz(t *testing.T, db *gorm.DB) {
	t.Helper()
	sets := []model.QuizSet{
		{Name: "品种题集", Category: "plant", Status: model.QuizSetActive},
	}
	if err := db.Create(&sets).Error; err != nil {
		t.Fatalf("seed sets: %v", err)
	}
	questions := []model.QuizQuestion{
		{Category: "plant", Title: "题1", Options: `["甲","乙","丙","丁"]`, AnswerIndex: 0, Explanation: "解析1"},
		{Category: "plant", Title: "题2", Options: `["甲","乙","丙","丁"]`, AnswerIndex: 1, Explanation: "解析2"},
	}
	if err := db.Create(&questions).Error; err != nil {
		t.Fatalf("seed questions: %v", err)
	}
	// v1 快照
	type snap struct {
		QuestionID  uint     `json:"question_id"`
		Title       string   `json:"title"`
		Options     []string `json:"options"`
		AnswerIndex int      `json:"answer_index"`
		Explanation string   `json:"explanation"`
	}
	s := []snap{
		{QuestionID: questions[0].ID, Title: "题1", Options: []string{"甲", "乙", "丙", "丁"}, AnswerIndex: 0, Explanation: "解析1"},
		{QuestionID: questions[1].ID, Title: "题2", Options: []string{"甲", "乙", "丙", "丁"}, AnswerIndex: 1, Explanation: "解析2"},
	}
	ids := []uint{questions[0].ID, questions[1].ID}
	idBytes, _ := json.Marshal(ids)
	snapBytes, _ := json.Marshal(s)
	if err := db.Create(&model.QuizSetVersion{SetID: sets[0].ID, Version: 1, QuestionIDs: string(idBytes), Snapshot: string(snapBytes)}).Error; err != nil {
		t.Fatalf("seed version: %v", err)
	}
}

func userToken(t *testing.T, userID uint, role string) string {
	t.Helper()
	tok, err := util.GenerateToken(userID, "tester", role, testJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return tok
}

func doJSON(t *testing.T, r http.Handler, method, path, token string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	raw, _ := io.ReadAll(w.Body)
	var parsed map[string]interface{}
	_ = json.Unmarshal(raw, &parsed)
	return w.Code, parsed
}

func TestQuizHTTPFullFlow(t *testing.T) {
	r, db := newQuizRouter(t)
	seedHTTPQuiz(t, db)
	user := userToken(t, 1001, "user")
	admin := userToken(t, 2002, "admin")

	// 1. 未登录访问被拒。
	if code, _ := doJSON(t, r, "GET", "/api/v1/quiz/sets", "", nil); code != http.StatusUnauthorized {
		t.Fatalf("未登录应 401，实际 %d", code)
	}

	// 2. 题集列表。
	code, body := doJSON(t, r, "GET", "/api/v1/quiz/sets", user, nil)
	if code != http.StatusOK {
		t.Fatalf("sets: code=%d body=%v", code, body)
	}
	dataList := body["data"].([]interface{})
	if len(dataList) != 1 {
		t.Fatalf("应有 1 个题集，实际 %d", len(dataList))
	}
	setData := dataList[0].(map[string]interface{})
	setID := int(setData["id"].(float64))
	if setData["question_count"].(float64) != 2 {
		t.Fatalf("题集应含 2 题")
	}

	// 3. 开始作答（固定 attempt_no 模拟客户端）。
	code, body = doJSON(t, r, "POST", "/api/v1/quiz/attempts", user, map[string]interface{}{
		"set_id": setID, "attempt_no": "fixed-no-1",
	})
	if code != http.StatusCreated {
		t.Fatalf("start: code=%d body=%v", code, body)
	}
	attempt := body["data"].(map[string]interface{})
	if attempt["version"].(float64) != 1 {
		t.Fatalf("新版本应为 v1")
	}
	questions := attempt["questions"].([]interface{})
	q0 := questions[0].(map[string]interface{})
	if q0["answer_index"] != nil {
		t.Fatalf("未交卷不得返回答案")
	}
	q0ID := int(q0["question_id"].(float64))
	q1ID := int(questions[1].(map[string]interface{})["question_id"].(float64))

	// 4. 开始请求幂等：同一 attempt_no 再次调用返回同一尝试。
	code, body = doJSON(t, r, "POST", "/api/v1/quiz/attempts", user, map[string]interface{}{
		"set_id": setID, "attempt_no": "fixed-no-1",
	})
	if code != http.StatusCreated || body["data"].(map[string]interface{})["id"] != attempt["id"] {
		t.Fatalf("开始接口幂等失败: code=%d body=%v", code, body)
	}

	// 5. 管理员修改题 1 答案 -> 自动级联 v2；进行中的尝试查看仍为 v1。
	code, body = doJSON(t, r, "PUT", "/api/v1/quiz/admin/questions/"+strconv.Itoa(q0ID), admin, map[string]interface{}{
		"category": "plant", "title": "题1-修订",
		"options": []string{"甲", "乙", "丙", "丁"}, "answer_index": 2, "explanation": "新解析",
	})
	if code != http.StatusOK {
		t.Fatalf("admin update: code=%d body=%v", code, body)
	}
	// 普通用户无权操作题库。
	if code, _ := doJSON(t, r, "PUT", "/api/v1/quiz/admin/questions/"+strconv.Itoa(q0ID), user, map[string]interface{}{
		"category": "plant", "title": "x", "options": []string{"a", "b"}, "answer_index": 0,
	}); code != http.StatusForbidden {
		t.Fatalf("普通用户改题应 403，实际 %d", code)
	}
	code, body = doJSON(t, r, "GET", "/api/v1/quiz/attempts/fixed-no-1", user, nil)
	if code != http.StatusOK {
		t.Fatalf("get attempt: code=%d body=%v", code, body)
	}
	cur := body["data"].(map[string]interface{})
	if cur["version"].(float64) != 1 {
		t.Fatalf("进行中尝试必须锁定 v1，实际 %v", cur["version"])
	}
	curQ0 := cur["questions"].([]interface{})[0].(map[string]interface{})
	if curQ0["title"] != "题1" {
		t.Fatalf("进行中尝试题面应为旧版，实际 %v", curQ0["title"])
	}

	// 6. 交卷：题1按 v1 正确答案 0 作答（题库已改成 2，但 v1 答案仍为 0），题2答错。
	payload := map[string]interface{}{
		"attempt_no": "fixed-no-1",
		"answers":    map[int]int{q0ID: 0, q1ID: 3},
	}
	code, body = doJSON(t, r, "POST", "/api/v1/quiz/attempts/submit", user, payload)
	if code != http.StatusOK {
		t.Fatalf("submit: code=%d body=%v", code, body)
	}
	result := body["data"].(map[string]interface{})
	if result["first_submitted"] != true {
		t.Fatalf("首次交卷 first_submitted 应为 true")
	}
	done := result["attempt"].(map[string]interface{})
	if done["score"].(float64) != 1 {
		t.Fatalf("v1 判分应得 1 分，实际 %v", done["score"])
	}

	// 7. 同 payload 重交：只认首次结果，不重复累计。
	code, body = doJSON(t, r, "POST", "/api/v1/quiz/attempts/submit", user, payload)
	if code != http.StatusOK || body["data"].(map[string]interface{})["first_submitted"] != false {
		t.Fatalf("重复交卷应返回 first_submitted=false: code=%d body=%v", code, body)
	}

	// 8. 进度：已完成 1 分、错题 1；复习清单里有题2。
	code, body = doJSON(t, r, "GET", "/api/v1/quiz/progress", user, nil)
	if code != http.StatusOK {
		t.Fatalf("progress: %d", code)
	}
	p := body["data"].(map[string]interface{})
	if p["total_score"].(float64) != 1 || p["review_count"].(float64) != 1 {
		t.Fatalf("进度口径错误: score=%v review=%v", p["total_score"], p["review_count"])
	}
	code, body = doJSON(t, r, "GET", "/api/v1/quiz/review", user, nil)
	if code != http.StatusOK || len(body["data"].([]interface{})) != 1 {
		t.Fatalf("复习清单应有 1 题: code=%d body=%v", code, body)
	}

	// 9. 复习题2（v1 答案 1）：答对第一次不退出，第二次退出；answer_no 幂等。
	reviewBody := map[string]interface{}{"answer_no": "ra-1", "question_id": q1ID, "selected": 1}
	code, body = doJSON(t, r, "POST", "/api/v1/quiz/review/answer", user, reviewBody)
	if code != http.StatusOK || body["data"].(map[string]interface{})["resolved"] != false {
		t.Fatalf("第一次复习答对不应退出: code=%d body=%v", code, body)
	}
	// 同 answer_no 重试，结果回放。
	code, body = doJSON(t, r, "POST", "/api/v1/quiz/review/answer", user, reviewBody)
	if code != http.StatusOK || body["data"].(map[string]interface{})["correct_streak"].(float64) != 1 {
		t.Fatalf("复习重试幂等失败（连对仍应为 1）: code=%d body=%v", code, body)
	}
	code, body = doJSON(t, r, "POST", "/api/v1/quiz/review/answer", user,
		map[string]interface{}{"answer_no": "ra-2", "question_id": q1ID, "selected": 1})
	if code != http.StatusOK || body["data"].(map[string]interface{})["resolved"] != true {
		t.Fatalf("连续答对两次应退出: code=%d body=%v", code, body)
	}
	code, body = doJSON(t, r, "GET", "/api/v1/quiz/review", user, nil)
	if len(body["data"].([]interface{})) != 0 {
		t.Fatalf("退出后复习清单应为空")
	}

	// 10. 已完成尝试查看仍按 v1（答案 0），随后新尝试拿到 v2（答案 2）。
	code, body = doJSON(t, r, "GET", "/api/v1/quiz/attempts/fixed-no-1", user, nil)
	doneQ0 := body["data"].(map[string]interface{})["questions"].([]interface{})[0].(map[string]interface{})
	if doneQ0["title"] != "题1" || doneQ0["answer_index"].(float64) != 0 {
		t.Fatalf("已完成尝试须按 v1 查看: %v", doneQ0)
	}
	code, body = doJSON(t, r, "POST", "/api/v1/quiz/attempts", user, map[string]interface{}{
		"set_id": setID, "attempt_no": "fixed-no-2",
	})
	if code != http.StatusCreated {
		t.Fatalf("第二次开始: code=%d body=%v", code, body)
	}
	second := body["data"].(map[string]interface{})
	if second["version"].(float64) != 2 {
		t.Fatalf("未开始的尝试应使用 v2，实际 %v", second["version"])
	}
	secondQ0 := second["questions"].([]interface{})[0].(map[string]interface{})
	if secondQ0["title"] != "题1-修订" {
		t.Fatalf("v2 题面应为修订版，实际 %v", secondQ0["title"])
	}
}
