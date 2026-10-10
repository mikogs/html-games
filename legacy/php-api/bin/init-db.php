<?php
// Creates the database (or brings its schema up to date) - run by
// 065_gry.yml as the php-fpm user, so the file belongs to the user that
// writes to it. Then loads the sets of questions from api/zadania/*.json
// (lib/questions.php). Prints "migrated to <n>" and / or "questions:" when
// it changed something.
// Not reachable from the web: nginx only ever runs api/index.php.
declare(strict_types=1);

if (PHP_SAPI !== 'cli') {
    exit(1);
}

require __DIR__ . '/../lib/http.php';
require __DIR__ . '/../lib/db.php';
require __DIR__ . '/../lib/questions.php';

$config = load_config(__DIR__ . '/../config.php');
$pdo = new PDO('sqlite:' . $config['db_path'], null, null, [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]);
$pdo->exec('PRAGMA busy_timeout = 5000');
$pdo->exec('PRAGMA journal_mode = WAL');
$applied = db_migrate($pdo);
$version = (int) $pdo->query('PRAGMA user_version')->fetchColumn();
$entries = (int) $pdo->query('SELECT COUNT(*) FROM hall_of_fame')->fetchColumn();
try {
    $changes = q_import_dir($pdo, __DIR__ . '/../zadania');
} catch (RuntimeException $e) {
    // a mistake in a file of questions: say which and where, and change nothing
    fwrite(STDERR, 'questions not loaded - ' . $e->getMessage() . "\n");
    exit(1);
}
$sets = (int) $pdo->query('SELECT COUNT(*) FROM question_sets')->fetchColumn();

echo $applied === [] ? "up to date" : "migrated to $version", " (schema v$version, $entries hall of fame entries, $sets sets of questions)\n";
if ($changes !== []) {
    echo 'questions: ', implode(', ', array_map(fn($slug, $what) => "$slug $what", array_keys($changes), $changes)), "\n";
}
