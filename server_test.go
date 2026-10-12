package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testApp(t *testing.T) *app {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	a, err := newApp(filepath.Join(t.TempDir(), "test.sqlite"), loc, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.db.Close() })
	return a
}
func request(a *app, method, path, body string, cookie *http.Cookie, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://example.com")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	a.handler().ServeHTTP(w, r)
	return w
}
func expect(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("HTTP %d, want %d: %s", w.Code, status, w.Body.String())
	}
}
func registerTestUser(t *testing.T, a *app) *http.Cookie {
	t.Helper()
	w := request(a, "POST", "/api/register", `{"username":"test","password":"long-password-123"}`, nil, "")
	expect(t, w, 201)
	return w.Result().Cookies()[0]
}
func addTestToken(t *testing.T, a *app, c *http.Cookie, value string) string {
	t.Helper()
	w := request(a, "POST", "/api/tokens", fmt.Sprintf(`{"token":%q,"note":"手机"}`, value), c, "")
	expect(t, w, 201)
	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result["key"]
}

func TestFirstRegistrationIsAtomic(t *testing.T) {
	a := testApp(t)
	statuses := make(chan int, 4)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- request(a, "POST", "/api/register", `{"username":"owner","password":"long-password-123"}`, nil, "").Code
		}()
	}
	wg.Wait()
	close(statuses)
	success := 0
	for status := range statuses {
		if status == 201 {
			success++
		} else if status != 409 {
			t.Fatalf("unexpected status %d", status)
		}
	}
	if success != 1 {
		t.Fatalf("created %d users", success)
	}
	var count int
	_ = a.db.QueryRow("SELECT count(*) FROM users").Scan(&count)
	if count != 1 {
		t.Fatal(count)
	}
}
func TestAuthenticationBoundaries(t *testing.T) {
	a := testApp(t)
	cookie := registerTestUser(t, a)
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge != 604800 {
		t.Fatal("unsafe cookie")
	}
	for _, path := range []string{"/api/tracks", "/api/tokens", "/api/sources"} {
		expect(t, request(a, "GET", path, "", nil, ""), 401)
	}
	w := request(a, "GET", "/api/tokens", "", cookie, "")
	expect(t, w, 200)
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("private response cacheable")
	}
	r := httptest.NewRequest("POST", "http://example.com/api/tokens", strings.NewReader(`{}`))
	r.AddCookie(cookie)
	r.Header.Set("Origin", "https://other.example")
	w = httptest.NewRecorder()
	a.handler().ServeHTTP(w, r)
	expect(t, w, 403)
	expect(t, request(a, "POST", "/api/logout", "", cookie, ""), 200)
	expect(t, request(a, "GET", "/api/tokens", "", cookie, ""), 401)
	expect(t, request(a, "POST", "/api/login", `{"username":"test","password":"wrong"}`, nil, ""), 401)
	w = request(a, "POST", "/api/login", `{"username":"test","password":"long-password-123"}`, nil, "")
	expect(t, w, 200)
	cookie = w.Result().Cookies()[0]
	_, _ = a.db.Exec("UPDATE sessions SET expires_ms=0")
	expect(t, request(a, "GET", "/api/tokens", "", cookie, ""), 401)
	for range 7 {
		request(a, "POST", "/api/login", `{}`, nil, "")
	}
	expect(t, request(a, "POST", "/api/login", `{}`, nil, ""), 429)
	a.publicURL = "https://tracks.example.com"
	a.secure = true
	r = httptest.NewRequest("POST", "http://internal/api/login", strings.NewReader(`{"username":"test","password":"long-password-123"}`))
	r.RemoteAddr = "192.0.2.50:123"
	r.Header.Set("Origin", a.publicURL)
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	a.handler().ServeHTTP(w, r)
	expect(t, w, 200)
	if !w.Result().Cookies()[0].Secure {
		t.Fatal("missing secure cookie")
	}
}
func TestTokenLimitAndRevocation(t *testing.T) {
	a := testApp(t)
	cookie := registerTestUser(t, a)
	token := strings.Repeat("a", 64)
	key := addTestToken(t, a, cookie, token)
	expect(t, request(a, "POST", "/api/tokens", fmt.Sprintf(`{"token":%q}`, token), cookie, ""), 409)
	var stored string
	_ = a.db.QueryRow("SELECT hash FROM tokens WHERE key=?", key).Scan(&stored)
	if stored == token || stored != digest(token) {
		t.Fatal("raw token stored")
	}
	for i := 1; i < 16; i++ {
		addTestToken(t, a, cookie, fmt.Sprintf("%064x", i))
	}
	expect(t, request(a, "POST", "/api/tokens", fmt.Sprintf(`{"token":%q}`, strings.Repeat("b", 64)), cookie, ""), 409)
	expect(t, request(a, "PATCH", "/api/tokens/"+key, `{"note":"日常手机"}`, cookie, ""), 200)
	body := `{"id":"设备1","device_hash":"1234567890abcdef","timestamp_ms":100000,"latitude":0,"longitude":0,"accuracy_m":null,"speed_mps":0,"bearing_deg":0,"altitude_m":-12.5}`
	expect(t, request(a, "POST", "/api/webhook", body, nil, token), 200)
	expect(t, request(a, "DELETE", "/api/tokens/"+key, "", cookie, ""), 200)
	expect(t, request(a, "POST", "/api/webhook", body, nil, token), 401)
	var count int
	_ = a.db.QueryRow("SELECT count(*) FROM points").Scan(&count)
	if count != 1 {
		t.Fatal("history deleted")
	}
}

func TestConcurrentTokenLimit(t *testing.T) {
	a := testApp(t)
	cookie := registerTestUser(t, a)
	for i := 0; i < 15; i++ {
		addTestToken(t, a, cookie, fmt.Sprintf("%064x", i))
	}
	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 15; i < 17; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			statuses <- request(a, "POST", "/api/tokens", fmt.Sprintf(`{"token":"%064x"}`, i), cookie, "").Code
		}(i)
	}
	wg.Wait()
	close(statuses)
	success := 0
	for status := range statuses {
		if status == 201 {
			success++
		} else if status != 409 {
			t.Fatalf("unexpected status %d", status)
		}
	}
	if success != 1 {
		t.Fatalf("created %d tokens at boundary", success)
	}
}

func TestHTTPSUpload(t *testing.T) {
	a := testApp(t)
	cookie := registerTestUser(t, a)
	token := strings.Repeat("d", 64)
	addTestToken(t, a, cookie, token)
	server := httptest.NewTLSServer(a.handler())
	defer server.Close()
	body := `{"id":"connection-test","device_hash":"0123456789abcdef","timestamp_ms":1767225600000,"latitude":31.2304,"longitude":121.4737,"accuracy_m":5,"speed_mps":null,"bearing_deg":null,"altitude_m":null}`
	r, err := http.NewRequest("POST", server.URL+"/api/webhook", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+token)
	resp, err := server.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("HTTPS upload: %d", resp.StatusCode)
	}
}
func TestWebhookValidationAndPersistence(t *testing.T) {
	a := testApp(t)
	cookie := registerTestUser(t, a)
	token := strings.Repeat("c", 64)
	addTestToken(t, a, cookie, token)
	valid := `{"id":"用户\"1","device_hash":"1234567890abcdef","timestamp_ms":100000,"latitude":22.5,"longitude":113.9,"accuracy_m":null,"speed_mps":null,"bearing_deg":null,"altitude_m":null}`
	expect(t, request(a, "POST", "/api/webhook", valid, nil, ""), 401)
	expect(t, request(a, "POST", "/api/webhook", valid, nil, strings.Repeat("f", 64)), 401)
	for _, body := range []string{`{}`, strings.Replace(valid, `"latitude":22.5`, `"latitude":91`, 1), strings.Replace(valid, `"longitude":113.9`, `"longitude":null`, 1), strings.Replace(valid, `"accuracy_m":null`, `"accuracy_m":-1`, 1), strings.Replace(valid, `"bearing_deg":null`, `"bearing_deg":360`, 1), strings.Replace(valid, `"timestamp_ms":100000`, `"timestamp_ms":9999999999999999`, 1), valid + valid, strings.Repeat("a", 17000)} {
		expect(t, request(a, "POST", "/api/webhook", body, nil, token), 400)
	}
	for range 2 {
		expect(t, request(a, "POST", "/api/webhook", valid, nil, token), 200)
	}
	older := strings.Replace(valid, `"timestamp_ms":100000`, `"timestamp_ms":90000`, 1)
	expect(t, request(a, "POST", "/api/webhook", older, nil, token), 200)
	var count int
	_ = a.db.QueryRow("SELECT count(*) FROM points").Scan(&count)
	if count != 2 {
		t.Fatal("not deduplicated", count)
	}
	var file string
	_ = a.db.QueryRow("SELECT file FROM pragma_database_list WHERE name='main'").Scan(&file)
	a.db.Close()
	b, err := newApp(file, a.location, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer b.db.Close()
	_ = b.db.QueryRow("SELECT count(*) FROM points").Scan(&count)
	if count != 2 {
		t.Fatal("not persisted")
	}
}
func pointAt(id, device string, ms int64, lat, lon float64) point {
	return point{ID: id, Device: device, Time: ms, Lat: &lat, Lon: &lon}
}
func TestTrackBoundariesAndDistance(t *testing.T) {
	points := []point{pointAt("a", "d", 1, 0, 0), pointAt("a", "d", 1800001, 0, 1), pointAt("a", "d", 3600002, 0, 2), pointAt("a", "e", 1, 0, 0), pointAt("b", "d", 1, 0, 0)}
	tracks, total := groupTracks(points)
	if len(tracks) != 4 || total.Points != 5 || total.Duration != 1800000 || math.Abs(total.Distance-111195.0802) > 0.1 {
		t.Fatalf("bad summary %+v", total)
	}
	for _, track := range tracks {
		if track.Count == 1 && (track.Distance != 0 || track.Duration != 0) {
			t.Fatal("single point has movement")
		}
	}
	if d := distance(pointAt("a", "d", 1, 0, 179.9), pointAt("a", "d", 2, 0, -179.9)); d > 23000 || d < 22000 {
		t.Fatal("antimeridian distance", d)
	}
}
func TestDateFilteringPaginationAndDetail(t *testing.T) {
	a := testApp(t)
	cookie := registerTestUser(t, a)
	base, _ := time.ParseInLocation("2006-01-02 15:04", "2026-01-01 23:55", a.location)
	for i := 0; i < 4; i++ {
		p := pointAt("daily", "1234567890abcdef", base.Add(time.Duration(i)*5*time.Minute).UnixMilli(), 0, float64(i))
		_, err := a.db.Exec("INSERT INTO points(reporting_id,device_hash,timestamp_ms,latitude,longitude) VALUES(?,?,?,?,?)", p.ID, p.Device, p.Time, p.Lat, p.Lon)
		if err != nil {
			t.Fatal(err)
		}
	}
	query := "?from=2026-01-02&to=2026-01-02&id=daily&device=1234567890abcdef"
	w := request(a, "GET", "/api/tracks"+query, "", cookie, "")
	expect(t, w, 200)
	var result struct {
		Tracks  []track `json:"tracks"`
		Summary summary `json:"summary"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Summary.Points != 3 || result.Summary.Duration != 600000 || len(result.Tracks) != 1 || len(result.Tracks[0].Points) != 0 {
		t.Fatalf("bad clipping %+v", result)
	}
	w = request(a, "GET", "/api/tracks/"+result.Tracks[0].Key+query, "", cookie, "")
	expect(t, w, 200)
	var detail track
	_ = json.Unmarshal(w.Body.Bytes(), &detail)
	if detail.Count != 3 || len(detail.Points) != 3 || detail.Distance != result.Summary.Distance {
		t.Fatal("detail differs from list")
	}
	for _, q := range []string{"?from=bad", "?from=2026-02-01&to=2026-01-01", "?page=0", "?page=abc"} {
		expect(t, request(a, "GET", "/api/tracks"+q, "", cookie, ""), 400)
	}
	for i := 0; i < 55; i++ {
		_, err := a.db.Exec("INSERT INTO points(reporting_id,device_hash,timestamp_ms,latitude,longitude) VALUES('many','1234567890abcdef',?,0,0)", base.Add(time.Duration(i)*31*time.Minute).UnixMilli())
		if err != nil {
			t.Fatal(err)
		}
	}
	w = request(a, "GET", "/api/tracks?from=2026-01-01&to=2026-01-03&id=many&page=2", "", cookie, "")
	expect(t, w, 200)
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	if result.Summary.Tracks != 55 || len(result.Tracks) != 5 {
		t.Fatal("pagination totals", result.Summary, len(result.Tracks))
	}
}
