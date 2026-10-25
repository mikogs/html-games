package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// A set of questions is a file questions/<slug>.json:
//
//	{"name": "KARTKÓWKA 2", "domain": "math", "position": 1, "fall_seconds": 45, "questions": [...]}
//
// "fall_seconds" (optional) is how long a question takes to fall through the
// empty well - its speed; without it the game's own speed. Each question is
// one of:
//
//	{"text": "370:10=?", "answer": "37", "wrong": ["370", "3700", "307"]}
//	    1-5 wrong answers; the game shows the right one and one of them
//	{"type": "tf", "text": "81-0=81×1", "answer": "PRAWDA"}
//	    true or false: the answer is PRAWDA or FAŁSZ, and the other one is the wrong one
//
// plus, optionally, "full": the question with its answer, shown when a player
// gets it wrong (by default it is made from the text and the answer: a blank,
// ___, filled in with the answer, or the answer after a closing "=").

const (
	textMax     = 80
	answerMax   = 40
	wrongMax    = 5
	nameMax     = 40
	fallMin     = 5
	fallMax     = 600
	answerTrue  = "PRAWDA"
	answerFalse = "FAŁSZ"
	// part of every set's hash: bump it when what is stored from a file
	// changes, so every set is loaded again
	format = 1
)

var (
	slugRe       = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	endsWithEqRe = regexp.MustCompile(`=\s*\??$`)
	trailingQmRe = regexp.MustCompile(`\s*\?$`)
)

// Set is a parsed questions/<slug>.json.
type Set struct {
	Slug        string
	Name        string
	Domain      string
	Position    int64
	FallSeconds *int64
	Questions   []ParsedQuestion
}

// ParsedQuestion is a question with everything it may leave out filled in.
type ParsedQuestion struct {
	Type   string
	Text   string
	Answer string
	Wrong  []string
	Full   string
}

// ParseSet checks the file of a set and fills in what its questions may leave
// out. file names it in the errors.
func ParseSet(file, slug string, raw []byte) (*Set, error) {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%s: %s", file, fmt.Sprintf(format, args...))
	}
	if !slugRe.MatchString(slug) {
		return nil, fail("the file name must be lowercase letters, digits and dashes")
	}
	var data struct {
		Name        json.RawMessage   `json:"name"`
		Domain      json.RawMessage   `json:"domain"`
		Position    json.RawMessage   `json:"position"`
		FallSeconds json.RawMessage   `json:"fall_seconds"`
		Questions   []json.RawMessage `json:"questions"`
	}
	if err := strictDecode(raw, &data); err != nil {
		return nil, fail("not valid JSON (%v)", err)
	}

	name, err := text(data.Name)
	if err != nil || name == "" || utf8.RuneCountInString(name) > nameMax {
		return nil, fail(`"name" must be 1-%d characters`, nameMax)
	}
	domain := "math"
	if data.Domain != nil {
		if domain, err = text(data.Domain); err != nil || domain == "" {
			return nil, fail(`"domain" must be a name, e.g. math`)
		}
	}
	var position int64
	if data.Position != nil {
		p, ok := wholeNumber(data.Position)
		if !ok {
			return nil, fail(`"position" must be a whole number`)
		}
		position = *p
	}
	var fall *int64
	if data.FallSeconds != nil && string(data.FallSeconds) != "null" {
		f, ok := wholeNumber(data.FallSeconds)
		if !ok || *f < fallMin || *f > fallMax {
			return nil, fail(`"fall_seconds" must be a whole number of seconds, %d-%d`, fallMin, fallMax)
		}
		fall = f
	}
	if len(data.Questions) == 0 {
		return nil, fail(`"questions" must be a non-empty list`)
	}

	set := &Set{Slug: slug, Name: name, Domain: domain, Position: position, FallSeconds: fall}
	for i, rawQ := range data.Questions {
		q, err := parseQuestion(rawQ)
		if err != nil {
			return nil, fail("question %d: %v", i+1, err)
		}
		set.Questions = append(set.Questions, *q)
	}
	return set, nil
}

func parseQuestion(raw json.RawMessage) (*ParsedQuestion, error) {
	var q struct {
		Type   json.RawMessage   `json:"type"`
		Text   json.RawMessage   `json:"text"`
		Answer json.RawMessage   `json:"answer"`
		Wrong  []json.RawMessage `json:"wrong"`
		Full   json.RawMessage   `json:"full"`
	}
	if err := strictDecode(raw, &q); err != nil {
		return nil, fmt.Errorf("not a question (%v)", err)
	}
	typ := "choice"
	if q.Type != nil {
		var err error
		if typ, err = text(q.Type); err != nil {
			typ = ""
		}
	}
	questionText, err := text(q.Text)
	if err != nil || questionText == "" || utf8.RuneCountInString(questionText) > textMax {
		return nil, fmt.Errorf(`"text" must be 1-%d characters`, textMax)
	}
	answer, err := text(q.Answer)
	if err != nil {
		answer = ""
	}

	var wrong []string
	switch typ {
	case "tf":
		if answer != answerTrue && answer != answerFalse {
			return nil, fmt.Errorf("a true / false question's answer must be %s or %s", answerTrue, answerFalse)
		}
		wrong = []string{answerTrue}
		if answer == answerTrue {
			wrong = []string{answerFalse}
		}
	case "choice":
		if answer == "" || utf8.RuneCountInString(answer) > answerMax {
			return nil, fmt.Errorf(`"answer" must be 1-%d characters`, answerMax)
		}
		for _, w := range q.Wrong {
			s, err := text(w)
			if err != nil {
				s = ""
			}
			wrong = append(wrong, s)
		}
		unique := slices.Clone(wrong)
		slices.Sort(unique)
		if len(wrong) == 0 || len(wrong) > wrongMax || len(slices.Compact(unique)) != len(wrong) ||
			slices.Contains(wrong, answer) || slices.Contains(wrong, "") {
			return nil, fmt.Errorf(`"wrong" must be 1-%d different answers, none of them the right one`, wrongMax)
		}
		for _, w := range wrong {
			if utf8.RuneCountInString(w) > answerMax {
				return nil, fmt.Errorf("a wrong answer longer than %d characters", answerMax)
			}
		}
	default:
		return nil, fmt.Errorf(`"type" must be choice (the default) or tf`)
	}

	full, err := text(q.Full)
	if err != nil {
		full = ""
	}
	if full == "" {
		switch {
		case typ == "tf":
			full = questionText + " - " + answer
		case strings.Contains(questionText, "___"):
			full = strings.ReplaceAll(questionText, "___", answer)
		case endsWithEqRe.MatchString(questionText):
			full = trailingQmRe.ReplaceAllString(questionText, "") + answer
		default:
			full = questionText + ": " + answer
		}
	}
	return &ParsedQuestion{Type: typ, Text: questionText, Answer: answer, Wrong: wrong, Full: full}, nil
}

// strictDecode decodes JSON, keeping numbers as they are written.
func strictDecode(raw []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	return d.Decode(v)
}

// text returns a JSON string or number as trimmed text, e.g. "37" for 37, so
// that answers can be written as numbers. A missing value is "".
func text(raw json.RawMessage) (string, error) {
	if raw == nil || string(raw) == "null" {
		return "", nil
	}
	var v any
	if err := strictDecode(raw, &v); err != nil {
		return "", err
	}
	switch v := v.(type) {
	case string:
		return strings.TrimSpace(v), nil
	case json.Number:
		return v.String(), nil
	}
	return "", fmt.Errorf("not text")
}

// wholeNumber returns a JSON integer, e.g. 45 but not 45.5 or "45".
func wholeNumber(raw json.RawMessage) (*int64, bool) {
	var v any
	if err := strictDecode(raw, &v); err != nil {
		return nil, false
	}
	n, ok := v.(json.Number)
	if !ok {
		return nil, false
	}
	i, err := n.Int64()
	if err != nil {
		return nil, false
	}
	return &i, true
}
