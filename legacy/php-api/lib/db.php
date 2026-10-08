<?php
declare(strict_types=1);

// One SQLite database shared by every game's API. The schema is versioned
// with SQLite's user_version: append to MIGRATIONS (never edit an applied
// one) and every connection brings the file up to date.
const MIGRATIONS = [
    1 => <<<'SQL'
        CREATE TABLE hall_of_fame (
            id           INTEGER PRIMARY KEY AUTOINCREMENT,
            game         TEXT    NOT NULL,
            category     TEXT    NOT NULL DEFAULT '',
            name         TEXT    NOT NULL,
            score        INTEGER NOT NULL,
            time_seconds INTEGER,
            level        INTEGER,
            completed    INTEGER NOT NULL DEFAULT 0,
            created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
        );
        CREATE INDEX hall_of_fame_ranking ON hall_of_fame (game, category, score DESC, time_seconds);
        SQL,
];

function db(string $path): PDO
{
    static $pdo = null;
    if ($pdo === null) {
        $pdo = new PDO('sqlite:' . $path, null, null, [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
        ]);
        $pdo->exec('PRAGMA busy_timeout = 5000');
        $pdo->exec('PRAGMA journal_mode = WAL');
        db_migrate($pdo);
    }
    return $pdo;
}

// Returns the versions it applied (empty when already up to date).
function db_migrate(PDO $pdo): array
{
    $applied = [];
    $pdo->exec('BEGIN IMMEDIATE');
    try {
        $version = (int) $pdo->query('PRAGMA user_version')->fetchColumn();
        foreach (MIGRATIONS as $to => $sql) {
            if ($to > $version) {
                $pdo->exec($sql);
                $pdo->exec('PRAGMA user_version = ' . (int) $to);
                $applied[] = $to;
            }
        }
        $pdo->exec('COMMIT');
    } catch (Throwable $e) {
        $pdo->exec('ROLLBACK');
        throw $e;
    }
    return $applied;
}
