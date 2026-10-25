# html-games

Tiny HTML games I made with my kids, just for fun. They're in Polish and
English, each one a single page that runs in the browser. Their small backend, a
hall of fame for each game and sets of questions written by hand, is in Go
on [mikogs/gocrud](https://github.com/mikogs/gocrud) (through
[mikogs/gorestapi](https://github.com/mikogs/gorestapi)). The games and
the backend were generated with [Claude](https://claude.com/claude-code).

They run at home on `https://gry.<host_domain>`, deployed by the
`065_gry.yml` playbook of [devops](https://github.com/mikogs/devops)
(`billy2/`), where they lived before they moved here.

| Path | What it is |
| --- | --- |
| `web/` | The landing page and the games, each one self-contained `index.html`: `liczozerca`, `laboratorium`, `laboratorium3d`, `psiaszkola`, `zwierzatka`, `szosa`. `lib/three/` is Three.js r169 (MIT), used by `laboratorium3d`. |
| `questions/` | Sets of questions written by hand, one `<slug>.json` per set (e.g. `kartkowka-2.json`, a test from school), built into the server. |
| `cmd/gry/` | The server of the API. |
| `cmd/migration/` | Writes a new migration file after a struct of `internal/store` changes. |
| `internal/store/` | The tables (structs of [gocrud](https://github.com/mikogs/gocrud)) and their migration files. |
| `internal/catalog/` | The list of the games, and the parser of the sets of questions. Both are written into the database when the server starts. |
| `internal/api/` | The HTTP API: [gorestapi](https://github.com/mikogs/gorestapi) `v1.0.0` over the tables, with the rules of the scores. |

## The games

| Path | What it is |
| --- | --- |
| `web/liczozerca/index.html` | Liczożerca: a Pac-Man-style maths game (one self-contained HTML file). |
| `web/laboratorium/index.html` | Tajne Laboratorium: a platform game - pick a character, collect keys, open doors (some with a PIN: 4 maths tasks), ride lifts worked by wall switches, go through wooden doors into small tinted side rooms (one screen each, with a computer and diamonds, sometimes a professor doing an experiment or drinking coffee), shoot cracked walls (3 shots) to find hidden places full of diamonds and a treasure room - and on level 3 a secret teleport (used with E, like doors) to a prehistoric jungle with a sky, clouds, a smoking volcano and jungle trees, where small dinosaurs walk and jewels (3 points, triple a diamond) and small bones (1 point) can be collected, then back through its own teleport, pass a plumber fixing a sink, coffee machines and water coolers, jump over or shoot mutants and avoid toxic drops from leaking pipes and toxic puddles (5 lives), find pistols (5 bullets each, kept between levels), score points (tasks, diamonds, floppy disks and CDs, level bonuses, the letters of NAUKA; an extra life every 200) with a hall of fame, solve maths tasks on computers - addition, subtraction, multiplication, division or all, results up to 10-100; levels 5-8 are the toxic zone: dirty slimy walls, mutated spiders and plants, half-lizard professors, two-headed cats and sheep with glowing eyes that take 2 shots each, more pistols, and an ice room full of frozen rats on every level; levels 4 and 8 end with a boss room before the exit - a fat professor throwing toxic test tubes (10 shots) and a big lizard in glasses spitting green saliva (20 shots), with a health bar, one-tile ledges to escape to and pistols that come back (one self-contained HTML file). |
| `web/szosa/index.html` | Szalona Szosa: a Lotus-style racing game - pick a car and race a fixed 7.8 km route against the clock, through traffic, narrow roads and road works; has a hall of fame (one self-contained HTML file). |
| `web/laboratorium3d/index.html` | Tajne Laboratorium 3D: the laboratory seen through the player's eyes, like Wolfenstein 3D, with no jumping - walk the corridors and rooms (arrows / WASD, the mouse after a click in the picture, or touch buttons and dragging on a phone), collect coloured keys for the doors of the same colour, open doors with a PIN (4 maths tasks), solve a task on every computer to open the exit, collect diamonds, floppy disks, CDs and the letters of NAUKA, find pistols (5 bullets each) to shoot mutants (a pig, a dog, toxic cats that take 2 shots) and cracked walls hiding pockets, avoid toxic puddles (5 lives); a map in the corner shows what has been seen (TAB: the whole level); 3 levels - the lab, a brick warehouse full of crates and the toxic reactor; the character, task settings and name are shared with Tajne Laboratorium; has a hall of fame of its own. Built with Three.js from `lib/three/` (one HTML file). |
| `web/psiaszkola/index.html` | Psia Szkoła: a Tetris-like quiz with a dog theme (Pan Billy, a cavalier king charles spaniel teacher with a bow tie) - bones with a question and two answers fall slowly down a well; pick the right one with the left / right arrow (+10 points) and the bone vanishes, a wrong or missing answer leaves it on the pile with the right answer shown (long questions and answers get a taller bone), down arrow drops it at once; the well holds 20, so 20 mistakes end the game; the teacher gives a thumbs up at 20 right answers, an OK at 30 and a wide smile at 40; has a hall of fame (one list per kind of task). Pick a domain (maths now; English and German are shown as coming soon), a kind (addition, subtraction, multiplication, division or all) and a level (results up to 10-100, as in Tajne Laboratorium), or a set of questions written by hand loaded from the API, such as KARTKÓWKA 2 (one self-contained HTML file). |
| `web/zwierzatka/index.html` | Zagubione Zwierzątka: a top-down game for learning the eight directions (N, NE, E, SE, S, SW, W, NW and their names, in Polish or in English; the top of the screen is always north) - pick a character as in Tajne Laboratorium (it starts with the one made there), then find lost animals (a fox, a cat, a dog, a rabbit, a hedgehog, a squirrel, a duckling, an owl, a fawn, a turtle, a budgie, a guinea pig) one at a time: they are hidden until the player is close, and a hint above the map (no compass, so the directions have to be learned) says which way the one lost now is, as the letters ("IDŹ NA NE") and the Polish name ("IDŹ NA PÓŁNOCNY WSCHÓD") in turn, changing as the player walks, and says when a river or a fence is in the way; 100 points per animal, a bonus for finding it fast, 200 per level; has a hall of fame. Level 1 is a fenced housing estate (two rows of 6 houses facing each other across a gravel road, with cars in front of some of them and three parked on the road; the gravel road leaves the fence to the west, to an asphalt road going north-south, and to the east, turning south to an asphalt road behind the south fence; the player starts in front of the 3rd house of the bottom row with a white station wagon, the next neighbours have a small white car and a brown SUV) in a forest with rivers and wooden bridges; levels 2-4 are forests (rivers, a lake, a dense wild forest) - 20, 30, 30 and 30 animals (one self-contained HTML file). |
| `web/lib/three/three.module.min.js` | [Three.js](https://threejs.org) r169 (MIT, its `LICENSE` next to it), the 3D engine of Tajne Laboratorium 3D - kept here, so the games don't depend on a CDN. |
| `web/lib/lang.js` | The language of the pages, and the flags that change it, see [Languages](#languages). |

The games call the API at `/api/v1/` of the site they are served from, or at
`window.GRY_ORIGIN` (from `lib/origin.js`, written by the playbook) when a
page is opened from disk.

### Languages

The pages are in Polish, or in English after a click on the British flag
under the title of the landing page or in a game's menu (the Polish flag
switches back). The choice is kept in the cookie `gry_lang` (`pl` or `en`)
for a year, for every page of the site, and also in `localStorage`, as a
page opened from disk can't keep cookies. `?lang=en` (or `?lang=pl`) in the
address picks the language too, e.g. for a link. The flags are only in the
menus: a change of the language reloads the page, which would end a game.

`web/lib/lang.js` reads the language before a game's script runs and
exposes it as `GryLang.lang`. In a game, every text is written in both
languages side by side, `L("POLSKI", "ENGLISH")`. The text of the page
itself has its English in `data-en` (and `data-en-title`,
`data-en-aria-label`, `data-en-placeholder` for attributes), and a longer
text, e.g. the rules of a game, is written twice, in elements with
`data-lang="pl"` and `data-lang="en"`, of which the game shows one. Without
`lib/lang.js`, e.g. a game's file opened on its own, a game is in Polish.

Not translated: the sets of questions (they are written by hand, in Polish,
though in English their true / false answers show as TRUE and FALSE), the
word NAUKA (science) of Tajne Laboratorium, whose letters are in its maps,
and the names in the halls of fame.

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
scores. What the games post:

- **Liczożerca**: `score` the points, `time` the seconds of the game,
  `level` the level played. At the end of a game it fetches its list's top
  10, and asks for a name only if the score would make it in (as do the
  others).
- **Tajne Laboratorium** and **Tajne Laboratorium 3D**: `score` the points
  (tasks, diamonds, level and NAUKA bonuses), `time` the seconds of the
  whole game, `level` the task difficulty (results up to `level` × 10) and
  `completed` whether the player got out of the last level.
- **Psia Szkoła**: `score` the points, `time` the seconds of the game and
  `level` the level played (none for a set of questions).
- **Szalona Szosa**: `score` the metres driven (the whole route for everyone
  who reached the finish) and `time` the seconds it took, so finishers rank
  by time, ahead of everyone who ran out of lives, who rank by how far they
  got.
- **Zagubione Zwierzątka**: `score` the points, `time` the seconds of the
  whole game, `level` the last level played (1-4) and `completed` whether
  all four levels were finished (a game can also be ended from the pause
  menu to save its score).

### A set of questions

`questions/<slug>.json`, where the slug is lowercase letters, digits and
dashes:

```json
{"name": "KARTKÓWKA 2", "domain": "math", "position": 1, "fall_seconds": 45, "questions": [
  {"text": "370:10=?", "answer": "37", "wrong": ["370", "3700", "307"]},
  {"type": "tf", "text": "81-0=81×1", "answer": "PRAWDA"}
]}
```

Each set is one more kind of task in Psia Szkoła's menu, in the order of
`position`, under its `domain` (`math`).

- A question has its right `answer` and 1-5 `wrong` ones; the game shows
  the right one and one of the wrong ones, on a random side.
- `"type": "tf"` is a true / false question: its `answer` is `PRAWDA` or
  `FAŁSZ`, and the game always shows PRAWDA on the left and FAŁSZ on the
  right.
- An optional `full` is what a player sees when they get the question
  wrong. By default it is made from the question: a blank, `___`, filled
  in with the answer, or the answer added after a closing `=`.
- `fall_seconds` (optional, 5-600) is the set's speed: how many seconds a
  question takes to fall through the empty well. Without it the game's own
  speed is used (about 33 seconds).
- A game of a set asks every question once, in a random order, and shows
  how many are left (ZOSTAŁO). Answering them all before the pile is full
  ends the game as completed - `CAŁA!` in the hall of fame.
- Names are up to 40 characters, texts up to 80, answers up to 40.

The server checks every file
when it starts and refuses to start with a mistake in one, saying where it
is. A set whose file changed is loaded again as a whole, and a set whose
file is gone is removed, but the scores of its list are kept.

## Building and running it

It needs Go 1.26 or newer, from [go.dev/dl](https://go.dev/dl/) (an older
`go`, e.g. Debian's, downloads 1.26 by itself, unless `GOTOOLCHAIN=local`
is set), and nothing else: SQLite is built in, in pure Go.

```sh
git clone https://github.com/mikogs/html-games.git
cd html-games
go test ./...                                       # the tests, with throwaway databases
CGO_ENABLED=0 go build -trimpath -o gry ./cmd/gry   # a single static binary, ./gry
./gry serve -web web                                # the games and the API on http://127.0.0.1:8090/
```

`./gry serve -web web` keeps its database in `games.db` in the current
directory, and serves both the API and the games, so a game can be played
right away. In production the binary is built the same way, by the
`065_gry.yml` playbook of [devops](https://github.com/mikogs/devops)
(`billy2/`), which runs it as a systemd service behind nginx, and nginx
serves `web/`.

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
