package main

import (
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"thufir/internal/auth"
	"thufir/internal/config"
	"thufir/internal/db"
	mw "thufir/internal/middleware"
	"thufir/internal/sync"
)

func main() {
	cfg := config.FromEnv()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	pool := db.NewPool(cfg.DatabaseURL)
	wa := auth.NewWebAuthn(cfg)
	cs := auth.NewChallengeStore()

	// Sub-FS rooted at the embedded `static/` directory.
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("embed: %v", err)
	}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware(cfg))

	// ── health ─────────────────────────────────────────────────────────────────
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	// ── auth routes (no session required) ─────────────────────────────────────
	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/status", auth.HandleStatus(pool))
		r.Get("/me", auth.HandleMe(pool))
		r.Post("/logout", auth.HandleLogout(pool))

		r.Post("/setup/options", auth.HandleSetupOptions(pool, wa, cs, cfg))
		r.Post("/setup/verify", auth.HandleSetupVerify(pool, wa, cs, cfg))

		r.Post("/login/options", auth.HandleLoginOptions(wa, cs, cfg))
		r.Post("/login/verify", auth.HandleLoginVerify(pool, wa, cs, cfg))

		r.Post("/device/options", auth.HandleDeviceOptions(pool, wa, cs, cfg))
		r.Post("/device/verify", auth.HandleDeviceVerify(pool, wa, cs, cfg))
		r.Get("/devices", auth.HandleListDevices(pool))
		r.Delete("/devices/{id}", auth.HandleDeleteDevice(pool))
	})

	// ── RxDB replication routes (session required) ────────────────────────────
	r.Route("/api/rxdb", func(r chi.Router) {
		r.Use(mw.RequireAuth(pool))
		for _, collection := range []string{"tasks", "projects", "areas"} {
			c := collection // capture loop var
			r.Post("/"+c+"/pull", sync.HandlePull(c, pool))
			r.Post("/"+c+"/push", sync.HandlePush(c, pool))
		}
	})

	// ── frontend (SPA, embedded) ───────────────────────────────────────────────
	r.NotFound(spaHandler(subFS))

	log.Printf("Thufir listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}

// corsMiddleware sets permissive CORS headers for allowed origins.
func corsMiddleware(cfg config.Config) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowed[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// spaHandler serves files from the embedded FS and falls back to index.html
// for any path that doesn't match a real file (client-side routing).
func spaHandler(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))

	// Read index.html once at startup; it will be served for all SPA routes.
	indexHTML, indexErr := fs.ReadFile(fsys, "index.html")

	return func(w http.ResponseWriter, r *http.Request) {
		// Strip the leading slash to get an fs.FS-relative path.
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}

		// Check whether the path maps to a real (non-directory) file.
		f, err := fsys.Open(name)
		if err == nil {
			stat, statErr := f.Stat()
			f.Close()
			if statErr == nil && !stat.IsDir() {
				// Immutable cache for SvelteKit's content-hashed bundles.
				if strings.HasPrefix(r.URL.Path, "/_app/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "no-cache")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// No matching file → serve the SPA shell.
		if indexErr != nil {
			http.Error(w, "Frontend not embedded. Build with: npm run build", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML) //nolint:errcheck
	}
}
