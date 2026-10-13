package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type loginAttempt struct {
	count int
	reset time.Time
}
type app struct {
	db        *sql.DB
	location  *time.Location
	publicURL string
	secure    bool
	webDir    string
	mu        sync.Mutex
	attempts  map[string]loginAttempt
}

func newApp(path string, location *time.Location, publicURL, webDir string) (*app, error) {
	publicURL = strings.TrimRight(publicURL, "/")
	if publicURL != "" {
		u, err := url.Parse(publicURL)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return nil, fmt.Errorf("PUBLIC_URL must be an http(s) origin without a path")
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;
CREATE TABLE IF NOT EXISTS users (singleton INTEGER PRIMARY KEY CHECK(singleton=1), username TEXT NOT NULL, password_hash TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS sessions (hash TEXT PRIMARY KEY, expires_ms INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS tokens (key TEXT PRIMARY KEY, hash TEXT NOT NULL UNIQUE, mask TEXT NOT NULL, note TEXT NOT NULL, created_ms INTEGER NOT NULL, last_upload_ms INTEGER);
CREATE TRIGGER IF NOT EXISTS token_limit BEFORE INSERT ON tokens WHEN (SELECT count(*) FROM tokens)>=16 BEGIN SELECT RAISE(ABORT,'token limit'); END;
CREATE TABLE IF NOT EXISTS points (reporting_id TEXT NOT NULL, device_hash TEXT NOT NULL, session_id TEXT NOT NULL, timestamp_ms INTEGER NOT NULL, latitude REAL NOT NULL, longitude REAL NOT NULL, accuracy_m REAL, speed_mps REAL, bearing_deg REAL, altitude_m REAL, UNIQUE(reporting_id,device_hash,session_id,timestamp_ms));
CREATE INDEX IF NOT EXISTS points_time ON points(timestamp_ms);`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &app{db: db, location: location, publicURL: publicURL, secure: strings.HasPrefix(publicURL, "https://"), webDir: webDir, attempts: make(map[string]loginAttempt)}, nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func randomKey() string {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value)
}
