package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AlexxIT/go2rtc/internal/db"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

var (
	// Simple in-memory session store (token -> username)
	// For a production app, this could be backed by SQLite or Redis
	sessions   = make(map[string]sessionInfo)
	sessionsMu sync.RWMutex
)

type sessionInfo struct {
	Username  string
	ExpiresAt time.Time
}

// Init ensures that there is at least one admin user in the database
func Init() {
	count, err := db.CountUsers()
	if err != nil {
		log.Error().Err(err).Msg("failed to count users")
		return
	}

	// Create default admin user if database is empty
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("P@ssw0rd"), bcrypt.DefaultCost)
		if err := db.CreateUser("admin", string(hash), "admin", true); err != nil {
			log.Error().Err(err).Msg("failed to create default admin user")
		} else {
			log.Info().Msg("created default admin/P@ssw0rd user")
		}
	}

	// Start session cleanup routine
	go cleanupSessions()
}

// CheckPassword verifies a plaintext password against the hash in the DB
func CheckPassword(username, password string) bool {
	hash, err := db.GetUserHash(username)
	if err != nil {
		return false
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken creates a new secure session token
func GenerateToken(username string) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)

	sessionsMu.Lock()
	sessions[token] = sessionInfo{
		Username:  username,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 1 day expiration
	}
	sessionsMu.Unlock()

	return token
}

// GenerateRandomToken creates a random URL-safe base64 string
func GenerateRandomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// ValidateToken checks if a token is valid and returns the username
func ValidateToken(token string) (string, bool) {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	info, exists := sessions[token]
	if !exists {
		return "", false
	}

	if time.Now().After(info.ExpiresAt) {
		return "", false
	}

	return info.Username, true
}

func cleanupSessions() {
	for {
		time.Sleep(1 * time.Hour)
		now := time.Now()

		sessionsMu.Lock()
		for token, info := range sessions {
			if now.After(info.ExpiresAt) {
				delete(sessions, token)
			}
		}
		sessionsMu.Unlock()
	}
}

// ExtractToken attempts to find an auth token in Cookies or Bearer header
func ExtractToken(r *http.Request) string {
	// 1. Check Cookie
	if cookie, err := r.Cookie("go2rtc_session"); err == nil {
		return cookie.Value
	}

	// 2. Check Authorization Bearer
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// 3. Check Query Param (for video streams in browser)
	if token := r.URL.Query().Get("token"); token != "" {
		// Clean the token from query so it doesn't propagate to upstream
		q := r.URL.Query()
		q.Del("token")
		r.URL.RawQuery = q.Encode()
		return token
	}

	return ""
}
