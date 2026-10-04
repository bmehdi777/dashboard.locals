package httpserver

import (
	"context"
	"io/fs"
	"log"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"dashboard.locals/internal/http/api/v1"
	"dashboard.locals/internal/http/webassets"
)

type Config struct {
	API      v1.Dependencies
	StaticFS fs.FS
	Logger   *log.Logger
}

func NewHandler(config Config) http.Handler {
	apiHandler := v1.NewHandler(config.API)
	staticFS := config.StaticFS
	if staticFS == nil {
		staticFS = webassets.DefaultFS()
	}
	static := http.FileServer(http.FS(staticFS))
	logger := config.Logger
	if logger == nil {
		logger = log.Default()
	}

	return withRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			apiHandler.ServeHTTP(w, r)
			return
		}
		serveFrontend(w, r, staticFS, static)
	}), logger)
}

func isAPIPath(value string) bool {
	return value == "/api/v1" || strings.HasPrefix(value, "/api/v1/")
}

func serveFrontend(w http.ResponseWriter, r *http.Request, assets fs.FS, server http.Handler) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writePlainError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	requested := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if requested == "" || requested == "." {
		requested = "index.html"
	}
	if _, err := fs.Stat(assets, requested); err == nil {
		server.ServeHTTP(w, r)
		return
	}
	// Extensionless routes are handled by the SPA entry point. Missing asset
	// paths with an extension remain normal 404s.
	if path.Ext(requested) == "" {
		fallback := r.Clone(r.Context())
		// FileServer intentionally redirects /index.html to ./; use the root
		// directory so its normal index.html handling serves the SPA entrypoint.
		fallback.URL.Path = "/"
		fallback.URL.RawPath = ""
		server.ServeHTTP(w, fallback)
		return
	}
	writePlainError(w, http.StatusNotFound, "resource not found")
}

func writePlainError(w http.ResponseWriter, status int, message string) {
	http.Error(w, message, status)
}

func withRecovery(next http.Handler, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Printf("http request recovered from panic: %s", r.Method)
				if isAPIPath(r.URL.Path) {
					// The API handlers write JSON themselves. This fallback is only
					// reached before a response has been committed in normal use.
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
				}
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type Server struct {
	httpServer *http.Server
}

func New(addr string, handler http.Handler, readTimeout, writeTimeout, idleTimeout time.Duration) *Server {
	return &Server{httpServer: &http.Server{
		Addr: addr, Handler: handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}}
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Serve(listener net.Listener) error {
	return s.httpServer.Serve(listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Addr() string {
	if s == nil || s.httpServer == nil {
		return ""
	}
	return s.httpServer.Addr
}
