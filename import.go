package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (a *app) importTrack(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") || !tokenPattern.MatchString(strings.TrimPrefix(header, "Bearer ")) {
		fail(w, 401, "上传 token 无效")
		return
	}
	var points []point
	if !decodeLimit(w, r, &points, 2<<20) {
		return
	}
	if len(points) == 0 || len(points) > 2000 {
		fail(w, 400, "每次导入必须包含1至2000个定位点")
		return
	}
	session := ""
	for i, p := range points {
		if p.Type != "location" || !validLocation(p) || p.ID != points[0].ID || p.Device != points[0].Device || (i > 0 && p.Time <= points[i-1].Time) {
			respond(w, 400, map[string]any{"error": "导入点必须有效、属于同一来源且时间严格递增", "code": "error.importPoint", "index": i})
			return
		}
		if p.Session != "" {
			if !sessionPattern.MatchString(p.Session) || (session != "" && session != p.Session) {
				fail(w, 400, "session_id 必须为空或为同一个标准小写 UUID v4")
				return
			}
			session = p.Session
		}
	}
	if session == "" {
		id, err := uuid.NewRandom()
		if err != nil {
			fail(w, 503, "保存失败，请重试")
			return
		}
		session = id.String()
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w, 503, "数据库暂不可用")
		return
	}
	defer tx.Rollback()
	var key string
	err = tx.QueryRowContext(r.Context(), "SELECT key FROM tokens WHERE hash=?", digest(strings.TrimPrefix(header, "Bearer "))).Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 401, "上传 token 未登记或已删除")
		return
	}
	if err != nil {
		fail(w, 503, "数据库暂不可用")
		return
	}
	first, last := points[0], points[len(points)-1]
	var existing string
	err = tx.QueryRowContext(r.Context(), `SELECT p.session_id FROM points p JOIN points q
 ON q.reporting_id=p.reporting_id AND q.device_hash=p.device_hash AND q.session_id=p.session_id
 WHERE p.reporting_id=? AND p.device_hash=? AND p.timestamp_ms=? AND p.latitude=? AND p.longitude=?
 AND q.timestamp_ms=? AND q.latitude=? AND q.longitude=?
 AND NOT EXISTS (SELECT 1 FROM points x WHERE x.reporting_id=p.reporting_id AND x.device_hash=p.device_hash AND x.session_id=p.session_id AND (x.timestamp_ms<p.timestamp_ms OR x.timestamp_ms>q.timestamp_ms))
 ORDER BY p.session_id LIMIT 1`, first.ID, first.Device, first.Time, first.Lat, first.Lon, last.Time, last.Lat, last.Lon).Scan(&existing)
	if err == nil {
		respond(w, 200, map[string]any{"ok": true, "duplicate": true, "session_id": existing, "inserted": 0})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		fail(w, 503, "数据库暂不可用")
		return
	}
	var count int
	if err = tx.QueryRowContext(r.Context(), "SELECT count(*) FROM points WHERE reporting_id=? AND device_hash=? AND session_id=?", first.ID, first.Device, session).Scan(&count); err != nil {
		fail(w, 503, "数据库暂不可用")
		return
	}
	if count != 0 {
		fail(w, 409, "导入会话已存在且首尾点不匹配")
		return
	}
	stmt, err := tx.PrepareContext(r.Context(), `INSERT INTO points(reporting_id,device_hash,session_id,timestamp_ms,latitude,longitude,accuracy_m,speed_mps,bearing_deg,altitude_m) VALUES(?,?,?,?,?,?,?,?,?,?)`)
	if err == nil {
		defer stmt.Close()
		for _, p := range points {
			if _, err = stmt.ExecContext(r.Context(), p.ID, p.Device, session, p.Time, p.Lat, p.Lon, p.Accuracy, p.Speed, p.Bearing, p.Altitude); err != nil {
				break
			}
		}
	}
	if err == nil {
		_, err = tx.ExecContext(r.Context(), "UPDATE tokens SET last_upload_ms=? WHERE key=?", time.Now().UnixMilli(), key)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("import storage error: %v", err)
		fail(w, 503, "保存失败，请重试")
		return
	}
	respond(w, 200, map[string]any{"ok": true, "duplicate": false, "session_id": session, "inserted": len(points)})
}
