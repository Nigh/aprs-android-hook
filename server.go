package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	code := errorCodes[message]
	if code == "" {
		code = "error.requestFailed"
	}
	respond(w, status, map[string]string{"error": message, "code": code})
}
func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		fail(w, 415, "请使用 application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	d := json.NewDecoder(r.Body)
	if err := d.Decode(value); err != nil {
		fail(w, 400, "JSON 格式无效或请求过大")
		return false
	}
	if err := d.Decode(new(any)); err != io.EOF {
		fail(w, 400, "请求只能包含一个 JSON 对象")
		return false
	}
	return true
}

func (a *app) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if a.db.PingContext(r.Context()) != nil {
			fail(w, 503, "数据库不可用")
			return
		}
		respond(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/auth", a.authState)
	mux.HandleFunc("POST /api/register", a.authWrite(a.register))
	mux.HandleFunc("POST /api/login", a.authWrite(a.login))
	mux.HandleFunc("POST /api/logout", a.private(a.logout))
	mux.HandleFunc("POST /api/webhook", a.webhook)
	mux.HandleFunc("GET /api/tokens", a.private(a.listTokens))
	mux.HandleFunc("POST /api/tokens", a.private(a.addToken))
	mux.HandleFunc("PATCH /api/tokens/{key}", a.private(a.editToken))
	mux.HandleFunc("DELETE /api/tokens/{key}", a.private(a.deleteToken))
	mux.HandleFunc("GET /api/tracks", a.private(a.tracks))
	mux.HandleFunc("GET /api/tracks/{key}", a.private(a.trackDetail))
	mux.HandleFunc("GET /api/sources", a.private(a.sources))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "接口不存在") })
	mux.Handle("/", http.FileServer(http.Dir(a.webDir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		mux.ServeHTTP(w, r)
	})
}

func (a *app) sameOrigin(w http.ResponseWriter, r *http.Request) bool {
	expected := a.publicURL
	if expected == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		expected = scheme + "://" + r.Host
	}
	if r.Header.Get("Origin") != expected {
		fail(w, 403, "请求来源不匹配，请检查 PUBLIC_URL")
		return false
	}
	return true
}
func (a *app) authenticated(r *http.Request) bool {
	cookie, err := r.Cookie("aprs_session")
	if err != nil {
		return false
	}
	var expires int64
	err = a.db.QueryRowContext(r.Context(), "SELECT expires_ms FROM sessions WHERE hash=?", digest(cookie.Value)).Scan(&expires)
	return err == nil && expires > time.Now().UnixMilli()
}
func (a *app) private(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authenticated(r) {
			fail(w, 401, "请先登录")
			return
		}
		if r.Method != "GET" && !a.sameOrigin(w, r) {
			return
		}
		next(w, r)
	}
}
func (a *app) authWrite(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.sameOrigin(w, r) {
			return
		}
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		a.mu.Lock()
		now := time.Now()
		for key, value := range a.attempts {
			if !value.reset.After(now) {
				delete(a.attempts, key)
			}
		}
		attempt := a.attempts[ip]
		if attempt.count >= 10 || (attempt.count == 0 && len(a.attempts) >= 4096) {
			a.mu.Unlock()
			w.Header().Set("Retry-After", "900")
			fail(w, 429, "尝试过于频繁，请15分钟后重试")
			return
		}
		if attempt.count == 0 {
			attempt.reset = now.Add(15 * time.Minute)
		}
		attempt.count++
		a.attempts[ip] = attempt
		a.mu.Unlock()
		next(w, r)
	}
}
func (a *app) authState(w http.ResponseWriter, r *http.Request) {
	var count int
	if err := a.db.QueryRowContext(r.Context(), "SELECT count(*) FROM users").Scan(&count); err != nil {
		fail(w, 500, "无法读取账户状态")
		return
	}
	now := time.Now().In(a.location)
	respond(w, 200, map[string]any{"registered": count != 0, "authenticated": a.authenticated(r), "timezone": a.location.String(), "today": now.Format("2006-01-02")})
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *app) register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !decode(w, r, &c) {
		return
	}
	c.Username = strings.TrimSpace(c.Username)
	if utf8.RuneCountInString(c.Username) < 1 || utf8.RuneCountInString(c.Username) > 64 || len(c.Password) < 12 || len(c.Password) > 72 {
		fail(w, 400, "用户名为1–64字，密码为12–72字节")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, "无法保存密码")
		return
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w, 500, "无法创建账户")
		return
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRow("SELECT count(*) FROM users").Scan(&count); err != nil {
		fail(w, 500, "无法创建账户")
		return
	}
	if count != 0 {
		fail(w, 409, "唯一账户已注册，请登录")
		return
	}
	if _, err = tx.Exec("INSERT INTO users VALUES(1,?,?)", c.Username, string(hash)); err != nil {
		fail(w, 500, "无法创建账户")
		return
	}
	key := randomKey()
	expiry := time.Now().Add(7 * 24 * time.Hour)
	if _, err = tx.Exec("INSERT INTO sessions VALUES(?,?)", digest(key), expiry.UnixMilli()); err != nil {
		fail(w, 500, "无法创建会话")
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w, 500, "无法创建账户")
		return
	}
	a.setSession(w, key, expiry)
	respond(w, 201, map[string]bool{"ok": true})
}
func (a *app) setSession(w http.ResponseWriter, key string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{Name: "aprs_session", Value: key, Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteStrictMode, Expires: expiry, MaxAge: 7 * 24 * 3600})
}
func (a *app) login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !decode(w, r, &c) {
		return
	}
	var username, hash string
	err := a.db.QueryRowContext(r.Context(), "SELECT username,password_hash FROM users WHERE singleton=1").Scan(&username, &hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		fail(w, 500, "无法读取账户")
		return
	}
	if err != nil {
		fail(w, 401, "用户名或密码错误")
		return
	}
	valid := bcrypt.CompareHashAndPassword([]byte(hash), []byte(c.Password)) == nil
	if !valid || strings.TrimSpace(c.Username) != username {
		fail(w, 401, "用户名或密码错误")
		return
	}
	key := randomKey()
	expiry := time.Now().Add(7 * 24 * time.Hour)
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w, 500, "无法创建会话")
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec("DELETE FROM sessions WHERE expires_ms<=?", time.Now().UnixMilli()); err == nil {
		_, err = tx.Exec("INSERT INTO sessions VALUES(?,?)", digest(key), expiry.UnixMilli())
	}
	if err != nil {
		fail(w, 500, "无法创建会话")
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w, 500, "无法创建会话")
		return
	}
	a.setSession(w, key, expiry)
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("aprs_session")
	if _, err := a.db.ExecContext(r.Context(), "DELETE FROM sessions WHERE hash=?", digest(cookie.Value)); err != nil {
		fail(w, 500, "退出失败，请重试")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "aprs_session", Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	respond(w, 200, map[string]bool{"ok": true})
}

var tokenPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var devicePattern = regexp.MustCompile(`^[0-9a-f]{16}$`)
var sessionPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type tokenInput struct {
	Token string `json:"token"`
	Note  string `json:"note"`
}

func (a *app) listTokens(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(), "SELECT key,mask,note,created_ms,last_upload_ms FROM tokens ORDER BY created_ms,key")
	if err != nil {
		fail(w, 500, "无法读取 token")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var key, mask, note string
		var created int64
		var last *int64
		if err := rows.Scan(&key, &mask, &note, &created, &last); err != nil {
			fail(w, 500, "无法读取 token")
			return
		}
		items = append(items, map[string]any{"key": key, "mask": mask, "note": note, "created_ms": created, "last_upload_ms": last})
	}
	if rows.Err() != nil {
		fail(w, 500, "无法读取 token")
		return
	}
	respond(w, 200, items)
}
func (a *app) addToken(w http.ResponseWriter, r *http.Request) {
	var input tokenInput
	if !decode(w, r, &input) {
		return
	}
	input.Token = strings.TrimSpace(input.Token)
	input.Note = strings.TrimSpace(input.Note)
	if !tokenPattern.MatchString(input.Token) || utf8.RuneCountInString(input.Note) > 128 {
		fail(w, 400, "token 必须为64位小写十六进制字符，备注最多128字")
		return
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w, 500, "无法保存 token")
		return
	}
	defer tx.Rollback()
	var count int
	var duplicate int
	if err = tx.QueryRow("SELECT count(*),count(CASE WHEN hash=? THEN 1 END) FROM tokens", digest(input.Token)).Scan(&count, &duplicate); err != nil {
		fail(w, 500, "无法保存 token")
		return
	}
	if duplicate != 0 {
		fail(w, 409, "该 token 已登记")
		return
	}
	if count >= 16 {
		fail(w, 409, "最多登记16个 token")
		return
	}
	key := randomKey()
	if _, err = tx.Exec("INSERT INTO tokens(key,hash,mask,note,created_ms) VALUES(?,?,?,?,?)", key, digest(input.Token), input.Token[:6]+"…"+input.Token[58:], input.Note, time.Now().UnixMilli()); err != nil {
		fail(w, 500, "无法保存 token")
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w, 500, "无法保存 token")
		return
	}
	respond(w, 201, map[string]string{"key": key})
}
func (a *app) editToken(w http.ResponseWriter, r *http.Request) {
	var input tokenInput
	if !decode(w, r, &input) {
		return
	}
	input.Note = strings.TrimSpace(input.Note)
	if utf8.RuneCountInString(input.Note) > 128 {
		fail(w, 400, "备注最多128字")
		return
	}
	result, err := a.db.ExecContext(r.Context(), "UPDATE tokens SET note=? WHERE key=?", input.Note, r.PathValue("key"))
	mutationResult(w, result, err)
}
func (a *app) deleteToken(w http.ResponseWriter, r *http.Request) {
	result, err := a.db.ExecContext(r.Context(), "DELETE FROM tokens WHERE key=?", r.PathValue("key"))
	mutationResult(w, result, err)
}
func mutationResult(w http.ResponseWriter, result sql.Result, err error) {
	if err != nil {
		fail(w, 500, "保存失败，请重试")
		return
	}
	n, err := result.RowsAffected()
	if err != nil {
		fail(w, 500, "保存失败，请重试")
		return
	}
	if n == 0 {
		fail(w, 404, "token 不存在，请刷新")
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}

func finite(value *float64) bool {
	return value == nil || (!math.IsNaN(*value) && !math.IsInf(*value, 0))
}
func (a *app) webhook(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") || !tokenPattern.MatchString(strings.TrimPrefix(header, "Bearer ")) {
		fail(w, 401, "上传 token 无效")
		return
	}
	var p point
	if !decode(w, r, &p) {
		return
	}
	if p.Type != "location" && p.Type != "test" {
		fail(w, 400, "上传类型无效")
		return
	}
	if !sessionPattern.MatchString(p.Session) {
		fail(w, 400, "session_id 必须为标准小写 UUID v4")
		return
	}
	if p.Type != "test" && (strings.TrimSpace(p.ID) == "" || utf8.RuneCountInString(p.ID) > 128 || !devicePattern.MatchString(p.Device) || p.Time <= 0 || p.Time > time.Now().Add(5*time.Minute).UnixMilli() || p.Lat == nil || p.Lon == nil || !finite(p.Lat) || !finite(p.Lon) || *p.Lat < -90 || *p.Lat > 90 || *p.Lon < -180 || *p.Lon > 180 || !finite(p.Accuracy) || !finite(p.Speed) || !finite(p.Bearing) || !finite(p.Altitude) || (p.Accuracy != nil && *p.Accuracy < 0) || (p.Speed != nil && *p.Speed < 0) || (p.Bearing != nil && (*p.Bearing < 0 || *p.Bearing >= 360))) {
		fail(w, 400, "定位字段缺失或超出有效范围")
		return
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w, 503, "数据库暂不可用")
		return
	}
	defer tx.Rollback()
	var key string
	err = tx.QueryRow("SELECT key FROM tokens WHERE hash=?", digest(strings.TrimPrefix(header, "Bearer "))).Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 401, "上传 token 未登记或已删除")
		return
	}
	if err != nil {
		fail(w, 503, "数据库暂不可用")
		return
	}
	if p.Type == "test" {
		respond(w, 200, map[string]any{"ok": true, "type": "test"})
		return
	}
	_, err = tx.Exec(`INSERT INTO points(reporting_id,device_hash,session_id,timestamp_ms,latitude,longitude,accuracy_m,speed_mps,bearing_deg,altitude_m) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(reporting_id,device_hash,session_id,timestamp_ms) DO NOTHING`, p.ID, p.Device, p.Session, p.Time, p.Lat, p.Lon, p.Accuracy, p.Speed, p.Bearing, p.Altitude)
	if err == nil {
		_, err = tx.Exec("UPDATE tokens SET last_upload_ms=? WHERE key=?", time.Now().UnixMilli(), key)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("webhook storage error: %v", err)
		fail(w, 503, "保存失败，请重试")
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}
