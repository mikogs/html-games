// Command migration writes a migration file with the changes that the
// structs of internal/store need, e.g. after adding a field:
//
//	go run ./cmd/migration add_points
//
// It applies the existing files to an empty database first. Read the file
// it writes: complete the -- TODO: parts and write the Down part.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mikogs/gocrud"
	"github.com/pressly/goose/v3"

	"github.com/mikogs/html-games/internal/store"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: migration <name>")
	}
	ctx := context.Background()

	dir, err := os.MkdirTemp("", "migration")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)
	db, err := store.Open(filepath.Join(dir, "dev.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// The existing files only, without store.Migrate's check that they match
	// the structs - they don't yet, which is why this runs.
	files, err := filepath.Glob("internal/store/migrations/*.sql")
	if err != nil {
		log.Fatal(err)
	}
	if len(files) > 0 {
		provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS("internal/store/migrations"))
		if err != nil {
			log.Fatal(err)
		}
		if _, err := provider.Up(ctx); err != nil {
			log.Fatalf("applying the migrations: %v", err)
		}
	}

	plans, err := store.NewRegistry(db).PlanMigrations(ctx)
	if err != nil {
		log.Fatal(err)
	}
	up := gocrud.MigrationSQL(plans...)
	if up == "" {
		fmt.Println("the migrations match the structs, there is nothing to write")
		return
	}
	name := filepath.Join("internal/store/migrations", time.Now().UTC().Format("20060102150405")+"_"+os.Args[1]+".sql")
	content := "-- +goose Up\n" + up + "\n-- +goose Down\n-- TODO: undo the changes above\n"
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("written", name)
}
