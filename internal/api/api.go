package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AlexxIT/go2rtc/internal/app"
	"github.com/AlexxIT/go2rtc/internal/auth"
	"github.com/AlexxIT/go2rtc/internal/db"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

func Init() {
	var cfg struct {
		Mod struct {
			Listen     string `yaml:"listen"`
			Username   string `yaml:"username"`
			Password   string `yaml:"password"`
			LocalAuth  bool   `yaml:"local_auth"`
			BasePath   string `yaml:"base_path"`
			StaticDir  string `yaml:"static_dir"`
			Origin     string `yaml:"origin"`
			TLSListen  string `yaml:"tls_listen"`
			TLSCert    string `yaml:"tls_cert"`
			TLSKey     string `yaml:"tls_key"`
			UnixListen string `yaml:"unix_listen"`

			AllowPaths    []string `yaml:"allow_paths"`
			StreamOrigins []string `yaml:"stream_origins"`
		} `yaml:"api"`
	}

	// default config
	cfg.Mod.Listen = ":1984"

	// load config from YAML
	app.LoadConfig(&cfg)

	if cfg.Mod.Listen == "" && cfg.Mod.UnixListen == "" && cfg.Mod.TLSListen == "" {
		return
	}

	allowPaths = cfg.Mod.AllowPaths
	basePath = cfg.Mod.BasePath
	streamOrigins = cfg.Mod.StreamOrigins
	log = app.GetLogger("api")

	initStatic(cfg.Mod.StaticDir)

	HandleFunc("api", apiHandler)
	HandleFunc("api/login", apiLogin)
	HandleFunc("api/logout", apiLogout)
	HandleFunc("api/users", apiUsers)
	HandleFunc("api/origins", apiOrigins)
	HandleFunc("api/tokens", apiTokens)
	HandleFunc("api/types", apiTypes)
	HandleFunc("api/config", configHandler)
	HandleFunc("api/exit", exitHandler)
	HandleFunc("api/restart", restartHandler)
	HandleFunc("api/log", logHandler)

	Handler = http.DefaultServeMux // 4th

	if cfg.Mod.Origin == "*" {
		Handler = middlewareCORS(Handler) // 3rd
	}

	// Always apply Auth middleware since we use the database now
	Handler = middlewareAuth(cfg.Mod.LocalAuth, Handler) // 2nd

	if log.Trace().Enabled() {
		Handler = middlewareLog(Handler) // 1st
	}

	if cfg.Mod.Listen != "" {
		_, port, _ := net.SplitHostPort(cfg.Mod.Listen)
		Port, _ = strconv.Atoi(port)
		go listen("tcp", cfg.Mod.Listen)
	}

	if cfg.Mod.UnixListen != "" {
		_ = syscall.Unlink(cfg.Mod.UnixListen)
		go listen("unix", cfg.Mod.UnixListen)
	}

	// Initialize the HTTPS server
	if cfg.Mod.TLSListen != "" && cfg.Mod.TLSCert != "" && cfg.Mod.TLSKey != "" {
		go tlsListen("tcp", cfg.Mod.TLSListen, cfg.Mod.TLSCert, cfg.Mod.TLSKey)
	}
}

func listen(network, address string) {
	ln, err := net.Listen(network, address)
	if err != nil {
		log.Error().Err(err).Msg("[api] listen")
		return
	}

	log.Info().Str("addr", address).Msg("[api] listen")

	server := http.Server{
		Handler:           Handler,
		ReadHeaderTimeout: 5 * time.Second, // Example: Set to 5 seconds
	}
	if err = server.Serve(ln); err != nil {
		log.Fatal().Err(err).Msg("[api] serve")
	}
}

func tlsListen(network, address, certFile, keyFile string) {
	var cert tls.Certificate
	var err error
	if strings.IndexByte(certFile, '\n') < 0 && strings.IndexByte(keyFile, '\n') < 0 {
		// check if file path
		cert, err = tls.LoadX509KeyPair(certFile, keyFile)
	} else {
		// if text file content
		cert, err = tls.X509KeyPair([]byte(certFile), []byte(keyFile))
	}
	if err != nil {
		log.Error().Err(err).Caller().Send()
		return
	}

	ln, err := net.Listen(network, address)
	if err != nil {
		log.Error().Err(err).Msg("[api] tls listen")
		return
	}

	log.Info().Str("addr", address).Msg("[api] tls listen")

	server := &http.Server{
		Handler:           Handler,
		TLSConfig:         &tls.Config{Certificates: []tls.Certificate{cert}},
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err = server.ServeTLS(ln, "", ""); err != nil {
		log.Fatal().Err(err).Msg("[api] tls serve")
	}
}

var Port int

const (
	MimeJSON = "application/json"
	MimeText = "text/plain"
)

var Handler http.Handler

// HandleFunc handle pattern with relative path:
// - "api/streams" => "{basepath}/api/streams"
// - "/streams"    => "/streams"
func HandleFunc(pattern string, handler http.HandlerFunc) {
	if len(pattern) == 0 || pattern[0] != '/' {
		pattern = basePath + "/" + pattern
	}
	if allowPaths != nil && !slices.Contains(allowPaths, pattern) {
		log.Trace().Str("path", pattern).Msg("[api] ignore path not in allow_paths")
		return
	}
	log.Trace().Str("path", pattern).Msg("[api] register path")
	http.HandleFunc(pattern, handler)
}

// ResponseJSON important always add Content-Type
// so go won't need to call http.DetectContentType
func ResponseJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", MimeJSON)
	_ = json.NewEncoder(w).Encode(v)
}

func ResponsePrettyJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", MimeJSON)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func Response(w http.ResponseWriter, body any, contentType string) {
	w.Header().Set("Content-Type", contentType)

	switch v := body.(type) {
	case []byte:
		_, _ = w.Write(v)
	case string:
		_, _ = w.Write([]byte(v))
	default:
		_, _ = fmt.Fprint(w, body)
	}
}

const StreamNotFound = "stream not found"

var allowPaths []string
var basePath string
var streamOrigins []string
var log zerolog.Logger

// AllowedOrigin returns the value to set for the Access-Control-Allow-Origin header
// based on the configured stream_origins whitelist and the SQLite origins table.
func AllowedOrigin(r *http.Request) string {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Same-origin requests (no Origin header) are always allowed
		return ""
	}

	// 1. Check global YAML origins
	for _, o := range streamOrigins {
		if o == "*" || o == origin {
			return o
		}
	}

	// 2. Check global origins from DB
	if allowed, _ := db.CheckOrigin(origin); allowed {
		return origin
	}

	return ""
}

func middlewareLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Trace().Msgf("[api] %s %s %s", r.Method, r.URL, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func isLoopback(remoteAddr string) bool {
	return strings.HasPrefix(remoteAddr, "127.") || strings.HasPrefix(remoteAddr, "[::1]") || remoteAddr == "@"
}

func extractOrigin(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func middlewareAuth(localAuth bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Allow static routes, assets, and internal login paths to bypass authentication
		exempt := path == "/api/login" || path == "/api/logout" || path == "/login.html" || path == "/favicon.ico" || path == "/manifest.json" || strings.HasSuffix(path, ".js") || strings.HasPrefix(path, "/icons/")
		publicStream := path == "/stream.html" || path == "/api/ws" || path == "/api/streams" || strings.HasPrefix(path, "/api/stream.")

		if !exempt {
			// 1. Check Session Token or API Token
			token := auth.ExtractToken(r)
			if token != "" {
				if _, ok := auth.ValidateToken(token); ok {
					next.ServeHTTP(w, r)
					return
				}
				if db.IsAPITokenActive(token) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// 2. Check Basic Auth (for legacy or direct API integrations)
			user, pass, ok := r.BasicAuth()
			if ok && auth.CheckPassword(user, pass) {
				next.ServeHTTP(w, r)
				return
			}

			// If asking for a public stream but not logged in, enforce Origin checks
			if publicStream && r.Method == "GET" {
				origin := r.Header.Get("Origin")
				if origin == "" {
					origin = extractOrigin(r.Header.Get("Referer"))
				}

				ownHttp := extractOrigin("http://" + r.Host)
				ownHttps := extractOrigin("https://" + r.Host)

				// Allow if accessed directly, internal API call from same host, or if origin is in DB
				if origin == "" || origin == ownHttp || origin == ownHttps {
					next.ServeHTTP(w, r)
					return
				}

				if allowed, _ := db.CheckOrigin(origin); allowed {
					next.ServeHTTP(w, r)
					return
				}

				for _, o := range streamOrigins {
					if o == "*" || o == origin {
						next.ServeHTTP(w, r)
						return
					}
				}

				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}

			// Unauthorized
			if strings.Contains(r.Header.Get("Accept"), "text/html") && r.Method == "GET" {
				http.Redirect(w, r, "login.html", http.StatusFound)
				return
			}

			w.Header().Set("Www-Authenticate", `Basic realm="go2rtc"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		next.ServeHTTP(w, r)
	})
}

var mu sync.Mutex

func apiLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if auth.CheckPassword(username, password) {
		token := auth.GenerateToken(username)

		http.SetCookie(w, &http.Cookie{
			Name:     "go2rtc_session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   86400, // 1 day
			SameSite: http.SameSiteLaxMode,
		})

		ResponseJSON(w, map[string]string{"status": "ok", "token": token})
		return
	}

	http.Error(w, "invalid username or password", http.StatusUnauthorized)
}

func apiLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "go2rtc_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	ResponseJSON(w, map[string]string{"status": "ok"})
}

func apiUsers(w http.ResponseWriter, r *http.Request) {
	// Require full authentication for user management
	// We check if the basic auth or cookie auth is an admin?
	// For simplicity, any authenticated user can view/manage, OR we can restrict to role='admin'
	// Since we haven't implemented roles fully into the session, we just trust the middleware here.

	switch r.Method {
	case "GET":
		users, err := db.GetUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, users)

	case "POST":
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username == "" || password == "" {
			http.Error(w, "username or password empty", http.StatusBadRequest)
			return
		}

		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err := db.CreateUser(username, string(hash), "admin"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, map[string]string{"status": "ok"})

	case "DELETE":
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "missing username", http.StatusBadRequest)
			return
		}
		if username == "admin" {
			http.Error(w, "cannot delete default admin", http.StatusBadRequest)
			return
		}

		if err := db.DeleteUser(username); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, map[string]string{"status": "ok"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiOrigins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		origins, err := db.GetOrigins()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, origins)

	case "POST":
		origin := r.FormValue("origin")
		if origin == "" {
			http.Error(w, "origin empty", http.StatusBadRequest)
			return
		}

		if err := db.CreateOrigin(origin); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, map[string]string{"status": "ok"})

	case "DELETE":
		origin := r.URL.Query().Get("origin")
		if origin == "" {
			http.Error(w, "missing origin", http.StatusBadRequest)
			return
		}

		if err := db.DeleteOrigin(origin); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, map[string]string{"status": "ok"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		tokens, err := db.GetAPITokens()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, tokens)

	case "POST":
		name := r.FormValue("name")
		if name == "" {
			http.Error(w, "name empty", http.StatusBadRequest)
			return
		}

		token := auth.GenerateRandomToken()
		if err := db.CreateAPIToken(name, token); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, map[string]string{"status": "ok", "token": token})

	case "PATCH":
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			idStr = r.FormValue("id")
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		isActiveStr := r.URL.Query().Get("is_active")
		if isActiveStr == "" {
			isActiveStr = r.FormValue("is_active")
		}
		isActive := isActiveStr == "true" || isActiveStr == "1"

		if err := db.ToggleAPIToken(id, isActive); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, map[string]string{"status": "ok"})

	case "DELETE":
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			idStr = r.FormValue("id")
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if err := db.DeleteAPIToken(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, map[string]string{"status": "ok"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiTypes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		types, err := db.GetCameraTypes()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, types)

	case "POST":
		name := r.FormValue("name")
		if name == "" {
			http.Error(w, "name empty", http.StatusBadRequest)
			return
		}

		if err := db.CreateCameraType(name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, map[string]string{"status": "ok"})

	case "DELETE":
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			idStr = r.FormValue("id")
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if err := db.DeleteCameraType(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ResponseJSON(w, map[string]string{"status": "ok"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	app.Info["host"] = r.Host
	mu.Unlock()

	ResponseJSON(w, app.Info)
}

func exitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	s := r.URL.Query().Get("code")
	code, err := strconv.Atoi(s)

	// https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html#tag_18_08_02
	if err != nil || code < 0 || code > 125 {
		http.Error(w, "Code must be in the range [0, 125]", http.StatusBadRequest)
		return
	}

	os.Exit(code)
}

func restartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	path, err := os.Executable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug().Msgf("[api] restart %s", path)

	go syscall.Exec(path, os.Args, os.Environ())
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Send current state of the log file immediately
		w.Header().Set("Content-Type", "application/jsonlines")
		_, _ = app.MemoryLog.WriteTo(w)
	case "DELETE":
		app.MemoryLog.Reset()
		Response(w, "OK", "text/plain")
	default:
		http.Error(w, "Method not allowed", http.StatusBadRequest)
	}
}

type Source struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Info     string `json:"info,omitempty"`
	URL      string `json:"url,omitempty"`
	Location string `json:"location,omitempty"`
}

func ResponseSources(w http.ResponseWriter, sources []*Source) {
	if len(sources) == 0 {
		http.Error(w, "no sources", http.StatusNotFound)
		return
	}

	var response = struct {
		Sources []*Source `json:"sources"`
	}{
		Sources: sources,
	}
	ResponseJSON(w, response)
}

func Error(w http.ResponseWriter, err error) {
	log.Error().Err(err).Caller(1).Send()

	http.Error(w, err.Error(), http.StatusInsufficientStorage)
}
