package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestImportTrack(t *testing.T) {
	a := testApp(t)
	token := strings.Repeat("a", 64)
	addTestToken(t, a, registerTestUser(t, a), token)
	makeBody := func(n int, session string) string {
		points := make([]map[string]any, n)
		for i := range points {
			points[i] = map[string]any{"type": "location", "id": "import", "device_hash": "0123456789abcdef", "session_id": session, "timestamp_ms": 1000 + i, "latitude": 31.2, "longitude": 121.4}
		}
		b, err := json.Marshal(points)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	body := makeBody(2000, "")
	expect(t, request(a, "POST", "/api/import", strings.Repeat(" ", 2<<20)+"[]", nil, token), 400)
	expect(t, request(a, "POST", "/api/import", body, nil, ""), 401)
	expect(t, request(a, "POST", "/api/import", body, nil, strings.Repeat("b", 64)), 401)
	for _, invalid := range []string{"null", "[]", "{}", makeBody(2001, ""), makeBody(2, "bad"), strings.Replace(makeBody(2, ""), `"latitude":31.2`, `"latitude":91`, 1), strings.Replace(makeBody(2, ""), `"timestamp_ms":1001`, `"timestamp_ms":1000`, 1), strings.Replace(makeBody(2, ""), `"id":"import"`, `"id":"other"`, 1), strings.Replace(makeBody(2, testSession), testSession, otherSession, 1), body + "[]"} {
		expect(t, request(a, "POST", "/api/import", invalid, nil, token), 400)
	}
	var count int
	a.db.QueryRow("SELECT count(*) FROM points").Scan(&count)
	if count != 0 {
		t.Fatal("invalid import saved points", count)
	}
	// Force a failure after the first insert to verify transaction rollback.
	if _, err := a.db.Exec(`CREATE TRIGGER reject_import BEFORE INSERT ON points WHEN NEW.timestamp_ms=1001 BEGIN SELECT RAISE(ABORT,'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	expect(t, request(a, "POST", "/api/import", makeBody(2, ""), nil, token), 503)
	a.db.QueryRow("SELECT count(*) FROM points").Scan(&count)
	if count != 0 {
		t.Fatal("partial import", count)
	}
	a.db.Exec("DROP TRIGGER reject_import")
	var wg sync.WaitGroup
	responses := make(chan string, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := request(a, "POST", "/api/import", body, nil, token)
			responses <- fmt.Sprintf("%d %s", w.Code, w.Body.String())
		}()
	}
	wg.Wait()
	close(responses)
	inserted, duplicates := 0, 0
	for response := range responses {
		if !strings.HasPrefix(response, "200 ") {
			t.Fatal(response)
		}
		if strings.Contains(response, `"duplicate":true`) {
			duplicates++
		} else {
			inserted++
		}
	}
	if inserted != 1 || duplicates != 1 {
		t.Fatal(inserted, duplicates)
	}
	var session string
	a.db.QueryRow("SELECT session_id FROM points LIMIT 1").Scan(&session)
	if !sessionPattern.MatchString(session) {
		t.Fatal(session)
	}
	a.db.QueryRow("SELECT count(*) FROM points").Scan(&count)
	if count != 2000 {
		t.Fatal(count)
	}
	expect(t, request(a, "POST", "/api/import", makeBody(2000, testSession), nil, token), 200)
	expect(t, request(a, "POST", "/api/import", makeBody(2, session), nil, token), 409)
	// A different endpoint must not be mistaken for an existing complete path.
	expect(t, request(a, "POST", "/api/import", makeBody(2, otherSession), nil, token), 200)
	a.db.QueryRow("SELECT count(*) FROM points WHERE session_id=?", otherSession).Scan(&count)
	if count != 2 {
		t.Fatal(count)
	}
	single := strings.Replace(makeBody(1, ""), `"id":"import"`, `"id":"single"`, 1)
	single = strings.Replace(single, `"session_id":""`, `"session_id":null`, 1)
	expect(t, request(a, "POST", "/api/import", single, nil, token), 200)
	w := request(a, "POST", "/api/import", single, nil, token)
	expect(t, w, 200)
	if !strings.Contains(w.Body.String(), `"duplicate":true`) {
		t.Fatal(w.Body.String())
	}
}
