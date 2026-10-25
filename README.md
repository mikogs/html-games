# html-games

Browser games for kids (in Polish) and the API they share: a hall of fame
of each game and sets of questions written by hand. Served on
`https://gry.<host_domain>`, deployed by the `065_gry.yml` playbook of
[devops](https://github.com/mikogs/devops) (`billy2/`), where they lived
until they were moved here with their history.

| Path | What it is |
| --- | --- |
| `web/` | The landing page and the games, each one self-contained `index.html`: `liczozerca`, `laboratorium`, `laboratorium3d`, `psiaszkola`, `zwierzatka`, `szosa`. `lib/three/` is Three.js r169 (MIT), used by `laboratorium3d`. |
| `questions/` | Sets of questions written by hand, one `<slug>.json` per set (e.g. `kartkowka-2.json`, a test from school), built into the server. |
| `cmd/gry/` | The server of the API. |
| `cmd/migration/` | Writes a new migration file after a struct of `internal/store` changes. |
| `internal/store/` | The tables (structs of [gocrud](https://github.com/mikogs/gocrud)) and their migration files. |
| `internal/catalog/` | The list of the games, and the parser of the sets of questions. Both are written into the database when the server starts. |
| `internal/api/` | The HTTP API: [gorestapi](https://github.com/mikogs/gorestapi) `v1.0.0` over the tables, with the rules of the scores. |

The games call the API at `/api/v1/` of the site they are served from, or at
`window.GRY_ORIGIN` (from `lib/origin.js`, written by the playbook) when a
page is opened from disk.

## The API

JSON, under `/api/v1/`, with `Access-Control-Allow-Origin: *` so that a game
opened from a file can use it too. There is no authentication: anyone who
can reach it can add any score.

| Request | Does |
| --- | --- |
| `GET /api/v1/games/` | The games, with the lists (`categories`) of their halls of fame. |
| `GET /api/v1/scores/?filter_val_Game=<game>&filter_val_Category=<category>&order=SortKey&order_direction=asc&limit=10` | The top 10 of a list. A game with a single list has the category `""`: `filter_val_Category=`. |
| `PUT /api/v1/scores/` | Adds an entry: `game`, `name` (1-20 characters), `score` (0-100000000), `category` (one of the game's, ignored for a game with a single list), and optional `time` (seconds), `level` and `completed`. Returns `201` with the `id`. |
| `GET /api/v1/question-sets/?order=Position` | The sets of questions, with `fall_seconds` and `question_count`. |
| `GET /api/v1/questions/?filter_val_SetID=<id>&order=Position&limit=1000` | The questions of a set. |
| `GET /api/v1/openapi.json` | The OpenAPI document of all of it. |

The lists, filters, pages and errors are gorestapi's, see its
[documentation](https://github.com/mikogs/gorestapi/blob/v1.0.0/docs/handler.md):
the filters and `order` take the names of the fields in Go (`Game`,
`SortKey`), the bodies and the responses the names in JSON (`game`,
`sort_key`), and an error is a problem of RFC 9457 with a `code`, e.g.
`VALIDATION_FAILED` with the failed fields in `errors`.

**The order of a hall of fame**: the higher score first, on a tie the
faster time (an entry without a time after the timed ones), on a full tie
the older entry. The server keeps it in `sort_key`, which it computes when
an entry is added, as gorestapi orders by one field. Every entry is kept.

**Categories**: Liczożerca has one list per operation (`add`, `sub`, `mul`,
`div`), Tajne Laboratorium and its 3D version one per kind of task (also
`all`), Psia Szkoła one per domain and kind (`math-add`, ...) and one per set
of questions (`set-<slug>`), and Szalona Szosa and Zagubione Zwierzątka a
single one. A game is added in `internal/catalog/games.go` before it can post
scores.

### A set of questions

`questions/<slug>.json`, where the slug is lowercase letters, digits and
dashes:

```json
{"name": "KARTKÓWKA 2", "domain": "math", "position": 1, "fall_seconds": 45, "questions": [
  {"text": "370:10=?", "answer": "37", "wrong": ["370", "3700", "307"]},
  {"type": "tf", "text": "81-0=81×1", "answer": "PRAWDA"}
]}
```

A question has 1-5 `wrong` answers, or is true or false (`"type": "tf"`,
the answer `PRAWDA` or `FAŁSZ`). `full`, the question with its answer shown
after a mistake, is made from the text when it is left out: a blank `___`
filled in, or the answer after a closing `=`. The server checks every file
when it starts and refuses to start with a mistake in one, saying where it
is. A set whose file changed is loaded again as a whole, and a set whose
file is gone is removed, but the scores of its list are kept.

## Running it

Go 1.26 (a newer toolchain is downloaded by `go` itself when needed).

```sh
go run ./cmd/gry serve -web web        # the games and the API on http://127.0.0.1:8090/
go test ./...
```

`gry serve` applies the migrations, loads the games and the sets of
questions, and serves the API. Its flags: `-addr` (`127.0.0.1:8090`), `-db`
(`games.db`), `-web` (also serve the games from a directory, for
development: nginx serves them in production), `-access-log`, and
`-trusted-proxies` (`127.0.0.1/32,::1/128`, whose `X-Forwarded-For` gives the
IP of the client for the rate limit).

`gry migrate -db <path>` does the same without serving, and prints
`migrated: [...]` and `catalog: ...` when it changes something.

`gry import-legacy -db <path> -from /var/lib/gry/gry.db` copies the hall of
fame of the database of the old PHP API into a database without scores,
keeping the IDs and the times of the entries.

## Changing the tables

The tables are created by the migration files in
`internal/store/migrations/`, applied with
[goose](https://github.com/pressly/goose) when the server starts. After
changing a struct of `internal/store/schema.go`:

```sh
go run ./cmd/migration <what_changed>
```

writes a new file with the changes. Read it, complete its `-- TODO:` parts
and write its `Down` part. A test checks that the files match the structs,
and the server checks it too before it starts. Never edit a file that is
applied already.
