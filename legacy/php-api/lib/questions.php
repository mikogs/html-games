<?php
declare(strict_types=1);

// Sets of questions written by hand for the games (e.g. a test from school),
// kept in the database. Their source is api/zadania/<slug>.json - one file
// per set - which bin/init-db.php loads into the database on every deploy.
//
// A file: {"name": "KARTKÓWKA 2", "domain": "math", "position": 1, "fall_seconds": 45, "questions": [...]}
// ("fall_seconds", optional: how long a question takes to fall through the empty well - its speed;
// without it the game's own speed)
// and each question one of:
//   {"text": "370:10=?", "answer": "37", "wrong": ["370", "3700", "307"]}
//       1-5 wrong answers; the game shows the right one and one of them
//   {"type": "tf", "text": "81-0=81×1", "answer": "PRAWDA"}
//       true or false: the answer is PRAWDA or FAŁSZ, and the other one is the wrong one
// plus, optionally, "full": the question with its answer, shown when a player
// gets it wrong (by default it is made from the text and the answer: a blank,
// ___, filled in with the answer, or the answer after a closing "=").

const Q_TEXT_MAX = 80;
const Q_ANSWER_MAX = 40;
const Q_WRONG_MAX = 5;
const Q_TRUE = 'PRAWDA';
const Q_FALSE = 'FAŁSZ';
const Q_FALL_MIN = 5;
const Q_FALL_MAX = 600;
// part of every set's hash: bump it when what is stored from a file changes, so every set is loaded again
const Q_FORMAT = 2;

// Loads every api/zadania/*.json into the database: a set whose file
// changed is replaced as a whole, a set whose file is gone is removed.
// Returns [slug => 'added' | 'updated' | 'removed'] for what changed.
function q_import_dir(PDO $pdo, string $dir): array
{
    $files = glob(rtrim($dir, '/') . '/*.json') ?: [];
    $sets = [];
    foreach ($files as $file) {
        $slug = basename($file, '.json');
        if (!preg_match('/^[a-z0-9]+(-[a-z0-9]+)*$/', $slug)) {
            throw new RuntimeException("$file: the file name must be lowercase letters, digits and dashes");
        }
        $raw = file_get_contents($file);
        $sets[$slug] = ['hash' => sha1(Q_FORMAT . "\n" . $raw), 'set' => q_parse_set($file, $raw)];
    }

    $changes = [];
    $pdo->exec('BEGIN IMMEDIATE');
    try {
        $known = $pdo->query('SELECT slug, source_hash FROM question_sets')->fetchAll(PDO::FETCH_KEY_PAIR);
        foreach ($sets as $slug => ['hash' => $hash, 'set' => $set]) {
            if (($known[$slug] ?? null) === $hash) {
                continue;
            }
            $changes[$slug] = isset($known[$slug]) ? 'updated' : 'added';
            $pdo->prepare('DELETE FROM questions WHERE set_slug = ?')->execute([$slug]);
            $pdo->prepare('INSERT INTO question_sets (slug, name, domain, position, fall_seconds, source_hash)
                           VALUES (:slug, :name, :domain, :position, :fall, :hash)
                           ON CONFLICT (slug) DO UPDATE SET name = :name, domain = :domain, position = :position, fall_seconds = :fall,
                           source_hash = :hash, updated_at = strftime(\'%Y-%m-%dT%H:%M:%SZ\', \'now\')')
                ->execute(['slug' => $slug, 'name' => $set['name'], 'domain' => $set['domain'], 'position' => $set['position'],
                           'fall' => $set['fall_seconds'], 'hash' => $hash]);
            $insert = $pdo->prepare('INSERT INTO questions (set_slug, position, type, text, answer, wrong, full)
                                     VALUES (?, ?, ?, ?, ?, ?, ?)');
            foreach ($set['questions'] as $i => $q) {
                $insert->execute([$slug, $i + 1, $q['type'], $q['text'], $q['answer'],
                                  json_encode($q['wrong'], JSON_UNESCAPED_UNICODE | JSON_THROW_ON_ERROR), $q['full']]);
            }
        }
        foreach (array_diff(array_keys($known), array_keys($sets)) as $slug) {
            $pdo->prepare('DELETE FROM questions WHERE set_slug = ?')->execute([$slug]);
            $pdo->prepare('DELETE FROM question_sets WHERE slug = ?')->execute([$slug]);
            $changes[$slug] = 'removed';
        }
        $pdo->exec('COMMIT');
    } catch (Throwable $e) {
        $pdo->exec('ROLLBACK');
        throw $e;
    }
    return $changes;
}

// Checks a set's file and fills in what a question may leave out.
function q_parse_set(string $file, string $raw): array
{
    $fail = fn(string $why) => throw new RuntimeException("$file: $why");
    try {
        $data = json_decode($raw, true, 16, JSON_THROW_ON_ERROR);
    } catch (JsonException $e) {
        $fail('not valid JSON (' . $e->getMessage() . ')');
    }
    $name = trim((string) ($data['name'] ?? ''));
    if ($name === '' || mb_strlen($name) > 40) {
        $fail('"name" must be 1-40 characters');
    }
    $questions = $data['questions'] ?? null;
    if (!is_array($questions) || $questions === [] || !array_is_list($questions)) {
        $fail('"questions" must be a non-empty list');
    }
    $fall = $data['fall_seconds'] ?? null;
    if ($fall !== null && (!is_int($fall) || $fall < Q_FALL_MIN || $fall > Q_FALL_MAX)) {
        $fail('"fall_seconds" must be a whole number of seconds, ' . Q_FALL_MIN . '-' . Q_FALL_MAX);
    }
    $out = [];
    foreach ($questions as $i => $q) {
        $where = 'question ' . ($i + 1);
        $type = $q['type'] ?? 'choice';
        $text = trim((string) ($q['text'] ?? ''));
        $answer = trim((string) ($q['answer'] ?? ''));
        if ($text === '' || mb_strlen($text) > Q_TEXT_MAX) {
            $fail("$where: \"text\" must be 1-" . Q_TEXT_MAX . ' characters');
        }
        if ($type === 'tf') {
            if (!in_array($answer, [Q_TRUE, Q_FALSE], true)) {
                $fail("$where: a true / false question's answer must be " . Q_TRUE . ' or ' . Q_FALSE);
            }
            $wrong = [$answer === Q_TRUE ? Q_FALSE : Q_TRUE];
        } elseif ($type === 'choice') {
            $wrong = array_map(fn($w) => trim((string) $w), $q['wrong'] ?? []);
            if ($answer === '' || mb_strlen($answer) > Q_ANSWER_MAX) {
                $fail("$where: \"answer\" must be 1-" . Q_ANSWER_MAX . ' characters');
            }
            if ($wrong === [] || count($wrong) > Q_WRONG_MAX || count(array_unique($wrong)) !== count($wrong)
                || in_array($answer, $wrong, true) || in_array('', $wrong, true)) {
                $fail("$where: \"wrong\" must be 1-" . Q_WRONG_MAX . ' different answers, none of them the right one');
            }
            foreach ($wrong as $w) {
                if (mb_strlen($w) > Q_ANSWER_MAX) {
                    $fail("$where: a wrong answer longer than " . Q_ANSWER_MAX . ' characters');
                }
            }
        } else {
            $fail("$where: \"type\" must be choice (the default) or tf");
        }
        $full = trim((string) ($q['full'] ?? ''));
        if ($full === '') {
            $full = match (true) {
                $type === 'tf' => "$text - $answer",
                str_contains($text, '___') => str_replace('___', $answer, $text),
                (bool) preg_match('/=\s*\??$/u', $text) => preg_replace('/\s*\?$/u', '', $text) . $answer,
                default => "$text: $answer",
            };
        }
        $out[] = ['type' => $type, 'text' => $text, 'answer' => $answer, 'wrong' => array_values($wrong), 'full' => $full];
    }
    return [
        'name' => $name,
        'domain' => (string) ($data['domain'] ?? 'math'),
        'position' => (int) ($data['position'] ?? 0),
        'fall_seconds' => $fall,
        'questions' => $out,
    ];
}

// The sets (without their questions), in their order.
function q_sets(PDO $pdo): array
{
    return $pdo->query('SELECT s.slug, s.name, s.domain, s.fall_seconds, COUNT(q.id) AS count FROM question_sets s
                        LEFT JOIN questions q ON q.set_slug = s.slug GROUP BY s.slug ORDER BY s.position, s.name')
        ->fetchAll();
}

//   GET /api/zadania/         the sets
//   GET /api/zadania/<slug>/  one set with all its questions
function q_route(PDO $pdo, ?string $slug, string $method): never
{
    only_methods($method, ['GET']);
    if ($slug === null) {
        json_response(['sets' => array_map(fn($s) => [...$s, 'count' => (int) $s['count']], q_sets($pdo))]);
    }
    $stmt = $pdo->prepare('SELECT slug, name, domain, fall_seconds FROM question_sets WHERE slug = ?');
    $stmt->execute([$slug]);
    $set = $stmt->fetch() ?: throw new HttpError(404, "unknown set '$slug'");
    $stmt = $pdo->prepare('SELECT type, text, answer, wrong, full FROM questions WHERE set_slug = ? ORDER BY position');
    $stmt->execute([$slug]);
    $set['questions'] = array_map(fn($q) => [...$q, 'wrong' => json_decode($q['wrong'], true)], $stmt->fetchAll());
    json_response($set);
}
