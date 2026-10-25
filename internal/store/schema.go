// Package store holds the tables of the games' API: the structs, the
// registry of gocrud over them, and the migration files that create them.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/mikogs/gocrud"
	"github.com/mikogs/gocrud/pkg/registry"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// Migrations are applied with goose. Write a new one with
// `go run ./cmd/migration <name>` after changing a struct, never edit one
// that is applied.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// The registry keys, which are also the URL paths of the API.
const (
	KeyGames        = "games"
	KeyScores       = "scores"
	KeyQuestionSets = "question-sets"
	KeyQuestions    = "questions"
)

// Game is one of the games and the lists of its hall of fame. It is written
// by catalog.Sync from the list in the code, never by the API.
type Game struct {
	ID    uint64 `json:"id"`
	Slug  string `json:"slug" validate:"required,max=40" crud:"uniq" crud_regexp:"^[a-z0-9]+(-[a-z0-9]+)*$"`
	Title string `json:"title" validate:"required,max=80"`
	// The lists of the hall of fame, each ranked on its own. Empty when the
	// game has a single one, the category "".
	Categories []string `json:"categories" crud:"json"`
	Position   int64    `json:"position"`
}

// Score is an entry of the hall of fame of a game. The API only adds them.
type Score struct {
	ID       uint64 `json:"id"`
	Game     string `json:"game" validate:"required,max=40" crud:"idx:score_ranking"`
	Category string `json:"category" validate:"max=40" crud:"idx:score_ranking"`
	Name     string `json:"name" validate:"required,max=20"`
	// What the game counts, e.g. points or metres driven.
	Score int64 `json:"score" validate:"min=0,max=100000000"`
	// Seconds the game took, and the level played, when the game has them.
	Time      *int64 `json:"time" validate:"omitempty,min=0,max=604800"`
	Level     *int64 `json:"level" validate:"omitempty,min=0,max=10000"`
	Completed bool   `json:"completed"`
	// The place in its list, lowest first: see SortKey. Set by the server.
	SortKey   int64     `json:"sort_key" crud:"idx:score_ranking"`
	CreatedAt time.Time `json:"created_at"`
}

// Every Score.Time is at most MaxTime, so a list ordered by
// sort_key ascending (and then by ID, as gorestapi does) is: the higher
// score first, on a tie the faster time, with an entry without a time after
// every timed one, and on a full tie the older entry.
const (
	MaxScore = 100_000_000
	MaxTime  = 7 * 86400
	noTime   = MaxTime + 1
)

// SortKey returns the sort_key of an entry. The largest one, of score 0 and no
// time, is about 6e13, far below the largest int64.
func SortKey(score int64, timeSeconds *int64) int64 {
	t := int64(noTime)
	if timeSeconds != nil {
		t = *timeSeconds
	}
	return (MaxScore-score)*(noTime+1) + t
}

// QuestionSet is a set of questions written by hand, e.g. a test from
// school, loaded from questions/<slug>.json by catalog.Sync.
type QuestionSet struct {
	ID     uint64 `json:"id"`
	Slug   string `json:"slug" validate:"required,max=60" crud:"uniq" crud_regexp:"^[a-z0-9]+(-[a-z0-9]+)*$"`
	Name   string `json:"name" validate:"required,max=40"`
	Domain string `json:"domain" validate:"required,max=20"`
	// The order of the sets, lowest first.
	Position int64 `json:"position"`
	// How many seconds a question takes to fall through the empty well of
	// Psia Szkoła, nil for the game's own speed.
	FallSeconds   *int64 `json:"fall_seconds" validate:"omitempty,min=5,max=600"`
	QuestionCount int64  `json:"question_count"`
	// The hash of the file the set was loaded from, to load only the sets
	// whose files changed.
	SourceHash string    `json:"-"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Question is one question of a QuestionSet.
type Question struct {
	ID       uint64 `json:"id"`
	SetID    uint64 `json:"set_id" validate:"required" crud:"idx fk:question_set on_delete:cascade"`
	Position int64  `json:"position"`
	// "choice": the right answer and one of the wrong ones are shown, "tf":
	// true or false, the answer is PRAWDA or FAŁSZ and the other is wrong.
	Type   string   `json:"type" validate:"oneof=choice tf"`
	Text   string   `json:"text" validate:"required,max=80"`
	Answer string   `json:"answer" validate:"required,max=40"`
	Wrong  []string `json:"wrong" crud:"json"`
	// The question with its answer, shown when a player gets it wrong.
	Full string `json:"full" validate:"required"`
}

// Open opens the SQLite database at path, creating it if needed. A single
// connection: SQLite writes one at a time anyway, and the API is small.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// NewRegistry returns the registry of all the tables.
func NewRegistry(db *sql.DB) *registry.CRUD {
	return registry.New(map[string]func() any{
		KeyGames:        func() any { return &Game{} },
		KeyScores:       func() any { return &Score{} },
		KeyQuestionSets: func() any { return &QuestionSet{} },
		KeyQuestions:    func() any { return &Question{} },
	}, db, gocrud.Options{Dialect: gocrud.DialectSQLite})
}

// Migrate applies the migration files that were not applied yet, and checks
// that the tables then match the structs. It returns the versions it applied.
func Migrate(ctx context.Context, db *sql.DB) ([]int64, error) {
	files, err := fs.Sub(Migrations, "migrations")
	if err != nil {
		return nil, err
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, files)
	if err != nil {
		return nil, err
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("applying the migrations: %w", err)
	}
	applied := make([]int64, 0, len(results))
	for _, r := range results {
		applied = append(applied, r.Source.Version)
	}
	if err := NewRegistry(db).CheckSchema(ctx); err != nil {
		if errors.Is(err, gocrud.ErrSchemaDiffers) {
			return applied, fmt.Errorf("the migrations do not match the structs: %w", err)
		}
		return applied, err
	}
	return applied, nil
}
