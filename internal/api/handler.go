package api

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/mikogs/gocrud/pkg/registry"
	"github.com/mikogs/gorestapi"
	"github.com/rs/cors"

	"github.com/mikogs/html-games/internal/store"
)

// Prefix is where the API is mounted.
const Prefix = "/api/v1"

// Options configure New.
type Options struct {
	Logger *slog.Logger
	// AccessLog logs every request.
	AccessLog bool
	// TrustedProxies are the proxies in front of the API, e.g. nginx on
	// 127.0.0.1, whose X-Forwarded-For has the IP of the client.
	TrustedProxies []netip.Prefix
}

const readOnly = gorestapi.DisableCreate | gorestapi.DisableUpdate | gorestapi.DisableDelete

// Routes are the endpoints, see the README. Everything but scores is only
// read: it is written by catalog.Sync.
var Routes = map[string]gorestapi.Route{
	store.KeyGames:        {Flags: readOnly},
	store.KeyScores:       {Flags: gorestapi.DisableUpdate | gorestapi.DisableDelete},
	store.KeyQuestionSets: {Flags: readOnly},
	store.KeyQuestions:    {Flags: readOnly},
}

// New returns the handler of the API at Prefix, its OpenAPI document at
// Prefix/openapi.json, and the health checks at /healthz and /readyz.
func New(db *sql.DB, reg *registry.CRUD, options Options) (http.Handler, error) {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	h := gorestapi.New(&Rules{CRUD: reg}, gorestapi.Options{
		Routes:         Routes,
		Logger:         logger,
		AccessLog:      options.AccessLog,
		TrustedProxies: options.TrustedProxies,
		// generous: a game fetches each list of its hall of fame at once
		RateLimiter:  gorestapi.NewRateLimiter(20, 60),
		MaxBodyBytes: 16 << 10,
	})
	spec, err := h.OpenAPI(gorestapi.OpenAPIInfo{
		Title:       "Games",
		Version:     "1.0.0",
		Description: "The halls of fame of the games and the sets of questions written by hand.",
		Servers:     []string{Prefix},
	})
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle(Prefix+"/", http.StripPrefix(Prefix, http.HandlerFunc(h.Serve)))
	mux.HandleFunc("GET "+Prefix+"/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(spec)
	})
	mux.HandleFunc("/healthz", gorestapi.Liveness)
	mux.Handle("/readyz", h.Readiness(func(ctx context.Context) error { return db.PingContext(ctx) }))

	// The games can be opened straight from a file (file://), which makes
	// every call cross-origin - and the data is public, without credentials.
	return cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet, http.MethodPut},
		AllowedHeaders: []string{"Content-Type", gorestapi.RequestIDHeader},
		ExposedHeaders: []string{gorestapi.RequestIDHeader, "Allow", "Retry-After"},
		MaxAge:         86400,
	}).Handler(mux), nil
}
