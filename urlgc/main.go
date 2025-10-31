package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"urlgc/internal/server"
)

// Service encapsulates business logic.
type Service struct {
	store *server.Store
	rng   *rand.Rand
}

func NewService(s *server.Store) *Service {
	return &Service{store: s, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

const codeLen = 8

var codeAlphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func (svc *Service) generateCode() string {
	b := make([]rune, codeLen)
	for i := range b {
		b[i] = codeAlphabet[svc.rng.Intn(len(codeAlphabet))]
	}
	return string(b)
}

var urlRx = regexp.MustCompile(`^(?i)(https?|ftp)://`) // require scheme

func normalizeURL(u string) (string, error) {
	u = strings.TrimSpace(u)
	if u == "" {
		return "", errors.New("empty url")
	}
	if !urlRx.MatchString(u) {
		u = "https://" + u
	}
	return u, nil
}

func (svc *Service) CreateShort(ctx context.Context, target string) (string, error) {
	t, err := normalizeURL(target)
	if err != nil {
		return "", err
	}
	// attempt a few times to avoid collisions
	for i := 0; i < 5; i++ {
		c := svc.generateCode()
		if err := svc.store.Put(c, t); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("could not allocate code")
}

func (svc *Service) Resolve(ctx context.Context, code string) (string, bool) {
	return svc.store.Get(code)
}

// API handlers
func apiCreateHandler(svc *Service, baseURL string) http.HandlerFunc {
	type req struct {
		Target string `json:"target"`
	}
	type resp struct {
		Code     string `json:"code"`
		ShortURL string `json:"shortUrl"`
		Target   string `json:"target"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		code, err := svc.CreateShort(r.Context(), body.Target)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		shortURL := strings.TrimRight(baseURL, "/") + "/" + code
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp{Code: code, ShortURL: shortURL, Target: body.Target})
	}
}

func apiHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}
}

// Redirect or 404 page
func redirectHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		// only accept exactly 8 base62
		if len(path) != codeLen || !isBase62(path) {
			notFoundHTML(w)
			return
		}
		if target, ok := svc.Resolve(r.Context(), path); ok {
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
		notFoundHTML(w)
	}
}

func isBase62(s string) bool {
	for _, ch := range s {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}

// notFoundHTML serves a simple error page
func notFoundHTML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>Not Found</title><style>body{font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Helvetica,Arial,sans-serif;padding:40px;max-width:720px;margin:auto} .card{border:1px solid #e5e7eb;border-radius:12px;padding:24px} a{color:#2563eb}</style></head><body><div class="card"><h1>Link not found</h1><p>The short URL you followed does not exist. Create one at <a href="/app">the app</a>.</p></div></body></html>`))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Initialize database
	db, err := server.NewDB("./data/sqlite.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Prepare components
	store := server.NewStore()

	// Load existing data from database into memory
	if err := db.LoadIntoStore(store); err != nil {
		log.Printf("Warning: failed to load data from database: %v", err)
	}

	// Start periodic sync (every 15 minutes)
	stopCh := make(chan struct{})
	go db.PeriodicSync(store, 15*time.Minute, stopCh)

	svc := NewService(store)

	// Base URL for returned short links
	host := os.Getenv("BASE_URL")
	if host == "" {
		host = "http://localhost:8080"
	}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", apiHealthHandler())
	mux.HandleFunc("/api/urls", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			apiCreateHandler(svc, host).ServeHTTP(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Serve the single-page app
	fs := http.FileServer(http.Dir("./web/spa/dist"))
	mux.Handle("/app/", http.StripPrefix("/app/", fs))

	mux.HandleFunc("/", redirectHandler(svc))

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	srv := &http.Server{Addr: addr, Handler: withCORS(mux)}

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down gracefully...")
		
		// Stop periodic sync
		close(stopCh)

		// Final sync to database
		if err := db.BulkPut(store.GetAll()); err != nil {
			log.Printf("Error during final database sync: %v", err)
		} else {
			log.Println("Final database sync completed")
		}

		// Shutdown HTTP server
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("URL Shortener listening on %s", addr)
	log.Printf("Open the app at %s/app", host)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
