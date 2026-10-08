<?php
declare(strict_types=1);

const HOF_DEFAULT_LIMIT = 10;
const HOF_MAX_LIMIT = 100;
const HOF_NAME_MAX = 20;
// Entries without a time rank after every timed one with the same score.
const HOF_NO_TIME = 2147483647;
// Higher score first; on a tie the faster time; on a full tie the older entry.
const HOF_ORDER = 'score DESC, COALESCE(time_seconds, ' . HOF_NO_TIME . ') ASC, id ASC';

function hof_list_games(array $games): array
{
    $out = [];
    foreach ($games as $slug => $game) {
        $out[$slug] = ['title' => $game['title'] ?? $slug, 'categories' => $game['categories'] ?? []];
    }
    return $out;
}

function hof_route(PDO $pdo, array $games, string $slug, string $method): never
{
    $game = $games[$slug] ?? throw new HttpError(404, "unknown game '$slug'");
    $categories = $game['categories'] ?? [];

    if ($method === 'GET') {
        $limit = filter_var($_GET['limit'] ?? HOF_DEFAULT_LIMIT, FILTER_VALIDATE_INT,
            ['options' => ['min_range' => 1, 'max_range' => HOF_MAX_LIMIT]]);
        if ($limit === false) {
            throw new HttpError(400, 'limit must be 1-' . HOF_MAX_LIMIT);
        }
        $wanted = $categories === [] ? [''] : $categories;
        if (isset($_GET['category'])) {
            $wanted = [hof_category($categories, $_GET['category'])];
        }
        $out = [];
        foreach ($wanted as $category) {
            $out[$category] = hof_top($pdo, $slug, $category, $limit);
        }
        json_response(['game' => $slug, 'limit' => $limit, 'categories' => $out]);
    }

    only_methods($method, ['GET', 'POST']);
    $body = read_body();
    $entry = [
        'game' => $slug,
        'category' => hof_category($categories, $body['category'] ?? null),
        'name' => hof_name($body['name'] ?? null),
        'score' => hof_int($body, 'score', 0, 100_000_000, required: true),
        'time_seconds' => hof_int($body, 'time', 0, 7 * 86400),
        'level' => hof_int($body, 'level', 0, 10_000),
        'completed' => filter_var($body['completed'] ?? false, FILTER_VALIDATE_BOOL) ? 1 : 0,
    ];
    $stmt = $pdo->prepare('INSERT INTO hall_of_fame (game, category, name, score, time_seconds, level, completed)
                           VALUES (:game, :category, :name, :score, :time_seconds, :level, :completed)');
    $stmt->execute($entry);
    $id = (int) $pdo->lastInsertId();

    $row = $pdo->query('SELECT * FROM hall_of_fame WHERE id = ' . $id)->fetch();
    json_response(['entry' => hof_format($row), 'rank' => hof_rank($pdo, $row)], 201);
}

function hof_top(PDO $pdo, string $slug, string $category, int $limit): array
{
    $stmt = $pdo->prepare('SELECT * FROM hall_of_fame WHERE game = ? AND category = ? ORDER BY ' . HOF_ORDER . ' LIMIT ?');
    $stmt->bindValue(1, $slug);
    $stmt->bindValue(2, $category);
    $stmt->bindValue(3, $limit, PDO::PARAM_INT);
    $stmt->execute();
    return array_map('hof_format', $stmt->fetchAll());
}

// 1-based place of a stored entry within its game + category.
function hof_rank(PDO $pdo, array $row): int
{
    $stmt = $pdo->prepare('SELECT COUNT(*) FROM hall_of_fame
        WHERE game = :game AND category = :category AND (
            score > :score
            OR (score = :score AND COALESCE(time_seconds, ' . HOF_NO_TIME . ') < :time)
            OR (score = :score AND COALESCE(time_seconds, ' . HOF_NO_TIME . ') = :time AND id < :id))');
    // Bound as integers: compared with COALESCE(...), which has no column
    // affinity, a text parameter would sort after every number.
    $stmt->bindValue('game', $row['game']);
    $stmt->bindValue('category', $row['category']);
    $stmt->bindValue('score', (int) $row['score'], PDO::PARAM_INT);
    $stmt->bindValue('time', (int) ($row['time_seconds'] ?? HOF_NO_TIME), PDO::PARAM_INT);
    $stmt->bindValue('id', (int) $row['id'], PDO::PARAM_INT);
    $stmt->execute();
    return (int) $stmt->fetchColumn() + 1;
}

function hof_format(array $row): array
{
    return [
        'id' => (int) $row['id'],
        'name' => $row['name'],
        'score' => (int) $row['score'],
        'time' => $row['time_seconds'] === null ? null : (int) $row['time_seconds'],
        'level' => $row['level'] === null ? null : (int) $row['level'],
        'completed' => (bool) $row['completed'],
        'created_at' => $row['created_at'],
    ];
}

function hof_category(array $categories, mixed $value): string
{
    if ($categories === []) {
        return '';
    }
    if (!is_string($value) || !in_array($value, $categories, true)) {
        throw new HttpError(400, 'category must be one of: ' . implode(', ', $categories));
    }
    return $value;
}

function hof_name(mixed $value): string
{
    if (!is_string($value)) {
        throw new HttpError(400, 'name is required');
    }
    // Drop control characters, collapse whitespace.
    $name = trim(preg_replace('/\s+/u', ' ', preg_replace('/\p{C}/u', '', $value) ?? '') ?? '');
    if ($name === '') {
        throw new HttpError(400, 'name is required');
    }
    if (mb_strlen($name) > HOF_NAME_MAX) {
        throw new HttpError(400, 'name can be at most ' . HOF_NAME_MAX . ' characters');
    }
    return $name;
}

function hof_int(array $body, string $key, int $min, int $max, bool $required = false): ?int
{
    if (!isset($body[$key]) || $body[$key] === '') {
        if ($required) {
            throw new HttpError(400, "$key is required");
        }
        return null;
    }
    $value = is_float($body[$key]) ? (int) floor($body[$key]) : $body[$key];
    $int = filter_var($value, FILTER_VALIDATE_INT, ['options' => ['min_range' => $min, 'max_range' => $max]]);
    if ($int === false) {
        throw new HttpError(400, "$key must be a whole number from $min to $max");
    }
    return $int;
}
