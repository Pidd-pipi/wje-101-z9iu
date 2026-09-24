package router

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
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/config"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

func newE2EEnv(t *testing.T) (*gorm.DB, http.Handler, string, uint) {
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

	cfg := &config.Config{
		CORSOrigins:  "http://localhost:28601",
		JWTSecret:    "e2e-test-secret",
		JWTExpire:    time.Hour,
		RateLimitReq: 100000,
		RateLimitWin: time.Minute,
		UploadDir:    t.TempDir(),
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	h := Setup(cfg, db, logger)

	user := &model.User{Username: "taster", Email: "t@example.com", PasswordHash: "x", Role: "user"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	bean := &model.CoffeeBean{Name: "瑰夏E2E", Origin: "巴拿马", ProcessMethod: "washed", FlavorTags: `["茉莉"]`}
	if err := db.Create(bean).Error; err != nil {
		t.Fatalf("create bean: %v", err)
	}
	token, err := util.GenerateToken(user.ID, user.Username, user.Role, cfg.JWTSecret, cfg.JWTExpire)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return db, h, token, bean.ID
}

type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"message"`
	Data json.RawMessage `json:"data"`
}

func doRequest(t *testing.T, h http.Handler, method, path, token string, body any) (int, envelope) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var env envelope
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &env)
	}
	return w.Code, env
}

func beanListData(t *testing.T, env envelope) []map[string]any {
	t.Helper()
	var page struct {
		List []map[string]any `json:"list"`
	}
	if err := json.Unmarshal(env.Data, &page); err != nil {
		t.Fatalf("unmarshal page: %v", err)
	}
	return page.List
}

func groupData(t *testing.T, env envelope) map[string][]map[string]any {
	t.Helper()
	var g map[string][]map[string]any
	if err := json.Unmarshal(env.Data, &g); err != nil {
		t.Fatalf("unmarshal groups: %v", err)
	}
	return g
}

func findBean(list []map[string]any, name string) map[string]any {
	for _, b := range list {
		if b["name"] == name {
			return b
		}
	}
	return nil
}

func TestBeanListHTTPLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, h, token, beanID := newE2EEnv(t)
	beanPath := "/api/v1/beans/" + strconv.FormatUint(uint64(beanID), 10)

	// 1. Anonymous browsing is allowed and carries the public tally.
	status, env := doRequest(t, h, http.MethodGet, "/api/v1/beans?page_size=20", "", nil)
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("anonymous list status=%d code=%d msg=%s", status, env.Code, env.Msg)
	}
	if b := findBean(beanListData(t, env), "瑰夏E2E"); b == nil {
		t.Fatal("bean missing from public list")
	} else if b["note_count"].(float64) != 0 {
		t.Fatalf("note_count = %v, want 0", b["note_count"])
	}

	// 2. Adding without a token is rejected.
	status, _ = doRequest(t, h, http.MethodPost, beanPath+"/list", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous add status = %d, want 401", status)
	}

	// 3. Authenticated add succeeds and is idempotent.
	status, env = doRequest(t, h, http.MethodPost, beanPath+"/list", token, nil)
	if status != http.StatusCreated || env.Code != 0 {
		t.Fatalf("add status=%d code=%d msg=%s", status, env.Code, env.Msg)
	}
	status, _ = doRequest(t, h, http.MethodPost, beanPath+"/list", token, nil)
	if status != http.StatusCreated {
		t.Fatalf("idempotent add status = %d, want 201", status)
	}

	// 4. The bean shows in the want group, not drunk.
	status, env = doRequest(t, h, http.MethodGet, "/api/v1/users/1/beans", token, nil)
	if status != http.StatusOK {
		t.Fatalf("groups status = %d", status)
	}
	groups := groupData(t, env)
	if findBean(groups["want"], "瑰夏E2E") == nil {
		t.Fatalf("bean not in want: %+v", groups["want"])
	}
	if findBean(groups["drunk"], "瑰夏E2E") != nil {
		t.Fatalf("bean unexpectedly drunk: %+v", groups["drunk"])
	}

	// 5. Library list flags in_list for this viewer.
	_, env = doRequest(t, h, http.MethodGet, "/api/v1/beans?page_size=20", token, nil)
	if b := findBean(beanListData(t, env), "瑰夏E2E"); b["in_list"] != true {
		t.Fatalf("in_list = %v, want true", b["in_list"])
	}

	// 6. Publishing a note for this bean moves it to drunk with count 1.
	noteBody := map[string]any{"coffee_name": "瑰夏E2E", "roast_level": "light", "overall_score": 8.5}
	status, env = doRequest(t, h, http.MethodPost, "/api/v1/notes", token, noteBody)
	if status != http.StatusCreated || env.Code != 0 {
		t.Fatalf("create note status=%d code=%d msg=%s", status, env.Code, env.Msg)
	}
	var created struct {
		ID uint `json:"id"`
	}
	_ = json.Unmarshal(env.Data, &created)
	if created.ID == 0 {
		t.Fatal("missing created note id")
	}

	_, env = doRequest(t, h, http.MethodGet, "/api/v1/users/1/beans", token, nil)
	groups = groupData(t, env)
	d := findBean(groups["drunk"], "瑰夏E2E")
	if d == nil || d["my_note_count"].(float64) != 1 {
		t.Fatalf("drunk group = %+v", groups["drunk"])
	}
	if findBean(groups["want"], "瑰夏E2E") != nil {
		t.Fatalf("bean still in want after note: %+v", groups["want"])
	}

	// 7. Removing a drunk bean is rejected with 409 (state preserved).
	status, _ = doRequest(t, h, http.MethodDelete, beanPath+"/list", token, nil)
	if status != http.StatusConflict {
		t.Fatalf("remove drunk status = %d, want 409", status)
	}

	// 8. Deleting the note brings the count to zero and returns it to want.
	status, _ = doRequest(t, h, http.MethodDelete, "/api/v1/notes/"+strconv.FormatUint(uint64(created.ID), 10), token, nil)
	if status != http.StatusOK {
		t.Fatalf("delete note status = %d, want 200", status)
	}
	_, env = doRequest(t, h, http.MethodGet, "/api/v1/users/1/beans", token, nil)
	groups = groupData(t, env)
	if findBean(groups["want"], "瑰夏E2E") == nil {
		t.Fatalf("bean did not return to want: %+v", groups["want"])
	}

	// 9. Removing now succeeds and empties the want entry.
	status, _ = doRequest(t, h, http.MethodDelete, beanPath+"/list", token, nil)
	if status != http.StatusOK {
		t.Fatalf("remove want status = %d, want 200", status)
	}
	_, env = doRequest(t, h, http.MethodGet, "/api/v1/users/1/beans", token, nil)
	groups = groupData(t, env)
	if findBean(groups["want"], "瑰夏E2E") != nil {
		t.Fatalf("bean still in want after remove: %+v", groups["want"])
	}
}

// TestBeanSwitchOverHTTP verifies that editing a note to select another bean
// decrements the old bean's tally and moves it back to want, while the new
// bean becomes drunk.
func TestBeanSwitchOverHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, h, token, _ := newE2EEnv(t)
	beanB := &model.CoffeeBean{Name: "翡翠E2E", Origin: "哥伦比亚", ProcessMethod: "natural", FlavorTags: `[]`}
	if err := db.Create(beanB).Error; err != nil {
		t.Fatalf("create bean B: %v", err)
	}

	noteBody := map[string]any{"coffee_name": "瑰夏E2E", "roast_level": "light", "overall_score": 8.5}
	status, env := doRequest(t, h, http.MethodPost, "/api/v1/notes", token, noteBody)
	if status != http.StatusCreated {
		t.Fatalf("create note status=%d msg=%s", status, env.Msg)
	}
	var created struct {
		ID uint `json:"id"`
	}
	_ = json.Unmarshal(env.Data, &created)

	// Initially A is drunk, B is nowhere.
	_, env = doRequest(t, h, http.MethodGet, "/api/v1/users/1/beans", token, nil)
	groups := groupData(t, env)
	if findBean(groups["drunk"], "瑰夏E2E") == nil {
		t.Fatalf("A should be drunk: %+v", groups["drunk"])
	}

	// Edit the note and switch to bean B.
	updateBody := map[string]any{"coffee_name": "翡翠E2E", "roast_level": "medium", "origin": "哥伦比亚", "overall_score": 7.5}
	notePath := "/api/v1/notes/" + strconv.FormatUint(uint64(created.ID), 10)
	status, env = doRequest(t, h, http.MethodPut, notePath, token, updateBody)
	if status != http.StatusOK {
		t.Fatalf("update note status=%d msg=%s", status, env.Msg)
	}

	_, env = doRequest(t, h, http.MethodGet, "/api/v1/users/1/beans", token, nil)
	groups = groupData(t, env)
	if findBean(groups["want"], "瑰夏E2E") == nil {
		t.Fatalf("old bean A should return to want: %+v", groups["want"])
	}
	if d := findBean(groups["drunk"], "翡翠E2E"); d == nil || d["my_note_count"].(float64) != 1 {
		t.Fatalf("new bean B should be drunk with count 1: %+v", groups["drunk"])
	}
}
