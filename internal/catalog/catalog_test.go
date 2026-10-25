package catalog

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	htmlgames "github.com/mikogs/html-games"
	"github.com/mikogs/html-games/internal/store"
)

func TestParseSetFillsIn(t *testing.T) {
	set, err := ParseSet("t.json", "t", []byte(`{"name": " T ", "questions": [
		{"type": "tf", "text": "1=1", "answer": "PRAWDA"},
		{"text": "2+___=4", "answer": 2, "wrong": [1, "3"]},
		{"text": "370:10=?", "answer": "37", "wrong": ["370"]},
		{"text": "Stolica", "answer": "Warszawa", "wrong": ["Kraków"]},
		{"text": "x", "answer": "y", "wrong": ["z"], "full": "own"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	if set.Name != "T" || set.Domain != "math" || set.Position != 0 || set.FallSeconds != nil {
		t.Fatalf("set = %+v", set)
	}
	want := []ParsedQuestion{
		{Type: "tf", Text: "1=1", Answer: "PRAWDA", Wrong: []string{"FAŁSZ"}, Full: "1=1 - PRAWDA"},
		{Type: "choice", Text: "2+___=4", Answer: "2", Wrong: []string{"1", "3"}, Full: "2+2=4"},
		{Type: "choice", Text: "370:10=?", Answer: "37", Wrong: []string{"370"}, Full: "370:10=37"},
		{Type: "choice", Text: "Stolica", Answer: "Warszawa", Wrong: []string{"Kraków"}, Full: "Stolica: Warszawa"},
		{Type: "choice", Text: "x", Answer: "y", Wrong: []string{"z"}, Full: "own"},
	}
	for i, q := range set.Questions {
		if q.Type != want[i].Type || q.Text != want[i].Text || q.Answer != want[i].Answer ||
			!slices.Equal(q.Wrong, want[i].Wrong) || q.Full != want[i].Full {
			t.Errorf("question %d = %+v, want %+v", i+1, q, want[i])
		}
	}
}

func TestParseSetRejects(t *testing.T) {
	for _, c := range []struct{ slug, json, err string }{
		{"Bad_Slug", `{"name": "a", "questions": [{"text": "a", "answer": "b", "wrong": ["c"]}]}`, "file name"},
		{"s", `{`, "not valid JSON"},
		{"s", `{"name": "", "questions": [{"text": "a", "answer": "b", "wrong": ["c"]}]}`, `"name"`},
		{"s", `{"name": "a", "questions": []}`, `"questions"`},
		{"s", `{"name": "a", "fall_seconds": 4, "questions": [{"text": "a", "answer": "b", "wrong": ["c"]}]}`, `"fall_seconds"`},
		{"s", `{"name": "a", "fall_seconds": 45.5, "questions": [{"text": "a", "answer": "b", "wrong": ["c"]}]}`, `"fall_seconds"`},
		{"s", `{"name": "a", "questions": [{"text": "", "answer": "b", "wrong": ["c"]}]}`, `question 1: "text"`},
		{"s", `{"name": "a", "questions": [{"type": "tf", "text": "a", "answer": "TAK"}]}`, "PRAWDA or FAŁSZ"},
		{"s", `{"name": "a", "questions": [{"text": "a", "answer": "b"}]}`, `"wrong"`},
		{"s", `{"name": "a", "questions": [{"text": "a", "answer": "b", "wrong": ["b"]}]}`, `"wrong"`},
		{"s", `{"name": "a", "questions": [{"text": "a", "answer": "b", "wrong": ["c", "c"]}]}`, `"wrong"`},
		{"s", `{"name": "a", "questions": [{"text": "a", "answer": "b", "wrong": ["1", "2", "3", "4", "5", "6"]}]}`, `"wrong"`},
		{"s", `{"name": "a", "questions": [{"type": "open", "text": "a", "answer": "b"}]}`, `"type"`},
	} {
		_, err := ParseSet("t.json", c.slug, []byte(c.json))
		if err == nil || !strings.Contains(err.Error(), c.err) {
			t.Errorf("%s: err = %v, want one about %s", c.json, err, c.err)
		}
	}
}

func TestTheSetsInTheRepositoryParse(t *testing.T) {
	files, _ := fs.Sub(htmlgames.Questions, "questions")
	sets, _, err := LoadSets(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) == 0 {
		t.Fatal("no sets in questions/")
	}
}

func TestSync(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := store.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	reg := store.NewRegistry(db)
	set := func(name string) *fstest.MapFile {
		return &fstest.MapFile{Data: []byte(`{"name": "` + name + `", "questions": [
			{"text": "1+1=?", "answer": "2", "wrong": ["3"]}, {"type": "tf", "text": "0=0", "answer": "PRAWDA"}]}`)}
	}

	changes, err := Sync(ctx, reg, fstest.MapFS{"a.json": set("A"), "b.json": set("B")})
	if err != nil {
		t.Fatal(err)
	}
	if got := changes.String(); got != "games: laboratorium added, laboratorium3d added, liczozerca added, psiaszkola added, szosa added, zwierzatka added; question sets: a added, b added" {
		t.Fatalf("first sync: %s", got)
	}
	if changes, err := Sync(ctx, reg, fstest.MapFS{"a.json": set("A"), "b.json": set("B")}); err != nil || !changes.Empty() {
		t.Fatalf("second sync: %v, %v", changes, err)
	}

	changes, err = Sync(ctx, reg, fstest.MapFS{"a.json": set("A2")})
	if err != nil {
		t.Fatal(err)
	}
	if got := changes.String(); got != "games: psiaszkola updated; question sets: a updated, b removed" {
		t.Fatalf("third sync: %s", got)
	}
	questions, err := list[store.Question](ctx, reg, store.KeyQuestions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(questions) != 2 {
		t.Fatalf("%d questions left, want the 2 of a", len(questions))
	}
	games, err := list[store.Game](ctx, reg, store.KeyGames, map[string]string{"Slug": "psiaszkola"})
	if err != nil || len(games) != 1 {
		t.Fatal(games, err)
	}
	if want := []string{"math-add", "math-sub", "math-mul", "math-div", "math-all", "set-a"}; !slices.Equal(games[0].Categories, want) {
		t.Fatalf("psiaszkola categories = %v, want %v", games[0].Categories, want)
	}
}
