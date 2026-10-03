package main

import (
	"encoding/json"
	"log/slog"

	"gorm.io/gorm"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// seedQuiz 播种知识点题库与三个题集（品种/文章/病虫害），
// 并为每个题集发布 v1 快照。内容来源于平台已有的品种、文章与病虫害资料。
func seedQuiz(db *gorm.DB, logger *slog.Logger) error {
	var questionCount int64
	if err := db.Model(&model.QuizQuestion{}).Count(&questionCount).Error; err != nil {
		return err
	}
	var setCount int64
	if err := db.Model(&model.QuizSet{}).Count(&setCount).Error; err != nil {
		return err
	}

	if setCount == 0 {
		sets := []model.QuizSet{
			{Name: "植物品种知识点", Category: constants.QuizCategoryPlant, Description: "科属、习性、光照温度与浇水要点", Status: model.QuizSetActive},
			{Name: "养护文章知识点", Category: constants.QuizCategoryArticle, Description: "换盆、施肥、修剪、病虫害防治与繁殖", Status: model.QuizSetActive},
			{Name: "病虫害识别与防治", Category: constants.QuizCategoryPest, Description: "症状识别、发病原因与用药处理", Status: model.QuizSetActive},
		}
		if err := db.Create(&sets).Error; err != nil {
			return err
		}
	}

	if questionCount == 0 {
		if err := db.Create(&seedQuizQuestions).Error; err != nil {
			return err
		}
	}

	// 为尚没有版本的题集发布初始版本（题目表已有数据但版本缺失时幂等补齐）。
	sets, err := quizSetRepo(db)
	if err != nil {
		return err
	}
	for _, set := range sets {
		var versionCount int64
		if err := db.Model(&model.QuizSetVersion{}).Where("set_id = ?", set.ID).Count(&versionCount).Error; err != nil {
			return err
		}
		if versionCount > 0 {
			continue
		}
		if err := publishQuizSeedVersion(db, set); err != nil {
			return err
		}
		logger.Info("quiz seed version published", "set_id", set.ID, "category", set.Category)
	}
	return nil
}

// quizSetRepo 返回全部题集（播种阶段不限状态）。
func quizSetRepo(db *gorm.DB) ([]model.QuizSet, error) {
	var sets []model.QuizSet
	if err := db.Order("id ASC").Find(&sets).Error; err != nil {
		return nil, err
	}
	return sets, nil
}

// publishQuizSeedVersion 按分类快照题目并发布新版本。
func publishQuizSeedVersion(db *gorm.DB, set model.QuizSet) error {
	var questions []model.QuizQuestion
	if err := db.Where("category = ?", set.Category).Order("id ASC").Find(&questions).Error; err != nil {
		return err
	}
	if len(questions) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(questions))
	type snapQuestion struct {
		QuestionID  uint     `json:"question_id"`
		Title       string   `json:"title"`
		Options     []string `json:"options"`
		AnswerIndex int      `json:"answer_index"`
		Explanation string   `json:"explanation"`
	}
	snap := make([]snapQuestion, 0, len(questions))
	for _, q := range questions {
		ids = append(ids, q.ID)
		var options []string
		_ = json.Unmarshal([]byte(q.Options), &options)
		snap = append(snap, snapQuestion{
			QuestionID:  q.ID,
			Title:       q.Title,
			Options:     options,
			AnswerIndex: q.AnswerIndex,
			Explanation: q.Explanation,
		})
	}
	idBytes, _ := json.Marshal(ids)
	snapBytes, _ := json.Marshal(snap)
	version := &model.QuizSetVersion{
		SetID:       set.ID,
		Version:     1,
		QuestionIDs: string(idBytes),
		Snapshot:    string(snapBytes),
	}
	return db.Create(version).Error
}

// mustOptions 序列化选项数组。
func mustOptions(options []string) string {
	b, _ := json.Marshal(options)
	return string(b)
}

// seedQuizQuestions 题库初始题目，答案与解析均对应平台现有资料。
var seedQuizQuestions = []model.QuizQuestion{
	// ---- 植物品种知识点（6 题）----
	{
		Category:    constants.QuizCategoryPlant,
		Title:       "以下哪种植物属于多肉植物？",
		Options:     mustOptions([]string{"月季", "吉娃娃", "碗莲", "龟背竹"}),
		AnswerIndex: 1,
		Explanation: "吉娃娃（多肉吉娃娃）为景天科拟石莲属多肉植物。",
	},
	{
		Category:    constants.QuizCategoryPlant,
		Title:       "龟背竹适合的光照条件是？",
		Options:     mustOptions([]string{"全日照暴晒", "明亮散射光", "完全黑暗", "强直射光"}),
		AnswerIndex: 1,
		Explanation: "龟背竹耐阴，适合明亮散射光环境，忌强光直射。",
	},
	{
		Category:    constants.QuizCategoryPlant,
		Title:       "碗莲属于哪一类植物？",
		Options:     mustOptions([]string{"多肉植物", "观叶植物", "水生植物", "观花灌木"}),
		AnswerIndex: 2,
		Explanation: "碗莲是睡莲科睡莲属的小型水生花卉，栽培需保持水位。",
	},
	{
		Category:    constants.QuizCategoryPlant,
		Title:       "库拉索芦荟的适宜生长温度范围是？",
		Options:     mustOptions([]string{"-5～5℃", "10～32℃", "30～45℃", "0～10℃"}),
		AnswerIndex: 1,
		Explanation: "库拉索芦荟适宜温度为 10～32℃，不耐霜冻。",
	},
	{
		Category:    constants.QuizCategoryPlant,
		Title:       "月季所属的科是？",
		Options:     mustOptions([]string{"蔷薇科", "天南星科", "景天科", "百合科"}),
		AnswerIndex: 0,
		Explanation: "月季为蔷薇科月季属植物，喜全日照。",
	},
	{
		Category:    constants.QuizCategoryPlant,
		Title:       "“清香木”叶片的显著特征是？",
		Options:     mustOptions([]string{"揉碎有清香气味", "叶面有白色绒毛", "叶片全年红色", "叶片肉质肥厚储水"}),
		AnswerIndex: 0,
		Explanation: "清香木为柏科圆柏属常绿灌木，叶片揉碎有清香。",
	},

	// ---- 养护文章知识点（6 题）----
	{
		Category:    constants.QuizCategoryArticle,
		Title:       "换盆的最佳季节通常是？",
		Options:     mustOptions([]string{"盛夏", "深冬", "春季", "雨季"}),
		AnswerIndex: 2,
		Explanation: "春季气温回升、根系活跃，是换盆的最佳时机；换盆前宜停浇 3 天。",
	},
	{
		Category:    constants.QuizCategoryArticle,
		Title:       "多肉植物生长季施肥的原则是？",
		Options:     mustOptions([]string{"浓肥勤施", "薄肥勤施，每月一次稀释液肥", "只施氮肥", "冬季大量施肥"}),
		AnswerIndex: 1,
		Explanation: "多肉施肥宜稀薄，生长季每月一次稀释液肥，休眠期停止施肥，避免肥害烧根。",
	},
	{
		Category:    constants.QuizCategoryArticle,
		Title:       "月季夏季修剪应以什么方式为主？",
		Options:     mustOptions([]string{"重剪到基部", "轻剪，剪除残花和细弱枝", "剪除所有叶片", "不做任何修剪"}),
		AnswerIndex: 1,
		Explanation: "月季夏季以轻剪为主，剪除残花和细弱枝，保留健壮枝条促进复花。",
	},
	{
		Category:    constants.QuizCategoryArticle,
		Title:       "龟背竹扦插繁殖时，枝条最好带有什么结构？",
		Options:     mustOptions([]string{"气生根", "花序", "块根", "种荚"}),
		AnswerIndex: 0,
		Explanation: "选取带气生根的健壮枝条，切口晾干后插入湿润蛭石，约三周生根。",
	},
	{
		Category:    constants.QuizCategoryArticle,
		Title:       "“见干见湿”的浇水原则适用于？",
		Options:     mustOptions([]string{"所有植物", "仅多肉植物", "绝大多数盆栽植物", "水生植物"}),
		AnswerIndex: 2,
		Explanation: "绝大多数盆栽植物遵循见干见湿，水生植物则需持续保持水位。",
	},
	{
		Category:    constants.QuizCategoryArticle,
		Title:       "春季换盆时，新盆的合适尺寸通常是？",
		Options:     mustOptions([]string{"和原盆一样大", "比原盆大 1-2 号", "越大越好", "比原盆小一号"}),
		AnswerIndex: 1,
		Explanation: "换盆宜选比原盆大 1-2 号的透气花盆，底部垫陶粒排水层，并修剪烂根消毒。",
	},

	// ---- 病虫害识别与防治（6 题）----
	{
		Category:    constants.QuizCategoryPest,
		Title:       "月季黑斑病的典型症状是？",
		Options:     mustOptions([]string{"叶片覆盖白粉", "黑色圆形斑点，边缘放射状", "叶背有蛛网", "叶片纵向卷曲"}),
		AnswerIndex: 1,
		Explanation: "黑斑病叶片出现黑色圆形斑点、边缘放射状，严重时落叶，高湿通风差易诱发。",
	},
	{
		Category:    constants.QuizCategoryPest,
		Title:       "多肉叶腋出现白色棉絮状物、叶片发黏发黄，最可能是？",
		Options:     mustOptions([]string{"介壳虫", "黑斑病", "红蜘蛛", "日灼"}),
		AnswerIndex: 0,
		Explanation: "这是多肉介壳虫的典型表现，可人工刮除后用酒精擦拭，严重时喷噻嗪酮。",
	},
	{
		Category:    constants.QuizCategoryPest,
		Title:       "红蜘蛛危害时，叶背常出现什么？",
		Options:     mustOptions([]string{"黑色圆形病斑", "白色棉絮", "细密蛛网与黄白色斑点", "透明黏液滴"}),
		AnswerIndex: 2,
		Explanation: "红蜘蛛在干燥高温下滋生，叶面出现细密黄白斑点、叶背有蛛网，可喷阿维菌素。",
	},
	{
		Category:    constants.QuizCategoryPest,
		Title:       "龟背竹叶斑病的主要诱因是？",
		Options:     mustOptions([]string{"浇水过多、长期积水", "光照过强", "施肥不足", "温度过低冻伤"}),
		AnswerIndex: 0,
		Explanation: "叶斑病由浇水过多、长期积水导致真菌感染，应控水通风、剪除病叶并喷多菌灵。",
	},
	{
		Category:    constants.QuizCategoryPest,
		Title:       "防治月季黑斑病可选用的药剂是？",
		Options:     mustOptions([]string{"代森锰锌或苯醚甲环唑", "赤霉素", "生根粉", "尿素"}),
		AnswerIndex: 0,
		Explanation: "月季黑斑病可喷施代森锰锌或苯醚甲环唑，每周一次连续 2-3 次，并及时摘除病叶。",
	},
	{
		Category:    constants.QuizCategoryPest,
		Title:       "发现少量介壳虫时，最直接的家庭处理方式是？",
		Options:     mustOptions([]string{"大量浇水冲淋", "用酒精棉擦拭刮除", "放在太阳下暴晒", "立即重剪到根部"}),
		AnswerIndex: 1,
		Explanation: "少量介壳虫可人工刮除后用酒精棉擦拭，严重时再用矿物油乳剂或噻嗪酮。",
	},
}
