package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// isDuplicateEntry 识别 MySQL / SQLite 的唯一键冲突。
func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint failed")
}

// QuizService 实现知识点题集：题库维护、版本快照、幂等作答与错题复习。
type QuizService struct {
	repo   *repository.QuizRepository
	logger *slog.Logger
}

// NewQuizService creates a QuizService.
func NewQuizService(repo *repository.QuizRepository, logger *slog.Logger) *QuizService {
	return &QuizService{repo: repo, logger: logger}
}

// marshalOptions 序列化选项数组。
func marshalOptions(options []string) (string, error) {
	b, err := json.Marshal(options)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unmarshalOptions 解析选项数组。
func unmarshalOptions(raw string) []string {
	var options []string
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		return []string{}
	}
	return options
}

// buildSnapshot 把题目列表整体快照成不可变版本内容。
// 题面、选项、正确答案与解析都在发布时冻结，之后题库改动不再影响该版本。
func buildSnapshot(questions []model.QuizQuestion) (questionIDsJSON, snapshotJSON string, err error) {
	ids := make([]uint, 0, len(questions))
	snap := make([]dto.SnapshotQuestion, 0, len(questions))
	for _, q := range questions {
		ids = append(ids, q.ID)
		snap = append(snap, dto.SnapshotQuestion{
			QuestionID:  q.ID,
			Title:       q.Title,
			Options:     unmarshalOptions(q.Options),
			AnswerIndex: q.AnswerIndex,
			Explanation: q.Explanation,
		})
	}
	idBytes, err := json.Marshal(ids)
	if err != nil {
		return "", "", err
	}
	snapBytes, err := json.Marshal(snap)
	if err != nil {
		return "", "", err
	}
	return string(idBytes), string(snapBytes), nil
}

func parseSnapshot(raw string) []dto.SnapshotQuestion {
	var snap []dto.SnapshotQuestion
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return []dto.SnapshotQuestion{}
	}
	return snap
}

// validateQuestionRequest 校验题目请求。
func validateQuestionRequest(req *dto.QuizQuestionRequest) error {
	if !constants.IsValidQuizCategory(req.Category) {
		return util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("QuizQuestion[category=%s] invalid category", req.Category))
	}
	if len(req.Options) < 2 {
		return util.NewAppError(422, constants.CodeValidationError, "quiz question requires at least 2 options")
	}
	if req.AnswerIndex < 0 || req.AnswerIndex >= len(req.Options) {
		return util.NewAppError(422, constants.CodeValidationError, "quiz question answer_index out of range")
	}
	for _, o := range req.Options {
		if o == "" {
			return util.NewAppError(422, constants.CodeValidationError, "quiz question option must not be empty")
		}
	}
	return nil
}

// ---- 题库管理（管理员） ----

// CreateQuestion 新增题库题目，并级联发布所属分类题集的新版本。
func (s *QuizService) CreateQuestion(req *dto.QuizQuestionRequest) (*dto.QuizQuestionDTO, error) {
	if err := validateQuestionRequest(req); err != nil {
		return nil, err
	}
	optionsJSON, err := marshalOptions(req.Options)
	if err != nil {
		return nil, fmt.Errorf("quiz question options marshal: %w", err)
	}
	q := &model.QuizQuestion{
		Category:    req.Category,
		Title:       req.Title,
		Options:     optionsJSON,
		AnswerIndex: req.AnswerIndex,
		Explanation: req.Explanation,
	}
	err = s.repo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateQuestionTx(tx, q); err != nil {
			return fmt.Errorf("quiz question create: %w", err)
		}
		return s.publishCategoryVersionTx(tx, req.Category)
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info("quiz question created, new set version published", "id", q.ID, "category", q.Category)
	return s.toQuestionDTO(q), nil
}

// UpdateQuestion 更新题库题目（含修正答案），并级联发布受影响分类题集的新版本。
// 已开始或已完成的作答仍锁定旧版本，不受影响；未开始的题集下次开始即用新版本。
func (s *QuizService) UpdateQuestion(id uint, req *dto.QuizQuestionRequest) (*dto.QuizQuestionDTO, error) {
	if err := validateQuestionRequest(req); err != nil {
		return nil, err
	}
	q, err := s.repo.FindQuestion(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("QuizQuestion[id=%d] not found", id))
		}
		return nil, err
	}
	oldCategory := q.Category
	optionsJSON, err := marshalOptions(req.Options)
	if err != nil {
		return nil, fmt.Errorf("quiz question options marshal: %w", err)
	}
	q.Category = req.Category
	q.Title = req.Title
	q.Options = optionsJSON
	q.AnswerIndex = req.AnswerIndex
	q.Explanation = req.Explanation
	err = s.repo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.repo.UpdateQuestionTx(tx, q); err != nil {
			return fmt.Errorf("quiz question update: %w", err)
		}
		if err := s.publishCategoryVersionTx(tx, oldCategory); err != nil {
			return err
		}
		if oldCategory != req.Category {
			if err := s.publishCategoryVersionTx(tx, req.Category); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info("quiz question updated, new set version(s) published", "id", id, "old_category", oldCategory, "new_category", req.Category)
	return s.toQuestionDTO(q), nil
}

// ListQuestions 按分类列出题库题目。
func (s *QuizService) ListQuestions(category string) ([]dto.QuizQuestionDTO, error) {
	if category != "" && !constants.IsValidQuizCategory(category) {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("QuizQuestion[category=%s] invalid category", category))
	}
	items, err := s.repo.ListQuestions(category)
	if err != nil {
		return nil, fmt.Errorf("quiz question list: %w", err)
	}
	out := make([]dto.QuizQuestionDTO, 0, len(items))
	for i := range items {
		out = append(out, *s.toQuestionDTO(&items[i]))
	}
	return out, nil
}

func (s *QuizService) toQuestionDTO(q *model.QuizQuestion) *dto.QuizQuestionDTO {
	return &dto.QuizQuestionDTO{
		ID:          q.ID,
		Category:    q.Category,
		Title:       q.Title,
		Options:     unmarshalOptions(q.Options),
		AnswerIndex: q.AnswerIndex,
		Explanation: q.Explanation,
		CreatedAt:   q.CreatedAt,
		UpdatedAt:   q.UpdatedAt,
	}
}

// publishCategoryVersionTx 在事务内为某分类的激活题集发布新版本。
// 未找到题集（尚未播种）时跳过，不影响题库写入。
func (s *QuizService) publishCategoryVersionTx(tx *gorm.DB, category string) error {
	set, err := s.repo.FindSetByCategoryTx(tx, category)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	all, err := s.repo.ListQuestionsTx(tx, category)
	if err != nil {
		return err
	}
	if len(all) == 0 {
		return nil
	}
	questionIDs, snapshot, err := buildSnapshot(all)
	if err != nil {
		return err
	}
	if err := s.repo.PublishVersionForSetTx(tx, set, snapshot, questionIDs); err != nil {
		return fmt.Errorf("quiz set version publish: %w", err)
	}
	return nil
}

// PublishCategoryVersion 发布某分类题集的新版本（供播种与管理入口复用）。
func (s *QuizService) PublishCategoryVersion(category string) error {
	if !constants.IsValidQuizCategory(category) {
		return util.NewAppError(422, constants.CodeValidationError, "invalid quiz category")
	}
	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		return s.publishCategoryVersionTx(tx, category)
	})
}

// EnsureSeedData 播种三个知识点题集（若尚不存在）。
func (s *QuizService) EnsureSeedData(sets []model.QuizSet) error {
	for _, seed := range sets {
		_, err := s.repo.FindSetByCategory(seed.Category)
		if err == nil {
			continue
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		set := seed
		if err := s.repo.CreateSet(&set); err != nil {
			return fmt.Errorf("quiz set create: %w", err)
		}
	}
	return nil
}

// ---- 题集浏览 ----

// ListSets 返回题集列表及最新版本信息。
func (s *QuizService) ListSets() ([]dto.QuizSetDTO, error) {
	sets, err := s.repo.ListSets()
	if err != nil {
		return nil, fmt.Errorf("quiz set list: %w", err)
	}
	out := make([]dto.QuizSetDTO, 0, len(sets))
	for i := range sets {
		set := &sets[i]
		item := dto.QuizSetDTO{
			ID:          set.ID,
			Name:        set.Name,
			Category:    set.Category,
			Description: set.Description,
			CreatedAt:   set.CreatedAt,
		}
		if v, err := s.repo.LatestVersion(set.ID); err == nil {
			item.LatestVersion = v.Version
			item.QuestionCount = len(parseSnapshot(v.Snapshot))
		}
		out = append(out, item)
	}
	return out, nil
}

// ---- 作答尝试 ----

// StartAttempt 开始（或按同一尝试号幂等恢复）一次作答。
// 开始时锁定题集当前最新版本：题面、选项、答案在整个尝试期间保持不变。
func (s *QuizService) StartAttempt(userID uint, req *dto.StartAttemptRequest) (*dto.AttemptDTO, error) {
	set, err := s.repo.FindSet(req.SetID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("QuizSet[id=%d] not found", req.SetID))
		}
		return nil, err
	}
	// 幂等：同一尝试号直接返回已有尝试（写失败重试的场景）。
	if existing, err := s.repo.FindAttemptByNo(userID, req.AttemptNo); err == nil {
		if existing.SetID != req.SetID {
			return nil, util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("QuizAttempt[no=%s] already bound to another set", req.AttemptNo))
		}
		return s.buildAttemptDTO(existing)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	// 同一题集同时只允许一个进行中的尝试；该约束由下方事务加行锁保证，
	// 存在 open 时直接幂等返回，避免进度歧义。
	version, err := s.repo.LatestVersion(set.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(409, constants.CodeConflict, "quiz set has no published version")
		}
		return nil, err
	}
	attempt := &model.QuizAttempt{
		UserID:     userID,
		AttemptNo:  req.AttemptNo,
		SetID:      set.ID,
		VersionID:  version.ID,
		Status:     model.QuizAttemptOpen,
		TotalCount: len(parseSnapshot(version.Snapshot)),
		StartedAt:  time.Now(),
	}
	// 事务 + 行锁：并发的两个开始请求只会有一个创建 open 尝试，
	// 另一个在锁上等待后读到已有 open，返回冲突让客户端继续原尝试。
	err = s.repo.DB().Transaction(func(tx *gorm.DB) error {
		if open, lerr := s.repo.LockOpenAttemptTx(tx, userID, set.ID); lerr == nil {
			attempt = open
			return repository.ErrConflict
		} else if !errors.Is(lerr, repository.ErrNotFound) {
			return lerr
		}
		if cerr := s.repo.CreateAttemptTx(tx, attempt); cerr != nil {
			if isDuplicateEntry(cerr) {
				return repository.ErrConflict
			}
			return fmt.Errorf("quiz attempt create: %w", cerr)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			// 已有进行中的尝试（并发或重复点击）：返回该尝试供继续作答。
			if existing, ferr := s.repo.FindOpenAttempt(userID, set.ID); ferr == nil {
				return s.buildAttemptDTO(existing)
			}
			// 极端情况下另一请求恰好同 attempt_no 已交卷。
			if existing, ferr := s.repo.FindAttemptByNo(userID, req.AttemptNo); ferr == nil {
				return s.buildAttemptDTO(existing)
			}
			return nil, err
		}
		return nil, err
	}
	s.logger.Info("quiz attempt started", "user_id", userID, "set_id", set.ID, "version_id", version.ID, "attempt_no", req.AttemptNo)
	return s.buildAttemptDTO(attempt)
}

// GetAttempt 按尝试号查看尝试（本人）。
func (s *QuizService) GetAttempt(userID uint, attemptNo string) (*dto.AttemptDTO, error) {
	attempt, err := s.repo.FindAttemptByNo(userID, attemptNo)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("QuizAttempt[no=%s] not found", attemptNo))
		}
		return nil, err
	}
	return s.buildAttemptDTO(attempt)
}

// SubmitAttempt 交卷判分。同一尝试号重复交卷只认首次结果；
// 写入失败后按同一尝试号重试不会重复累计成绩与错题。
func (s *QuizService) SubmitAttempt(userID uint, req *dto.SubmitAttemptRequest) (*dto.SubmitResult, error) {
	attempt, err := s.repo.FindAttemptByNo(userID, req.AttemptNo)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("QuizAttempt[no=%s] not found", req.AttemptNo))
		}
		return nil, err
	}
	// 已交卷：幂等回放，成绩与错题不再重复累计。
	if attempt.Status == model.QuizAttemptDone {
		dtoView, err := s.buildAttemptDTO(attempt)
		if err != nil {
			return nil, err
		}
		return &dto.SubmitResult{Attempt: dtoView, FirstSubmitted: false}, nil
	}
	version, err := s.repo.FindVersion(attempt.VersionID)
	if err != nil {
		return nil, err
	}
	snapshot := parseSnapshot(version.Snapshot)
	answerByQuestion := make(map[uint]dto.SnapshotQuestion, len(snapshot))
	for _, q := range snapshot {
		answerByQuestion[q.QuestionID] = q
	}
	// 校验作答键均属于该版本、选项下标合法。
	for qid, selected := range req.Answers {
		q, ok := answerByQuestion[qid]
		if !ok {
			return nil, util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("question_id=%d not in attempt version snapshot", qid))
		}
		if selected < 0 || selected >= len(q.Options) {
			return nil, util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("question_id=%d selected option out of range", qid))
		}
	}
	rows := make([]model.QuizAttemptAnswer, 0, len(snapshot))
	score := 0
	wrongIDs := make([]uint, 0)
	for _, q := range snapshot {
		selected, answered := req.Answers[q.QuestionID]
		if !answered {
			selected = -1
		}
		correct := answered && selected == q.AnswerIndex
		if correct {
			score++
		} else {
			wrongIDs = append(wrongIDs, q.QuestionID)
		}
		rows = append(rows, model.QuizAttemptAnswer{
			AttemptID:  attempt.ID,
			QuestionID: q.QuestionID,
			Selected:   selected,
			Correct:    correct,
		})
	}
	now := time.Now()
	err = s.repo.DB().Transaction(func(tx *gorm.DB) error {
		// 行锁 + 状态复查，防止同一尝试号并发交卷：后到的事务会在锁上等待，
		// 拿到锁后读到 done 而中止（外层转为幂等回放）。
		var fresh model.QuizAttempt
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&fresh, attempt.ID).Error; err != nil {
			return err
		}
		if fresh.Status == model.QuizAttemptDone {
			return repository.ErrConflict
		}
		if err := s.repo.CreateAttemptAnswersTx(tx, rows); err != nil {
			// 兜底：(attempt_id, question_id) 唯一键冲突说明并发交卷已写入，按冲突回放。
			if isDuplicateEntry(err) {
				return repository.ErrConflict
			}
			return fmt.Errorf("quiz attempt answers create: %w", err)
		}
		fresh.Status = model.QuizAttemptDone
		fresh.Score = score
		fresh.TotalCount = len(snapshot)
		fresh.SubmittedAt = &now
		if err := tx.Save(&fresh).Error; err != nil {
			return err
		}
		for _, qid := range wrongIDs {
			if err := s.repo.UpsertReviewItemOnWrongTx(tx, userID, qid); err != nil {
				return fmt.Errorf("quiz review item upsert: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			latest, _ := s.repo.FindAttemptByNo(userID, req.AttemptNo)
			dtoView, _ := s.buildAttemptDTO(latest)
			return &dto.SubmitResult{Attempt: dtoView, FirstSubmitted: false}, nil
		}
		return nil, err
	}
	s.logger.Info("quiz attempt submitted", "user_id", userID, "attempt_no", req.AttemptNo, "score", score, "total", len(snapshot), "wrong", len(wrongIDs))
	updated, _ := s.repo.FindAttemptByNo(userID, req.AttemptNo)
	dtoView, err := s.buildAttemptDTO(updated)
	if err != nil {
		return nil, err
	}
	return &dto.SubmitResult{Attempt: dtoView, FirstSubmitted: true}, nil
}

// buildAttemptDTO 组装尝试视图。未交卷不返回正确答案；交卷后返回判分明细。
func (s *QuizService) buildAttemptDTO(attempt *model.QuizAttempt) (*dto.AttemptDTO, error) {
	set, err := s.repo.FindSet(attempt.SetID)
	if err != nil {
		return nil, err
	}
	version, err := s.repo.FindVersion(attempt.VersionID)
	if err != nil {
		return nil, err
	}
	snapshot := parseSnapshot(version.Snapshot)
	questions := make([]dto.AttemptQuestion, 0, len(snapshot))
	selectedMap := map[uint]model.QuizAttemptAnswer{}
	if attempt.Status == model.QuizAttemptDone {
		rows, err := s.repo.ListAttemptAnswers(attempt.ID)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			selectedMap[row.QuestionID] = row
		}
	}
	for _, q := range snapshot {
		aq := dto.AttemptQuestion{
			QuestionID: q.QuestionID,
			Title:      q.Title,
			Options:    q.Options,
		}
		if attempt.Status == model.QuizAttemptDone {
			answer := q.AnswerIndex
			aq.AnswerIndex = &answer
			aq.Explanation = q.Explanation
			if row, ok := selectedMap[q.QuestionID]; ok {
				selected := row.Selected
				correct := row.Correct
				aq.Selected = &selected
				aq.Correct = &correct
			}
		}
		questions = append(questions, aq)
	}
	return &dto.AttemptDTO{
		ID:          attempt.ID,
		AttemptNo:   attempt.AttemptNo,
		SetID:       attempt.SetID,
		SetName:     set.Name,
		VersionID:   version.ID,
		Version:     version.Version,
		Status:      attempt.Status,
		Score:       attempt.Score,
		TotalCount:  attempt.TotalCount,
		StartedAt:   attempt.StartedAt,
		SubmittedAt: attempt.SubmittedAt,
		Questions:   questions,
	}, nil
}

// ---- 错题复习 ----

// ListReview 列出当前用户的错题复习清单（未退出的条目）。
func (s *QuizService) ListReview(userID uint) ([]dto.ReviewItemDTO, error) {
	items, err := s.repo.ListReviewItems(userID)
	if err != nil {
		return nil, fmt.Errorf("quiz review list: %w", err)
	}
	out := make([]dto.ReviewItemDTO, 0, len(items))
	for _, item := range items {
		q, err := s.repo.FindQuestion(item.QuestionID)
		if err != nil {
			continue
		}
		out = append(out, dto.ReviewItemDTO{
			QuestionID:    q.ID,
			Title:         q.Title,
			Options:       unmarshalOptions(q.Options),
			AnswerIndex:   q.AnswerIndex,
			Explanation:   q.Explanation,
			CorrectStreak: item.CorrectStreak,
			UpdatedAt:     item.UpdatedAt,
		})
	}
	return out, nil
}

// AnswerReview 处理复习页单题作答。answer_no 为客户端幂等键：
// 写失败按同一 answer_no 重试返回同一结果，连对次数不重复累计。
// 错题连续答对两次才退出复习清单；答错则连对次数清零。
func (s *QuizService) AnswerReview(userID uint, req *dto.ReviewAnswerRequest) (*dto.ReviewAnswerResult, error) {
	// 幂等回放：同一 answer_no 已处理过，直接返回，不再更新连对次数。
	if record, err := s.repo.FindReviewAnswerByNo(userID, req.AnswerNo); err == nil {
		q, qerr := s.repo.FindQuestion(record.QuestionID)
		if qerr != nil {
			return nil, qerr
		}
		result := &dto.ReviewAnswerResult{Correct: record.Correct, AnswerIndex: q.AnswerIndex}
		if item, ierr := s.repo.FindReviewItem(userID, record.QuestionID); ierr == nil {
			result.CorrectStreak = item.CorrectStreak
			result.Resolved = false
		} else {
			result.CorrectStreak = constants.QuizReviewPassStreak
			result.Resolved = true
		}
		return result, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	question, err := s.repo.FindQuestion(req.QuestionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("QuizQuestion[id=%d] not found", req.QuestionID))
		}
		return nil, err
	}
	item, err := s.repo.FindReviewItem(userID, req.QuestionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(409, constants.CodeConflict, "question is not in the review list")
		}
		return nil, err
	}
	correct := req.Selected >= 0 && req.Selected < len(unmarshalOptions(question.Options)) && req.Selected == question.AnswerIndex
	err = s.repo.DB().Transaction(func(tx *gorm.DB) error {
		record := &model.QuizReviewAnswer{
			UserID:     userID,
			AnswerNo:   req.AnswerNo,
			QuestionID: req.QuestionID,
			Selected:   req.Selected,
			Correct:    correct,
		}
		if err := s.repo.CreateReviewAnswerTx(tx, record); err != nil {
			if isDuplicateEntry(err) {
				return repository.ErrConflict
			}
			return fmt.Errorf("quiz review answer create: %w", err)
		}
		// 行锁复查复习清单，避免与交卷/其他复习请求并发时覆盖连对计数。
		var freshItem model.QuizReviewItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&freshItem, item.ID).Error; err != nil {
			return err
		}
		item = &freshItem
		if correct {
			item.CorrectStreak++
			if item.CorrectStreak >= constants.QuizReviewPassStreak {
				item.Resolved = true
			}
		} else {
			item.CorrectStreak = 0
			item.LastWrongAt = time.Now()
		}
		return s.repo.SaveReviewItemTx(tx, item)
	})
	if err != nil {
		// 并发下同一 answer_no 已被另一请求处理：回放首次结果，不重复累计。
		if errors.Is(err, repository.ErrConflict) {
			if record, rerr := s.repo.FindReviewAnswerByNo(userID, req.AnswerNo); rerr == nil {
				q, qerr := s.repo.FindQuestion(record.QuestionID)
				if qerr != nil {
					return nil, qerr
				}
				result := &dto.ReviewAnswerResult{Correct: record.Correct, AnswerIndex: q.AnswerIndex}
				if ri, ierr := s.repo.FindReviewItem(userID, record.QuestionID); ierr == nil {
					result.CorrectStreak = ri.CorrectStreak
				} else {
					result.CorrectStreak = constants.QuizReviewPassStreak
					result.Resolved = true
				}
				return result, nil
			}
		}
		return nil, err
	}
	s.logger.Info("quiz review answered", "user_id", userID, "question_id", req.QuestionID, "correct", correct, "streak", item.CorrectStreak, "resolved", item.Resolved)
	return &dto.ReviewAnswerResult{
		Correct:       correct,
		CorrectStreak: item.CorrectStreak,
		Resolved:      item.Resolved,
		AnswerIndex:   question.AnswerIndex,
	}, nil
}

// ---- 进度（首页与复习页读取同一份结果） ----

// Progress 返回当前用户各题集进度、错题数与累计成绩。
// 成绩只统计每个题集首次完成的尝试，重考不重复累计。
func (s *QuizService) Progress(userID uint) (*dto.QuizProgress, error) {
	sets, err := s.repo.ListSets()
	if err != nil {
		return nil, err
	}
	progress := &dto.QuizProgress{Sets: []dto.SetProgress{}}
	for _, set := range sets {
		sp := dto.SetProgress{SetID: set.ID, Status: "not_started"}
		if done, err := s.repo.FirstDoneAttempt(userID, set.ID); err == nil {
			if v, verr := s.repo.FindVersion(done.VersionID); verr == nil {
				sp.Version = v.Version
			}
			sp.VersionID = done.VersionID
			sp.Status = model.QuizAttemptDone
			sp.Score = done.Score
			sp.TotalCount = done.TotalCount
			sp.AttemptID = done.ID
			sp.AttemptNo = done.AttemptNo
			progress.TotalScore += done.Score
			progress.TotalQuestions += done.TotalCount
		} else if errors.Is(err, repository.ErrNotFound) {
			if open, oerr := s.repo.FindOpenAttempt(userID, set.ID); oerr == nil {
				if v, verr := s.repo.FindVersion(open.VersionID); verr == nil {
					sp.Version = v.Version
				}
				sp.VersionID = open.VersionID
				sp.Status = model.QuizAttemptOpen
				sp.TotalCount = open.TotalCount
				sp.AttemptID = open.ID
				sp.AttemptNo = open.AttemptNo
			}
		} else {
			return nil, err
		}
		progress.Sets = append(progress.Sets, sp)
	}
	review, err := s.repo.ListReviewItems(userID)
	if err != nil {
		return nil, err
	}
	progress.ReviewCount = len(review)
	return progress, nil
}
