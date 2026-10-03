package main

import (
	"io"
	"log/slog"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

func newSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:seed_"+t.Name()+"?mode=memory"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestSeedQuizCreatesThreeSetsWithVersions(t *testing.T) {
	db := newSeedTestDB(t)
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := seedQuiz(db, testLogger); err != nil {
		t.Fatalf("seedQuiz: %v", err)
	}

	var sets []model.QuizSet
	if err := db.Order("id ASC").Find(&sets).Error; err != nil {
		t.Fatalf("query sets: %v", err)
	}
	if len(sets) != 3 {
		t.Fatalf("应播种 3 个题集，实际 %d", len(sets))
	}

	var questionCount int64
	db.Model(&model.QuizQuestion{}).Count(&questionCount)
	if questionCount != 18 {
		t.Fatalf("应播种 18 道题（3 册 x 6 题），实际 %d", questionCount)
	}

	for _, set := range sets {
		var v model.QuizSetVersion
		if err := db.Where("set_id = ?", set.ID).First(&v).Error; err != nil {
			t.Fatalf("题集 %s 缺少 v1 版本: %v", set.Category, err)
		}
		if v.Version != 1 {
			t.Fatalf("题集 %s 初始版本应为 1，实际 %d", set.Category, v.Version)
		}
		var n int64
		db.Model(&model.QuizQuestion{}).Where("category = ?", set.Category).Count(&n)
		if n != 6 {
			t.Fatalf("题集 %s 应含 6 题，实际 %d", set.Category, n)
		}
		if len(v.Snapshot) == 0 || v.Snapshot[0] != '[' {
			t.Fatalf("题集 %s 快照格式异常", set.Category)
		}
	}

	// 幂等：再次执行不应重复创建题集或版本。
	if err := seedQuiz(db, testLogger); err != nil {
		t.Fatalf("seedQuiz 二次执行: %v", err)
	}
	var setCount, versionCount int64
	db.Model(&model.QuizSet{}).Count(&setCount)
	db.Model(&model.QuizSetVersion{}).Count(&versionCount)
	if setCount != 3 || versionCount != 3 {
		t.Fatalf("二次播种后题集/版本数应不变，实际 sets=%d versions=%d", setCount, versionCount)
	}
}
