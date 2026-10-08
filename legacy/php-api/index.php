<?php
// Front controller for https://gry.billy.local/api/ - nginx sends every
// request under /api/ here, so nothing else in this directory is ever run
// as PHP from the web.
//
//   GET  /api/                           what's here
//   GET  /api/sala_slawy/                the games that have a hall of fame
//   GET  /api/sala_slawy/<game>/         top entries per category (?category=, ?limit=)
//   POST /api/sala_slawy/<game>/         add an entry (JSON: name, score, category, time, level, completed)
//
// No authentication on purpose (for now) - anyone can post any score.
declare(strict_types=1);

require __DIR__ . '/lib/http.php';
require __DIR__ . '/lib/db.php';
require __DIR__ . '/lib/halloffame.php';

send_cors_headers();
if (($_SERVER['REQUEST_METHOD'] ?? 'GET') === 'OPTIONS') {
    http_response_code(204);
    exit;
}

try {
    $config = load_config(__DIR__ . '/config.php');
    $games = require __DIR__ . '/games.php';

    $path = parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH) ?: '/';
    $path = preg_replace('#^/api(?=/|$)#', '', $path);
    $parts = array_values(array_filter(explode('/', $path), 'strlen'));
    $method = $_SERVER['REQUEST_METHOD'] ?? 'GET';

    match (true) {
        $parts === [] => only_methods($method, ['GET']) ?? json_response([
            'name' => 'gry.billy.local api',
            'endpoints' => ['/api/sala_slawy/', '/api/sala_slawy/{game}/'],
        ]),
        $parts === ['sala_slawy'] => only_methods($method, ['GET']) ?? json_response([
            'games' => hof_list_games($games),
        ]),
        count($parts) === 2 && $parts[0] === 'sala_slawy' => hof_route(db($config['db_path']), $games, $parts[1], $method),
        default => throw new HttpError(404, 'not found'),
    };
} catch (HttpError $e) {
    json_response(['error' => $e->getMessage()], $e->status);
} catch (Throwable $e) {
    error_log('gry api: ' . $e);
    json_response(['error' => 'internal error'], 500);
}
