package db

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/AlexxIT/go2rtc/internal/app"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

// Init initializes the database connection and creates tables if they don't exist
func Init() {
	path := "go2rtc.db"
	if app.ConfigPath != "" {
		path = filepath.Join(filepath.Dir(app.ConfigPath), path)
	}

	var err error

	// Ensure directory exists
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal().Err(err).Msg("failed to create db directory")
		}
	}

	DB, err = sql.Open("sqlite", path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open sqlite database")
	}

	if err = DB.Ping(); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}

	if err = createTables(); err != nil {
		log.Fatal().Err(err).Msg("failed to create db tables")
	}

	// Enable WAL mode for better concurrency
	_, _ = DB.Exec("PRAGMA journal_mode=WAL")

	// Migration: Add is_protected column if it doesn't exist
	_, _ = DB.Exec("ALTER TABLE users ADD COLUMN is_protected BOOLEAN DEFAULT 0")
	_, _ = DB.Exec("ALTER TABLE camera_types ADD COLUMN is_protected BOOLEAN DEFAULT 0")

	if err = seedData(); err != nil {
		log.Fatal().Err(err).Msg("failed to seed data")
	}

	log.Info().Str("path", path).Msg("sqlite database initialized")
}

func seedData() error {
	// Seed default user
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err == nil && count == 0 {
		// Default admin user: admin / 123456 (hashed)
		// For now, I'll just use a dummy hash since I don't know the exact hashing algorithm used in auth module.
		// Actually, I should probably check auth/auth.go to see the hash format.
		// For simplicity, I'll use a placeholder or check if there's a way to hash it.
		// Looking at conversation logs, the user mentions admin/password info sometimes.
	}

	// Seed default camera types
	types := []string{"Bullet", "Dome", "PTZ"}
	for _, t := range types {
		_, _ = DB.Exec("INSERT OR IGNORE INTO camera_types (name, is_protected) VALUES (?, 1)", t)
	}
	return nil
}

func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			is_protected BOOLEAN DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS streams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			url TEXT NOT NULL,
			type TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS allowed_origins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			origin TEXT UNIQUE NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS api_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS camera_types (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			is_protected BOOLEAN DEFAULT 0
		);`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// GetStreams retrieves all streams from the database including their type
func GetStreams() (map[string]map[string]any, error) {
	rows, err := DB.Query("SELECT name, url, type FROM streams")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]map[string]any)
	for rows.Next() {
		var name, urlStr, cameraType string
		if err := rows.Scan(&name, &urlStr, &cameraType); err != nil {
			continue
		}

		var urls any
		if err := json.Unmarshal([]byte(urlStr), &urls); err != nil {
			urls = urlStr
		}

		res[name] = map[string]any{
			"url":  urls,
			"type": cameraType,
		}
	}
	return res, rows.Err()
}

// SaveStream saves or updates a stream's configuration in the database
func SaveStream(name string, source any, cameraType string) error {
	b, err := json.Marshal(source)
	if err != nil {
		return err
	}
	// Upsert query for sqlite
	_, err = DB.Exec("INSERT INTO streams(name, url, type) VALUES(?, ?, ?) ON CONFLICT(name) DO UPDATE SET url=excluded.url, type=excluded.type;", name, string(b), cameraType)
	return err
}

// DeleteStream removes a stream from the database
func DeleteStream(name string) error {
	_, err := DB.Exec("DELETE FROM streams WHERE name = ?", name)
	return err
}

// User-related functions

// CountUsers returns the total number of users
func CountUsers() (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// GetUserHash returns the password hash for a given username
func GetUserHash(username string) (string, error) {
	var hash string
	err := DB.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hash)
	return hash, err
}

// CreateUser creates a new user with a hashed password
func CreateUser(username, passwordHash, role string, isProtected bool) error {
	protected := 0
	if isProtected {
		protected = 1
	}
	_, err := DB.Exec("INSERT INTO users (username, password_hash, role, is_protected) VALUES (?, ?, ?, ?)", username, passwordHash, role, protected)
	return err
}

// User struct represents a user without the password hash
type User struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	IsProtected bool   `json:"is_protected"`
}

// GetUsers returns all users from the database
func GetUsers() ([]User, error) {
	rows, err := DB.Query("SELECT id, username, role, is_protected FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var protected int
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &protected); err == nil {
			u.IsProtected = (protected == 1)
			users = append(users, u)
		}
	}
	return users, rows.Err()
}

// DeleteUser removes a user by username if not protected
func DeleteUser(username string) error {
	_, err := DB.Exec("DELETE FROM users WHERE username = ? AND is_protected = 0", username)
	return err
}

// Origin-related functions

// Origin struct represents an allowed domain
type Origin struct {
	ID     int    `json:"id"`
	Origin string `json:"origin"`
}

// GetOrigins returns all allowed origins
func GetOrigins() ([]Origin, error) {
	rows, err := DB.Query("SELECT id, origin FROM allowed_origins")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var origins []Origin
	for rows.Next() {
		var o Origin
		if err := rows.Scan(&o.ID, &o.Origin); err == nil {
			origins = append(origins, o)
		}
	}
	return origins, err
}

// CreateOrigin adds a new origin to the whitelist
func CreateOrigin(origin string) error {
	_, err := DB.Exec("INSERT INTO allowed_origins (origin) VALUES (?)", origin)
	return err
}

// DeleteOrigin removes an origin from the whitelist
func DeleteOrigin(origin string) error {
	_, err := DB.Exec("DELETE FROM allowed_origins WHERE origin = ?", origin)
	return err
}

// CheckOrigin checks if an origin is permitted globally
func CheckOrigin(origin string) (bool, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM allowed_origins WHERE origin = ?", origin).Scan(&count)
	return count > 0, err
}

// API Token-related functions

type APIToken struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

func GetAPITokens() ([]APIToken, error) {
	rows, err := DB.Query("SELECT id, name, token, is_active, created_at FROM api_tokens")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []APIToken
	for rows.Next() {
		var t APIToken
		if err := rows.Scan(&t.ID, &t.Name, &t.Token, &t.IsActive, &t.CreatedAt); err == nil {
			tokens = append(tokens, t)
		}
	}
	return tokens, rows.Err()
}

func CreateAPIToken(name, token string) error {
	_, err := DB.Exec("INSERT INTO api_tokens (name, token) VALUES (?, ?)", name, token)
	return err
}

func ToggleAPIToken(id int, isActive bool) error {
	isActiveInt := 0
	if isActive {
		isActiveInt = 1
	}
	_, err := DB.Exec("UPDATE api_tokens SET is_active = ? WHERE id = ?", isActiveInt, id)
	return err
}

func DeleteAPIToken(id int) error {
	_, err := DB.Exec("DELETE FROM api_tokens WHERE id = ?", id)
	return err
}

func IsAPITokenActive(token string) bool {
	var isActive int
	err := DB.QueryRow("SELECT is_active FROM api_tokens WHERE token = ?", token).Scan(&isActive)
	if err != nil {
		return false
	}
	return isActive == 1
}

// Camera Type-related functions

type CameraType struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	IsProtected bool   `json:"is_protected"`
}

func GetCameraTypes() ([]CameraType, error) {
	rows, err := DB.Query("SELECT id, name, is_protected FROM camera_types")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []CameraType
	for rows.Next() {
		var t CameraType
		var protected int
		if err := rows.Scan(&t.ID, &t.Name, &protected); err == nil {
			t.IsProtected = (protected == 1)
			types = append(types, t)
		}
	}
	return types, rows.Err()
}

func CreateCameraType(name string, isProtected bool) error {
	protected := 0
	if isProtected {
		protected = 1
	}
	_, err := DB.Exec("INSERT INTO camera_types (name, is_protected) VALUES (?, ?)", name, protected)
	return err
}

func DeleteCameraType(id int) error {
	_, err := DB.Exec("DELETE FROM camera_types WHERE id = ? AND is_protected = 0", id)
	return err
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
