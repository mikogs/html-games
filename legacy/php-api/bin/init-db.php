<?php
// Creates the database (or brings its schema up to date) - run by
// 065_gry.yml as the php-fpm user, so the file belongs to the user that
// writes to it. Prints "migrated to <n>" when it changed something.
// Not reachable from the web: nginx only ever runs api/index.php.
declare(strict_types=1);

if (PHP_SAPI !== 'cli') {
    exit(1);
}

require __DIR__ . '/../lib/http.php';
require __DIR__ . '/../lib/db.php';

$config = load_config(__DIR__ . '/../config.php');
$pdo = new PDO('sqlite:' . $config['db_path'], null, null, [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]);
$pdo->exec('PRAGMA busy_timeout = 5000');
$pdo->exec('PRAGMA journal_mode = WAL');
$applied = db_migrate($pdo);
$version = (int) $pdo->query('PRAGMA user_version')->fetchColumn();
$entries = (int) $pdo->query('SELECT COUNT(*) FROM hall_of_fame')->fetchColumn();

echo $applied === [] ? "up to date" : "migrated to $version", " (schema v$version, $entries hall of fame entries)\n";
