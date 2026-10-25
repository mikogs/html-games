package catalog

import (
	"cmp"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mikogs/gocrud/pkg/registry"

	"github.com/mikogs/html-games/internal/store"
)

// Changes says what Sync changed, by slug: "added", "updated" or "removed".
type Changes struct {
	Games        map[string]string
	QuestionSets map[string]string
}

// Empty is true when Sync changed nothing.
func (c Changes) Empty() bool { return len(c.Games) == 0 && len(c.QuestionSets) == 0 }

func (c Changes) String() string {
	var parts []string
	for _, group := range []struct {
		name    string
		changes map[string]string
	}{{"games", c.Games}, {"question sets", c.QuestionSets}} {
		slugs := make([]string, 0, len(group.changes))
		for slug := range group.changes {
			slugs = append(slugs, slug)
		}
		slices.Sort(slugs)
		for i, slug := range slugs {
			slugs[i] = slug + " " + group.changes[slug]
		}
		if len(slugs) > 0 {
			parts = append(parts, group.name+": "+strings.Join(slugs, ", "))
		}
	}
	return strings.Join(parts, "; ")
}

// LoadSets parses every *.json at the root of files, see ParseSet, with the
// hash of each file.
func LoadSets(files fs.FS) ([]*Set, map[string]string, error) {
	names, err := fs.Glob(files, "*.json")
	if err != nil {
		return nil, nil, err
	}
	var sets []*Set
	hashes := map[string]string{}
	for _, name := range names {
		raw, err := fs.ReadFile(files, name)
		if err != nil {
			return nil, nil, err
		}
		slug := strings.TrimSuffix(path.Base(name), ".json")
		set, err := ParseSet("questions/"+name, slug, raw)
		if err != nil {
			return nil, nil, err
		}
		sum := sha1.Sum(append([]byte(strconv.Itoa(format)+"\n"), raw...))
		sets = append(sets, set)
		hashes[slug] = hex.EncodeToString(sum[:])
	}
	return sets, hashes, nil
}

// Sync brings the database in line with Games and the sets of questions in
// files, in one transaction: a set whose file changed is replaced as a whole,
// and the sets and games that are gone are removed. The scores are never
// touched, also of a game or a set that is removed.
func Sync(ctx context.Context, reg *registry.CRUD, files fs.FS) (Changes, error) {
	sets, hashes, err := LoadSets(files)
	if err != nil {
		return Changes{}, err
	}
	changes := Changes{Games: map[string]string{}, QuestionSets: map[string]string{}}
	err = reg.RunInTx(ctx, func(tx *registry.CRUD) error {
		if err := syncSets(ctx, tx, sets, hashes, changes.QuestionSets); err != nil {
			return err
		}
		return syncGames(ctx, tx, changes.Games)
	})
	if err != nil {
		return Changes{}, err
	}
	return changes, nil
}

func syncSets(ctx context.Context, tx *registry.CRUD, sets []*Set, hashes map[string]string, changes map[string]string) error {
	stored, err := list[store.QuestionSet](ctx, tx, store.KeyQuestionSets, nil)
	if err != nil {
		return err
	}
	known := map[string]*store.QuestionSet{}
	for _, s := range stored {
		known[s.Slug] = s
	}
	for _, set := range sets {
		row := known[set.Slug]
		delete(known, set.Slug)
		if row != nil && row.SourceHash == hashes[set.Slug] {
			continue
		}
		if row == nil {
			row = &store.QuestionSet{}
			changes[set.Slug] = "added"
		} else {
			changes[set.Slug] = "updated"
			if err := deleteQuestions(ctx, tx, row.ID); err != nil {
				return err
			}
		}
		row.Slug, row.Name, row.Domain, row.Position, row.FallSeconds = set.Slug, set.Name, set.Domain, set.Position, set.FallSeconds
		row.QuestionCount = int64(len(set.Questions))
		row.SourceHash = hashes[set.Slug]
		row.UpdatedAt = time.Now().UTC()
		if err := tx.Save(ctx, row, nil); err != nil {
			return fmt.Errorf("question set %s: %w", set.Slug, err)
		}
		for i, q := range set.Questions {
			question := &store.Question{
				SetID: row.ID, Position: int64(i + 1),
				Type: q.Type, Text: q.Text, Answer: q.Answer, Wrong: q.Wrong, Full: q.Full,
			}
			if err := tx.Save(ctx, question, nil); err != nil {
				return fmt.Errorf("question set %s, question %d: %w", set.Slug, i+1, err)
			}
		}
	}
	for slug, row := range known {
		// its questions go with it (on_delete:cascade)
		if err := tx.Delete(ctx, store.KeyQuestionSets, row.ID); err != nil {
			return fmt.Errorf("question set %s: %w", slug, err)
		}
		changes[slug] = "removed"
	}
	return nil
}

func deleteQuestions(ctx context.Context, tx *registry.CRUD, setID uint64) error {
	questions, err := list[store.Question](ctx, tx, store.KeyQuestions, map[string]string{"SetID": strconv.FormatUint(setID, 10)})
	if err != nil {
		return err
	}
	for _, q := range questions {
		if err := tx.Delete(ctx, store.KeyQuestions, q.ID); err != nil {
			return err
		}
	}
	return nil
}

// syncGames writes Games, with the categories of the sets of questions now in
// the database.
func syncGames(ctx context.Context, tx *registry.CRUD, changes map[string]string) error {
	sets, err := list[store.QuestionSet](ctx, tx, store.KeyQuestionSets, nil)
	if err != nil {
		return err
	}
	slices.SortStableFunc(sets, func(a, b *store.QuestionSet) int {
		return cmp.Or(cmp.Compare(a.Position, b.Position), cmp.Compare(a.Name, b.Name))
	})
	stored, err := list[store.Game](ctx, tx, store.KeyGames, nil)
	if err != nil {
		return err
	}
	known := map[string]*store.Game{}
	for _, g := range stored {
		known[g.Slug] = g
	}
	for i, def := range Games {
		categories := append([]string{}, def.Categories...)
		if def.QuestionSets {
			for _, s := range sets {
				categories = append(categories, SetCategory(s.Slug))
			}
		}
		row := known[def.Slug]
		delete(known, def.Slug)
		if row == nil {
			row = &store.Game{}
			changes[def.Slug] = "added"
		} else if row.Title == def.Title && slices.Equal(row.Categories, categories) && row.Position == int64(i+1) {
			continue
		} else {
			changes[def.Slug] = "updated"
		}
		row.Slug, row.Title, row.Categories, row.Position = def.Slug, def.Title, categories, int64(i+1)
		if err := tx.Save(ctx, row, nil); err != nil {
			return fmt.Errorf("game %s: %w", def.Slug, err)
		}
	}
	for slug, row := range known {
		if err := tx.Delete(ctx, store.KeyGames, row.ID); err != nil {
			return fmt.Errorf("game %s: %w", slug, err)
		}
		changes[slug] = "removed"
	}
	return nil
}

// list returns all the records of key that match filters, by field name.
func list[T any](ctx context.Context, reg *registry.CRUD, key string, filters map[string]string) ([]*T, error) {
	objs, err := reg.List(ctx, key, registry.ListOptions{Order: "ID", OrderDirection: "asc", Filters: registry.Filters{Vals: filters}})
	if err != nil {
		return nil, err
	}
	out := make([]*T, 0, len(objs))
	for _, o := range objs {
		out = append(out, o.(*T))
	}
	return out, nil
}
