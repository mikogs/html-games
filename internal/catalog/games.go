// Package catalog is what the API serves but never changes: the games, with
// the lists of their halls of fame, and the sets of questions written by
// hand. Sync writes both into the database.
package catalog

// GameDef is a game that can post scores. Its entries are ranked separately
// per category; no categories means a single, unnamed list (category "").
type GameDef struct {
	Slug       string
	Title      string
	Categories []string
	// Also one list per set of questions, as category "set-<slug>".
	QuestionSets bool
}

// Games are in the order of the landing page. Add a game here before it can
// post scores.
var Games = []GameDef{
	{Slug: "liczozerca", Title: "Liczożerca", Categories: []string{"add", "sub", "mul", "div"}},
	// one list per kind of task; score = points, time = seconds of the whole game,
	// level = task difficulty (results up to level * 10), completed = escaped from the last level
	{Slug: "laboratorium", Title: "Tajne Laboratorium", Categories: []string{"add", "sub", "mul", "div", "all"}},
	// the same as Tajne Laboratorium, in lists of its own
	{Slug: "laboratorium3d", Title: "Tajne Laboratorium 3D", Categories: []string{"add", "sub", "mul", "div", "all"}},
	// one list per domain and kind of task (only maths so far), and one per set of questions;
	// score = points (10 per right answer), time = seconds of the game,
	// level = the level played (results up to level * 10; none for a set of questions)
	{Slug: "psiaszkola", Title: "Psia Szkoła", Categories: []string{"math-add", "math-sub", "math-mul", "math-div", "math-all"}, QuestionSets: true},
	// score = points (100 per animal found, a bonus for finding it fast, 200 per level), time = seconds of the game,
	// level = the last level played (1-4), completed = all four levels done
	{Slug: "zwierzatka", Title: "Zagubione Zwierzątka"},
	// score = metres driven (the whole route when finished), time = seconds to get there
	{Slug: "szosa", Title: "Szalona Szosa"},
}

// SetCategory is the category of the list of a set of questions.
func SetCategory(setSlug string) string { return "set-" + setSlug }
