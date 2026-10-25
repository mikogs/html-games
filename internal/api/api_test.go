package api

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	htmlgames "github.com/mikogs/html-games"
	"github.com/mikogs/html-games/internal/catalog"
	"github.com/mikogs/html-games/internal/store"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := store.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	reg := store.NewRegistry(db)
	files, _ := fs.Sub(htmlgames.Questions, "questions")
	if _, err := catalog.Sync(ctx, reg, files); err != nil {
		t.Fatal(err)
	}
	h, err := New(db, reg, Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

type response struct {
	status int
	OK     bool              `json:"ok"`
	Code   string            `json:"code"`
	Data   json.RawMessage   `json:"data"`
	Errors map[string]string `json:"errors"`
}

func call(t *testing.T, srv *httptest.Server, method, path, body string) response {
	t.Helper()
	req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out := response{status: res.StatusCode}
	raw, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s %s: %d %s", method, path, res.StatusCode, raw)
	}
	return out
}

func addScore(t *testing.T, srv *httptest.Server, body string) uint64 {
	t.Helper()
	res := call(t, srv, http.MethodPut, "/api/v1/scores/", body)
	if res.status != http.StatusCreated {
		t.Fatalf("PUT %s: %d %s %v", body, res.status, res.Code, res.Errors)
	}
	var data struct{ ID uint64 }
	_ = json.Unmarshal(res.Data, &data)
	return data.ID
}

func list[T any](t *testing.T, srv *httptest.Server, path string) []T {
	t.Helper()
	res := call(t, srv, http.MethodGet, path, "")
	if res.status != http.StatusOK {
		t.Fatalf("GET %s: %d %s %v", path, res.status, res.Code, res.Errors)
	}
	var out []T
	if err := json.Unmarshal(res.Data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestHallOfFameOrder(t *testing.T) {
	srv := newServer(t)
	// added out of order: the IDs are the order of a full tie
	ids := map[string]uint64{}
	for _, e := range []struct{ name, body string }{
		{"slow", `"score": 500, "time": 90`},
		{"untimed", `"score": 500`},
		{"fast", `"score": 500, "time": 60`},
		{"tie-first", `"score": 200, "time": 30`},
		{"tie-second", `"score": 200, "time": 30`},
		{"best", `"score": 900, "time": 999`},
	} {
		ids[e.name] = addScore(t, srv, `{"game": "liczozerca", "category": "add", "name": "`+e.name+`", `+e.body+`}`)
	}
	// another list, which must not show up
	addScore(t, srv, `{"game": "liczozerca", "category": "sub", "name": "other", "score": 99999}`)

	got := list[store.Score](t, srv, "/api/v1/scores/?filter_val_Game=liczozerca&filter_val_Category=add&order=SortKey&order_direction=asc&limit=10")
	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	if want := "best fast slow untimed tie-first tie-second"; strings.Join(names, " ") != want {
		t.Fatalf("order = %v, want %s", names, want)
	}
	if got[0].ID != ids["best"] || got[0].CreatedAt.IsZero() || got[0].Time == nil || *got[0].Time != 999 {
		t.Fatalf("first entry = %+v", got[0])
	}
}

func TestSingleListGame(t *testing.T) {
	srv := newServer(t)
	// a category sent to a game with a single list is dropped
	addScore(t, srv, `{"game": "szosa", "category": "whatever", "name": "  Ala \u0007  ma  kota\u200b ", "score": 7800, "time": 300, "completed": true}`)
	got := list[store.Score](t, srv, "/api/v1/scores/?filter_val_Game=szosa&filter_val_Category=&order=SortKey&order_direction=asc")
	if len(got) != 1 || got[0].Name != "Ala ma kota" || got[0].Category != "" || !got[0].Completed {
		t.Fatalf("got %+v", got)
	}
}

func TestScoreValidation(t *testing.T) {
	srv := newServer(t)
	for _, c := range []struct {
		body  string
		field string
	}{
		{`{"game": "tetris", "name": "a", "score": 1}`, "Game"},
		{`{"game": "liczozerca", "category": "pow", "name": "a", "score": 1}`, "Category"},
		{`{"game": "liczozerca", "name": "a", "score": 1}`, "Category"},
		{`{"game": "szosa", "name": " \t ", "score": 1}`, "Name"},
		{`{"game": "szosa", "name": "123456789012345678901", "score": 1}`, "Name"},
		{`{"game": "szosa", "name": "a", "score": -1}`, "Score"},
		{`{"game": "szosa", "name": "a", "score": 100000001}`, "Score"},
		{`{"game": "szosa", "name": "a", "score": 1, "time": 604801}`, "Time"},
		{`{"game": "szosa", "name": "a", "score": 1, "level": -1}`, "Level"},
		{`{"game": "szosa", "name": "a", "score": 1, "rank": 1}`, "rank"},
	} {
		res := call(t, srv, http.MethodPut, "/api/v1/scores/", c.body)
		if res.status != http.StatusBadRequest || res.Errors[c.field] == "" {
			t.Errorf("%s: %d %s %v, want a 400 about %s", c.body, res.status, res.Code, res.Errors, c.field)
		}
	}
	// 20 characters, but more bytes
	addScore(t, srv, `{"game": "szosa", "name": "ŻÓŁĆŻÓŁĆŻÓŁĆŻÓŁĆŻÓŁĆ", "score": 1}`)
	// the server sets these, whatever the client sends
	addScore(t, srv, `{"game": "szosa", "name": "cheat", "score": 1, "sort_key": -5, "created_at": "2000-01-01T00:00:00Z", "id": 1}`)
	got := list[store.Score](t, srv, "/api/v1/scores/?filter_val_Name=cheat")
	if len(got) != 1 || got[0].SortKey != store.SortKey(1, nil) || got[0].CreatedAt.Year() == 2000 || got[0].ID == 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestReadOnlyRoutes(t *testing.T) {
	srv := newServer(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodPut, "/api/v1/games/"},
		{http.MethodPut, "/api/v1/question-sets/"},
		{http.MethodPut, "/api/v1/questions/"},
		{http.MethodPut, "/api/v1/scores/1"},
		{http.MethodDelete, "/api/v1/scores/1"},
	} {
		if res := call(t, srv, c.method, c.path, `{}`); res.status != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: %d, want 405", c.method, c.path, res.status)
		}
	}
}

func TestGamesAndQuestions(t *testing.T) {
	srv := newServer(t)
	games := list[store.Game](t, srv, "/api/v1/games/?order=Position&limit=100")
	if len(games) != len(catalog.Games) || games[0].Slug != catalog.Games[0].Slug {
		t.Fatalf("games = %+v", games)
	}

	sets := list[store.QuestionSet](t, srv, "/api/v1/question-sets/?order=Position&limit=100")
	if len(sets) == 0 {
		t.Fatal("no question sets")
	}
	var set store.QuestionSet
	for _, s := range sets {
		if s.Slug == "kartkowka-2" {
			set = s
		}
	}
	if set.FallSeconds == nil || *set.FallSeconds != 45 || set.QuestionCount == 0 {
		t.Fatalf("kartkowka-2 = %+v", set)
	}
	questions := list[store.Question](t, srv, "/api/v1/questions/?filter_val_SetID="+jsonNumber(set.ID)+"&order=Position&limit=1000")
	if int64(len(questions)) != set.QuestionCount || questions[0].Position != 1 || len(questions[0].Wrong) != 1 {
		t.Fatalf("%d questions, the first %+v", len(questions), questions[0])
	}

	// the source hash is internal
	res := call(t, srv, http.MethodGet, "/api/v1/question-sets/?limit=1", "")
	if strings.Contains(string(res.Data), "hash") {
		t.Fatalf("question sets show their hash: %s", res.Data)
	}
}

func jsonNumber(id uint64) string {
	b, _ := json.Marshal(id)
	return string(b)
}

func TestCORSAndHealth(t *testing.T) {
	srv := newServer(t)
	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/v1/scores/", nil)
	req.Header.Set("Origin", "null") // a page opened from a file
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode/100 != 2 || res.Header.Get("Access-Control-Allow-Origin") != "*" ||
		!strings.Contains(res.Header.Get("Access-Control-Allow-Methods"), "PUT") {
		t.Fatalf("preflight: %d %v", res.StatusCode, res.Header)
	}

	for _, path := range []string{"/healthz", "/readyz", "/api/v1/openapi.json"} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Errorf("GET %s: %d", path, res.StatusCode)
		}
	}
}
