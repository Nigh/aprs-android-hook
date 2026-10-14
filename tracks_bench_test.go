package main

import (
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkTrackMap1000x30(b *testing.B) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		b.Fatal(err)
	}
	a, err := newApp(filepath.Join(b.TempDir(), "tracks.sqlite"), location, "", b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	defer a.db.Close()
	tx, err := a.db.Begin()
	if err != nil {
		b.Fatal(err)
	}
	statement, err := tx.Prepare(`INSERT INTO points(reporting_id,device_hash,session_id,timestamp_ms,latitude,longitude) VALUES(?,?,?,?,?,?)`)
	if err != nil {
		b.Fatal(err)
	}
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, location).UnixMilli()
	for i := range 1000 {
		for j := range 30 {
			_, err = statement.Exec("phone", "0123456789abcdef", fmt.Sprintf("%08x-0000-4000-8000-000000000000", i), start+int64(j)*60000, 30+float64(i)/10000+float64(j)/100000, 120+float64(i)/10000+float64(j)/100000)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
	statement.Close()
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/api/tracks?from=2026-01-01&to=2026-01-01&map=1", nil)
	b.ResetTimer()
	for range b.N {
		w := httptest.NewRecorder()
		a.tracks(w, r)
		if w.Code != 200 {
			b.Fatal(w.Body.String())
		}
	}
}
