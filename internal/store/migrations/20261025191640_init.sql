-- +goose Up
-- table "game"
CREATE TABLE IF NOT EXISTS "game" ("id" INTEGER PRIMARY KEY,"slug" TEXT NOT NULL DEFAULT '' UNIQUE,"title" TEXT NOT NULL DEFAULT '',"categories" TEXT NOT NULL DEFAULT 'null',"position" INTEGER NOT NULL DEFAULT 0);

-- table "question_set"
CREATE TABLE IF NOT EXISTS "question_set" ("id" INTEGER PRIMARY KEY,"slug" TEXT NOT NULL DEFAULT '' UNIQUE,"name" TEXT NOT NULL DEFAULT '',"domain" TEXT NOT NULL DEFAULT '',"position" INTEGER NOT NULL DEFAULT 0,"fall_seconds" INTEGER,"question_count" INTEGER NOT NULL DEFAULT 0,"source_hash" TEXT NOT NULL DEFAULT '',"updated_at" DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00.000000000+00:00');

-- table "question"
CREATE TABLE IF NOT EXISTS "question" ("id" INTEGER PRIMARY KEY,"set_id" INTEGER NOT NULL DEFAULT 0 REFERENCES "question_set"("id") ON DELETE CASCADE,"position" INTEGER NOT NULL DEFAULT 0,"type" TEXT NOT NULL DEFAULT '',"text" TEXT NOT NULL DEFAULT '',"answer" TEXT NOT NULL DEFAULT '',"wrong" TEXT NOT NULL DEFAULT 'null',"full" TEXT NOT NULL DEFAULT '');
CREATE INDEX IF NOT EXISTS "idx_question_set_id" ON "question" ("set_id");

-- table "score"
CREATE TABLE IF NOT EXISTS "score" ("id" INTEGER PRIMARY KEY,"game" TEXT NOT NULL DEFAULT '',"category" TEXT NOT NULL DEFAULT '',"name" TEXT NOT NULL DEFAULT '',"score" INTEGER NOT NULL DEFAULT 0,"time" INTEGER,"level" INTEGER,"completed" INTEGER NOT NULL DEFAULT 0,"sort_key" INTEGER NOT NULL DEFAULT 0,"created_at" DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00.000000000+00:00');
CREATE INDEX IF NOT EXISTS "idx_score_score_ranking" ON "score" ("game","category","sort_key");

-- +goose Down
DROP TABLE "score";
DROP TABLE "question";
DROP TABLE "question_set";
DROP TABLE "game";
