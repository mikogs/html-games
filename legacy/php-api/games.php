<?php
// Games with a hall of fame. A game's entries are ranked separately per
// category (an empty list means the game has a single, unnamed one).
// Add a game here before it can post scores.
declare(strict_types=1);

return [
    'liczozerca' => [
        'title' => 'Liczożerca',
        'categories' => ['add', 'sub', 'mul', 'div'],
    ],
    // one list per kind of task; score = points, time = seconds of the whole game,
    // level = task difficulty (results up to level * 10), completed = escaped from the last level
    'laboratorium' => [
        'title' => 'Tajne Laboratorium',
        'categories' => ['add', 'sub', 'mul', 'div', 'all'],
    ],
    // one list per domain and kind of task (only maths so far), and one per set of questions
    // (set-<slug>); score = points (10 per right answer), time = seconds of the game,
    // level = the level played (results up to level * 10; none for a set of questions)
    'psiaszkola' => [
        'title' => 'Psia Szkoła',
        'categories' => ['math-add', 'math-sub', 'math-mul', 'math-div', 'math-all'],
        'question_sets' => true,
    ],
    // score = metres driven (the whole route when finished), time = seconds to get there
    'szosa' => [
        'title' => 'Szalona Szosa',
        'categories' => [],
    ],
];
