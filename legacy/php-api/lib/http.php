<?php
declare(strict_types=1);

final class HttpError extends RuntimeException
{
    public function __construct(public readonly int $status, string $message, public readonly array $headers = [])
    {
        parent::__construct($message);
    }
}

// The games can also be opened straight from a file (file://), which makes
// every call cross-origin - so any origin is allowed.
function send_cors_headers(): void
{
    header('Access-Control-Allow-Origin: *');
    header('Access-Control-Allow-Methods: GET, POST, OPTIONS');
    header('Access-Control-Allow-Headers: Content-Type');
    header('Access-Control-Max-Age: 86400');
}

function json_response(array $data, int $status = 200): never
{
    http_response_code($status);
    header('Content-Type: application/json; charset=utf-8');
    header('Cache-Control: no-store');
    echo json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_THROW_ON_ERROR);
    exit;
}

// Returns null when the method is allowed, so callers can write
// `only_methods(...) ?? handle()`.
function only_methods(string $method, array $allowed): ?bool
{
    if (!in_array($method, $allowed, true)) {
        header('Allow: ' . implode(', ', [...$allowed, 'OPTIONS']));
        throw new HttpError(405, 'method not allowed');
    }
    return null;
}

// JSON body, or a regular form post as a fallback.
function read_body(int $max_bytes = 16384): array
{
    $raw = file_get_contents('php://input', false, null, 0, $max_bytes + 1);
    if ($raw === false) {
        throw new HttpError(400, 'could not read the request body');
    }
    if (strlen($raw) > $max_bytes) {
        throw new HttpError(413, 'request body too large');
    }
    $type = strtolower($_SERVER['CONTENT_TYPE'] ?? '');
    if (str_starts_with($type, 'application/json') || ($raw !== '' && $_POST === [])) {
        try {
            $data = json_decode($raw, true, 8, JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            throw new HttpError(400, 'body is not valid JSON');
        }
        if (!is_array($data)) {
            throw new HttpError(400, 'body must be a JSON object');
        }
        return $data;
    }
    return $_POST;
}

// config.php is written by the 065_gry.yml playbook (roles/gry's
// templates/config.php.j2) and isn't part of the sources.
function load_config(string $file): array
{
    if (!is_file($file)) {
        throw new RuntimeException("$file is missing - deploy with 065_gry.yml");
    }
    $config = require $file;
    if (!is_array($config) || !isset($config['db_path'])) {
        throw new RuntimeException("$file must return ['db_path' => ...]");
    }
    return $config;
}
