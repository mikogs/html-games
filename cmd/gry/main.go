// Command gry is the server of the games' API, see the README.
//
//	gry serve [-addr 127.0.0.1:8090] [-db games.db] [-web web]
//	gry migrate [-db games.db]
//	gry import-legacy [-db games.db] -from /var/lib/gry/gry.db
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mikogs/gocrud/pkg/registry"

	htmlgames "github.com/mikogs/html-games"
	"github.com/mikogs/html-games/internal/api"
	"github.com/mikogs/html-games/internal/catalog"
	"github.com/mikogs/html-games/internal/store"
)

const usage = `usage:
  gry serve [flags]          migrate, then serve the API
  gry migrate [flags]        bring the database up to date, and load the games and the sets of questions
  gry import-legacy [flags]  copy the hall of fame of the PHP API's database into an empty one
Run "gry <command> -h" for the flags of a command.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	var err error
	switch os.Args[1] {
	case "serve":
		err = serve(os.Args[2:], logger)
	case "migrate":
		err = migrate(os.Args[2:])
	case "import-legacy":
		err = importLegacy(os.Args[2:])
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gry:", err)
		os.Exit(1)
	}
}

func dbFlag(flags *flag.FlagSet) *string {
	return flags.String("db", "games.db", "the SQLite database, created if it does not exist")
}

// open opens the database, applies the migrations and loads the catalog.
func open(ctx context.Context, path string) (*sql.DB, *registry.CRUD, []int64, catalog.Changes, error) {
	db, err := store.Open(path)
	if err != nil {
		return nil, nil, nil, catalog.Changes{}, err
	}
	applied, err := store.Migrate(ctx, db)
	if err != nil {
		db.Close()
		return nil, nil, nil, catalog.Changes{}, err
	}
	reg := store.NewRegistry(db)
	files, err := fs.Sub(htmlgames.Questions, "questions")
	if err != nil {
		db.Close()
		return nil, nil, nil, catalog.Changes{}, err
	}
	changes, err := catalog.Sync(ctx, reg, files)
	if err != nil {
		db.Close()
		return nil, nil, nil, catalog.Changes{}, fmt.Errorf("loading the catalog: %w", err)
	}
	return db, reg, applied, changes, nil
}

func serve(args []string, logger *slog.Logger) error {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := flags.String("addr", "127.0.0.1:8090", "the address to listen on")
	dbPath := dbFlag(flags)
	web := flags.String("web", "", "also serve the games from this directory, e.g. web (for development: nginx serves them in production)")
	accessLog := flags.Bool("access-log", false, "log every request")
	proxies := flags.String("trusted-proxies", "127.0.0.1/32,::1/128", "comma-separated networks of the proxies in front of the API, whose X-Forwarded-For is trusted")
	_ = flags.Parse(args)

	var trusted []netip.Prefix
	for p := range strings.SplitSeq(*proxies, ",") {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(p)
		if err != nil {
			return fmt.Errorf("-trusted-proxies: %w", err)
		}
		trusted = append(trusted, prefix)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, reg, applied, changes, err := open(ctx, *dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	logger.Info("database ready", "db", *dbPath, "migrations_applied", applied, "catalog_changes", changes.String())

	handler, err := api.New(db, reg, api.Options{Logger: logger, AccessLog: *accessLog, TrustedProxies: trusted})
	if err != nil {
		return err
	}
	if *web != "" {
		mux := http.NewServeMux()
		mux.Handle("/", http.FileServer(http.Dir(*web)))
		mux.Handle(api.Prefix+"/", handler)
		mux.Handle("/healthz", handler)
		mux.Handle("/readyz", handler)
		handler = mux
	}

	// the timeouts keep slow clients from holding connections open
	srv := &http.Server{
		Addr: *addr, Handler: handler,
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 2 * time.Minute,
	}
	errs := make(chan error, 1)
	go func() { errs <- srv.ListenAndServe() }()
	logger.Info("listening", "addr", *addr, "web", *web)

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errs; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// migrate prints "up to date" or "migrated: <versions>", then the changes
// of the catalog, if any - the playbook reports a change when it says
// "migrated" or "catalog:".
func migrate(args []string) error {
	flags := flag.NewFlagSet("migrate", flag.ExitOnError)
	dbPath := dbFlag(flags)
	_ = flags.Parse(args)

	ctx := context.Background()
	db, reg, applied, changes, err := open(ctx, *dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	scores, err := reg.Num(ctx, store.KeyScores, registry.Filters{})
	if err != nil {
		return err
	}
	sets, err := reg.Num(ctx, store.KeyQuestionSets, registry.Filters{})
	if err != nil {
		return err
	}
	status := "up to date"
	if len(applied) > 0 {
		status = fmt.Sprint("migrated: ", applied)
	}
	fmt.Printf("%s (%d hall of fame entries, %d sets of questions)\n", status, scores, sets)
	if !changes.Empty() {
		fmt.Println("catalog:", changes)
	}
	return nil
}

// importLegacy copies the hall_of_fame table of the PHP API's database into
// the scores, with their IDs and times. It only imports into a database
// without scores, so it cannot run twice.
func importLegacy(args []string) error {
	flags := flag.NewFlagSet("import-legacy", flag.ExitOnError)
	dbPath := dbFlag(flags)
	from := flags.String("from", "", "the database of the PHP API, e.g. /var/lib/gry/gry.db (only read)")
	_ = flags.Parse(args)
	if *from == "" {
		return errors.New("import-legacy: -from is required")
	}
	if _, err := os.Stat(*from); err != nil {
		return err
	}

	ctx := context.Background()
	db, reg, _, _, err := open(ctx, *dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	if n, err := reg.Num(ctx, store.KeyScores, registry.Filters{}); err != nil {
		return err
	} else if n > 0 {
		return fmt.Errorf("%s already has %d scores - import into a new database", *dbPath, n)
	}

	legacy, err := sql.Open("sqlite", "file:"+*from+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	defer legacy.Close()
	rows, err := legacy.QueryContext(ctx, `SELECT id, game, category, name, score, time_seconds, level, completed, created_at
		FROM hall_of_fame ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var imported int
	err = reg.RunInTx(ctx, func(tx *registry.CRUD) error {
		for rows.Next() {
			var s store.Score
			var createdAt string
			if err := rows.Scan(&s.ID, &s.Game, &s.Category, &s.Name, &s.Score, &s.Time, &s.Level, &s.Completed, &createdAt); err != nil {
				return err
			}
			t, err := time.Parse(time.RFC3339, createdAt)
			if err != nil {
				return fmt.Errorf("entry %d: created_at %q: %w", s.ID, createdAt, err)
			}
			s.CreatedAt = t.UTC()
			s.SortKey = store.SortKey(s.Score, s.Time)
			// straight into the registry, not through api.Rules: the entries
			// were checked when they were posted, and keep their times
			if err := tx.Save(ctx, &s, nil); err != nil {
				return fmt.Errorf("entry %d: %w", s.ID, err)
			}
			imported++
		}
		return rows.Err()
	})
	if err != nil {
		return err
	}
	fmt.Printf("imported %d hall of fame entries from %s\n", imported, *from)
	return nil
}
