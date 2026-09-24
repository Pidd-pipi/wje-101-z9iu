package service

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/repository"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.User{}, &model.TastingNote{}, &model.BrewRecipe{},
		&model.CoffeeBean{}, &model.Comment{}, &model.Like{},
		&model.UserFollow{}, &model.UserBeanList{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

type beanListFixture struct {
	db       *gorm.DB
	noteSvc  *NoteService
	listSvc  *UserBeanListService
	beanRepo *repository.CoffeeBeanRepository
	noteRepo *repository.TastingNoteRepository
	listRepo *repository.UserBeanListRepository
}

func newBeanListFixture(t *testing.T) *beanListFixture {
	db := newTestDB(t)
	beanRepo := repository.NewCoffeeBeanRepository(db)
	noteRepo := repository.NewTastingNoteRepository(db)
	listRepo := repository.NewUserBeanListRepository(db)
	logger := newTestLogger()
	return &beanListFixture{
		db:       db,
		beanRepo: beanRepo,
		noteRepo: noteRepo,
		listRepo: listRepo,
		noteSvc:  NewNoteService(noteRepo, beanRepo, listRepo, logger),
		listSvc:  NewUserBeanListService(listRepo, beanRepo, noteRepo, logger),
	}
}

func createTestBean(t *testing.T, f *beanListFixture, name string) *model.CoffeeBean {
	t.Helper()
	b := &model.CoffeeBean{Name: name, Origin: "埃塞俄比亚", ProcessMethod: "washed", FlavorTags: "[]"}
	if err := f.beanRepo.Create(b); err != nil {
		t.Fatalf("create bean: %v", err)
	}
	return b
}

func TestBeanListLifecycle(t *testing.T) {
	f := newBeanListFixture(t)
	const userID uint = 7
	bean := createTestBean(t, f, "测试豆A")
	other := createTestBean(t, f, "测试豆B")

	cases := []struct {
		name       string
		run        func() error
		wantErr    bool
		wantStatus int
		check      func(t *testing.T)
	}{
		{
			name: "add puts bean on want list, idempotent",
			run: func() error {
				if err := f.listSvc.Add(userID, bean.ID); err != nil {
					return err
				}
				return f.listSvc.Add(userID, bean.ID)
			},
			check: func(t *testing.T) {
				groups, err := f.listSvc.GetUserBeanGroups(userID)
				if err != nil {
					t.Fatalf("groups: %v", err)
				}
				if len(groups.Want) != 1 || groups.Want[0].ID != bean.ID {
					t.Fatalf("want groups = %+v", groups.Want)
				}
				if len(groups.Drunk) != 0 {
					t.Fatalf("drunk groups = %+v", groups.Drunk)
				}
			},
		},
		{
			name: "publishing a note moves bean to drunk and tracks it",
			run: func() error {
				_, err := f.noteSvc.Create(userID, &model.TastingNote{
					CoffeeName: other.Name, RoastLevel: "light", OverallScore: 8.0,
				})
				return err
			},
			check: func(t *testing.T) {
				groups, _ := f.listSvc.GetUserBeanGroups(userID)
				if len(groups.Drunk) != 1 || groups.Drunk[0].ID != other.ID || groups.Drunk[0].MyNoteCount != 1 {
					t.Fatalf("drunk groups = %+v", groups.Drunk)
				}
				if len(groups.Want) != 1 || groups.Want[0].ID != bean.ID {
					t.Fatalf("want groups = %+v", groups.Want)
				}
			},
		},
		{
			name: "a second note accumulates the count",
			run: func() error {
				_, err := f.noteSvc.Create(userID, &model.TastingNote{
					CoffeeName: other.Name, RoastLevel: "medium", OverallScore: 7.0,
				})
				return err
			},
			check: func(t *testing.T) {
				groups, _ := f.listSvc.GetUserBeanGroups(userID)
				if groups.Drunk[0].MyNoteCount != 2 {
					t.Fatalf("drunk count = %d, want 2", groups.Drunk[0].MyNoteCount)
				}
			},
		},
		{
			name:       "removing a drunk bean is rejected with conflict",
			run:        func() error { return f.listSvc.Remove(userID, other.ID) },
			wantErr:    true,
			wantStatus: 409,
		},
		{
			name: "switching the note's bean decrements the old count",
			run: func() error {
				notes, err := f.noteRepo.ListByUserAndBeanName(userID, other.Name)
				if err != nil {
					return err
				}
				target := notes[0]
				_, err = f.noteSvc.Update(userID, target.ID, &model.TastingNote{
					CoffeeName: bean.Name, RoastLevel: "dark",
				})
				return err
			},
			check: func(t *testing.T) {
				groups, _ := f.listSvc.GetUserBeanGroups(userID)
				// other.Name drops from 2 to 1 (still drunk); bean.Name gains a note (now drunk, out of want)
				if len(groups.Want) != 0 {
					t.Fatalf("want groups = %+v", groups.Want)
				}
				counts := map[uint]int64{}
				for _, d := range groups.Drunk {
					counts[d.ID] = d.MyNoteCount
				}
				if counts[bean.ID] != 1 || counts[other.ID] != 1 {
					t.Fatalf("drunk groups = %+v", groups.Drunk)
				}
			},
		},
		{
			name: "deleting all notes of a bean returns it to want",
			run: func() error {
				notes, err := f.noteRepo.ListByUserAndBeanName(userID, other.Name)
				if err != nil {
					return err
				}
				for _, n := range notes {
					if err := f.noteSvc.Delete(userID, n.ID); err != nil {
						return err
					}
				}
				return nil
			},
			check: func(t *testing.T) {
				groups, _ := f.listSvc.GetUserBeanGroups(userID)
				if len(groups.Want) != 1 || groups.Want[0].ID != other.ID {
					t.Fatalf("want groups = %+v", groups.Want)
				}
				if len(groups.Drunk) != 1 || groups.Drunk[0].ID != bean.ID {
					t.Fatalf("drunk groups = %+v", groups.Drunk)
				}
			},
		},
		{
			name: "remove now moves the zero-note bean out of the list",
			run:  func() error { return f.listSvc.Remove(userID, other.ID) },
			check: func(t *testing.T) {
				groups, _ := f.listSvc.GetUserBeanGroups(userID)
				for _, w := range groups.Want {
					if w.ID == other.ID {
						t.Fatalf("other bean still in want list: %+v", groups.Want)
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var appErr *util.AppError
				if errors.As(err, &appErr) && appErr.HTTPStatus != tc.wantStatus {
					t.Fatalf("status = %d, want %d", appErr.HTTPStatus, tc.wantStatus)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t)
			}
		})
	}
}

func TestAddUnknownBeanFails(t *testing.T) {
	f := newBeanListFixture(t)
	err := f.listSvc.Add(1, 9999)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != 404 {
		t.Fatalf("err = %v, want 404 AppError", err)
	}
}
