package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type point struct {
	Type     string   `json:"type"`
	ID       string   `json:"id"`
	Device   string   `json:"device_hash"`
	Session  string   `json:"session_id"`
	Time     int64    `json:"timestamp_ms"`
	Lat      *float64 `json:"latitude"`
	Lon      *float64 `json:"longitude"`
	Accuracy *float64 `json:"accuracy_m"`
	Speed    *float64 `json:"speed_mps"`
	Bearing  *float64 `json:"bearing_deg"`
	Altitude *float64 `json:"altitude_m"`
}
type track struct {
	Key      string  `json:"key"`
	ID       string  `json:"id"`
	Device   string  `json:"device_hash"`
	Session  string  `json:"session_id"`
	Start    int64   `json:"start_ms"`
	End      int64   `json:"end_ms"`
	Count    int     `json:"point_count"`
	Distance float64 `json:"distance_m"`
	Duration int64   `json:"duration_ms"`
	Points   []point `json:"points,omitempty"`
}
type summary struct {
	Tracks   int     `json:"track_count"`
	Points   int     `json:"point_count"`
	Distance float64 `json:"distance_m"`
	Duration int64   `json:"duration_ms"`
}
type mapTrack struct {
	Key         string       `json:"key"`
	Page        int          `json:"page"`
	Coordinates [][2]float64 `json:"coordinates"`
}

func distance(a, b point) float64 {
	rad := math.Pi / 180
	lat1, lat2 := *a.Lat*rad, *b.Lat*rad
	s := math.Pow(math.Sin((lat2-lat1)/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin((*b.Lon-*a.Lon)*rad/2), 2)
	return 6371008.8 * 2 * math.Atan2(math.Sqrt(math.Min(1, s)), math.Sqrt(math.Max(0, 1-s)))
}

func groupTracks(points []point) ([]track, summary) {
	tracks := []track{}
	for _, p := range points {
		var current *track
		if len(tracks) > 0 {
			current = &tracks[len(tracks)-1]
		}
		if current == nil || current.ID != p.ID || current.Device != p.Device || current.Session != p.Session {
			keyData, _ := json.Marshal([]string{p.ID, p.Device, p.Session})
			tracks = append(tracks, track{Key: base64.RawURLEncoding.EncodeToString(keyData), ID: p.ID, Device: p.Device, Session: p.Session, Start: p.Time, End: p.Time})
			current = &tracks[len(tracks)-1]
		}
		if current.Count > 0 {
			current.Distance += distance(current.Points[current.Count-1], p)
		}
		current.Points = append(current.Points, p)
		current.Count++
		current.End = p.Time
		current.Duration = current.End - current.Start
	}
	sort.Slice(tracks, func(i, j int) bool {
		if tracks[i].End == tracks[j].End {
			return tracks[i].Key < tracks[j].Key
		}
		return tracks[i].End > tracks[j].End
	})
	total := summary{Tracks: len(tracks)}
	for _, t := range tracks {
		total.Points += t.Count
		total.Distance += t.Distance
		total.Duration += t.Duration
	}
	return tracks, total
}

func (a *app) trackRange(r *http.Request) (time.Time, time.Time, error) {
	q := r.URL.Query()
	now := time.Now().In(a.location)
	from := q.Get("from")
	if from == "" {
		from = now.Format("2006-01-02")
	}
	to := q.Get("to")
	if to == "" {
		to = from
	}
	start, err := time.ParseInLocation("2006-01-02", from, a.location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("开始日期无效")
	}
	end, err := time.ParseInLocation("2006-01-02", to, a.location)
	if err != nil || end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("结束日期不能早于开始日期")
	}
	return start, end.AddDate(0, 0, 1), nil
}

func scanTracks(rows *sql.Rows, withMetrics bool) ([]track, summary, error) {
	defer rows.Close()
	points := []point{}
	for rows.Next() {
		p := point{Type: "location"}
		var err error
		if withMetrics {
			err = rows.Scan(&p.ID, &p.Device, &p.Session, &p.Time, &p.Lat, &p.Lon, &p.Accuracy, &p.Speed, &p.Bearing, &p.Altitude)
		} else {
			err = rows.Scan(&p.ID, &p.Device, &p.Session, &p.Time, &p.Lat, &p.Lon)
		}
		if err != nil {
			return nil, summary{}, err
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, summary{}, err
	}
	tracks, total := groupTracks(points)
	return tracks, total, nil
}

func (a *app) queryTracks(r *http.Request) ([]track, summary, error) {
	start, end, err := a.trackRange(r)
	if err != nil {
		return nil, summary{}, err
	}
	q := r.URL.Query()
	rows, err := a.db.QueryContext(r.Context(), `SELECT reporting_id,device_hash,session_id,timestamp_ms,latitude,longitude FROM points
WHERE timestamp_ms>=?1 AND timestamp_ms<?2 AND (?3='' OR reporting_id=?3) AND (?4='' OR device_hash=?4)
ORDER BY reporting_id,device_hash,session_id,timestamp_ms`, start.UnixMilli(), end.UnixMilli(), q.Get("id"), q.Get("device"))
	if err != nil {
		return nil, summary{}, err
	}
	return scanTracks(rows, false)
}

func (a *app) tracks(w http.ResponseWriter, r *http.Request) {
	tracks, total, err := a.queryTracks(r)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	page := 1
	if value := r.URL.Query().Get("page"); value != "" {
		page, err = strconv.Atoi(value)
		if err != nil || page < 1 || page > 1000000 {
			fail(w, 400, "页码无效")
			return
		}
	}
	start := min((page-1)*50, len(tracks))
	end := min(start+50, len(tracks))
	items := tracks[start:end]
	var paths []mapTrack
	var allTracks []track
	if r.URL.Query().Get("map") == "1" {
		paths = make([]mapTrack, 0, len(tracks))
		for i, t := range tracks {
			if len(t.Points) < 2 {
				continue
			}
			path := mapTrack{Key: t.Key, Page: i/50 + 1, Coordinates: make([][2]float64, len(t.Points))}
			for j, p := range t.Points {
				path.Coordinates[j] = [2]float64{*p.Lon, *p.Lat}
			}
			paths = append(paths, path)
		}
		allTracks = tracks
	}
	for i := range tracks {
		tracks[i].Points = nil
	}
	respond(w, 200, map[string]any{"tracks": items, "all_tracks": allTracks, "paths": paths, "summary": total, "page": page, "page_size": 50, "timezone": a.location.String()})
}
func (a *app) trackDetail(w http.ResponseWriter, r *http.Request) {
	start, end, err := a.trackRange(r)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	key, err := base64.RawURLEncoding.DecodeString(r.PathValue("key"))
	var parts []string
	if err != nil || json.Unmarshal(key, &parts) != nil || len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		fail(w, 404, "路径已变化，请刷新列表")
		return
	}
	q := r.URL.Query()
	if (q.Get("id") != "" && q.Get("id") != parts[0]) || (q.Get("device") != "" && q.Get("device") != parts[1]) {
		fail(w, 404, "路径已变化，请刷新列表")
		return
	}
	rows, err := a.db.QueryContext(r.Context(), `SELECT reporting_id,device_hash,session_id,timestamp_ms,latitude,longitude,accuracy_m,speed_mps,bearing_deg,altitude_m FROM points
WHERE reporting_id=?1 AND device_hash=?2 AND session_id=?3 AND timestamp_ms>=?4 AND timestamp_ms<?5
ORDER BY timestamp_ms`, parts[0], parts[1], parts[2], start.UnixMilli(), end.UnixMilli())
	if err != nil {
		fail(w, 500, "无法读取路径")
		return
	}
	tracks, _, err := scanTracks(rows, true)
	if err != nil {
		fail(w, 500, "无法读取路径")
		return
	}
	if len(tracks) == 1 && tracks[0].Key == r.PathValue("key") {
		respond(w, 200, tracks[0])
		return
	}
	fail(w, 404, "路径已变化，请刷新列表")
}
func (a *app) sources(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(), `SELECT DISTINCT reporting_id,device_hash FROM points ORDER BY reporting_id,device_hash`)
	if err != nil {
		fail(w, 500, "无法读取设备")
		return
	}
	defer rows.Close()
	items := []map[string]string{}
	for rows.Next() {
		var id, device string
		if err := rows.Scan(&id, &device); err != nil {
			fail(w, 500, "无法读取设备")
			return
		}
		items = append(items, map[string]string{"id": id, "device_hash": device})
	}
	if rows.Err() != nil {
		fail(w, 500, "无法读取设备")
		return
	}
	respond(w, 200, items)
}
