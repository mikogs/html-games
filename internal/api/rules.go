// Package api is the HTTP API of the games, gorestapi over the tables of
// internal/store, at /api/v1/.
package api

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/mikogs/gocrud/pkg/registry"

	"github.com/mikogs/html-games/internal/store"
)

// Rules is the registry with the rules of the scores, which the handler
// saves through it: the game and the category have to exist, the server
// cleans up the name and sets sort_key and created_at, whatever the client
// sent.
type Rules struct {
	*registry.CRUD
}

var (
	controlRe    = regexp.MustCompile(`\p{C}`)
	whitespaceRe = regexp.MustCompile(`\s+`)
)

// CleanName drops control characters and collapses whitespace.
func CleanName(name string) string {
	return strings.TrimSpace(whitespaceRe.ReplaceAllString(controlRe.ReplaceAllString(name, ""), " "))
}

func (r *Rules) Save(ctx context.Context, obj any, userID any) error {
	score, ok := obj.(*store.Score)
	if !ok {
		return r.CRUD.Save(ctx, obj, userID)
	}
	score.Name = CleanName(score.Name)

	games, err := r.CRUD.List(ctx, store.KeyGames, registry.ListOptions{
		Limit: 1, Filters: registry.Filters{Vals: map[string]string{"Slug": score.Game}},
	})
	if err != nil {
		return err
	}
	if len(games) == 0 {
		return &registry.ModelValidationError{Err: errors.New("unknown game"), Violations: map[string]string{"Game": "oneof"}}
	}
	categories := games[0].(*store.Game).Categories
	if len(categories) == 0 {
		// a single list
		score.Category = ""
	} else if !slices.Contains(categories, score.Category) {
		return &registry.ModelValidationError{Err: errors.New("unknown category"), Violations: map[string]string{"Category": "oneof"}}
	}

	score.SortKey = store.SortKey(score.Score, score.Time)
	score.CreatedAt = time.Now().UTC()
	return r.CRUD.Save(ctx, score, userID)
}
