package queue

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/tracktime-sh/cli/internal/config"
	"github.com/tracktime-sh/cli/internal/heartbeat"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS heartbeats (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	timestamp TEXT NOT NULL,
	project TEXT,
	language TEXT,
	editor TEXT NOT NULL,
	file_path TEXT,
	is_write INTEGER DEFAULT 0,
	created_at TEXT DEFAULT (datetime('now')),
	synced INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_heartbeats_synced ON heartbeats(synced);
`

type Queue struct {
	db *sql.DB
}

func dbPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "queue.db"), nil
}

func Open() (*Queue, error) {
	if err := config.EnsureDir(); err != nil {
		return nil, err
	}

	path, err := dbPath()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	return &Queue{db: db}, nil
}

func (q *Queue) Close() error {
	return q.db.Close()
}

func (q *Queue) Insert(hb *heartbeat.Heartbeat) error {
	_, err := q.db.Exec(`
		INSERT INTO heartbeats (timestamp, project, language, editor, file_path, is_write)
		VALUES (?, ?, ?, ?, ?, ?)
	`, hb.Timestamp.Format(time.RFC3339Nano), hb.Project, hb.Language, hb.Editor, hb.FilePath, boolToInt(hb.IsWrite))
	return err
}

func (q *Queue) InsertBatch(heartbeats []*heartbeat.Heartbeat) error {
	tx, err := q.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO heartbeats (timestamp, project, language, editor, file_path, is_write)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, hb := range heartbeats {
		_, err := stmt.Exec(hb.Timestamp.Format(time.RFC3339Nano), hb.Project, hb.Language, hb.Editor, hb.FilePath, boolToInt(hb.IsWrite))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (q *Queue) FetchUnsynced(limit int) ([]*heartbeat.Heartbeat, []int64, error) {
	rows, err := q.db.Query(`
		SELECT id, timestamp, project, language, editor, file_path, is_write
		FROM heartbeats
		WHERE synced = 0
		ORDER BY id ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var heartbeats []*heartbeat.Heartbeat
	var ids []int64

	for rows.Next() {
		var id int64
		var ts string
		var project, language, editor, filePath sql.NullString
		var isWrite int

		if err := rows.Scan(&id, &ts, &project, &language, &editor, &filePath, &isWrite); err != nil {
			return nil, nil, err
		}

		timestamp, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			timestamp, _ = time.Parse(time.RFC3339, ts)
		}

		hb := &heartbeat.Heartbeat{
			Timestamp: timestamp,
			Project:   nullStringToString(project),
			Language:  nullStringToString(language),
			Editor:    editor.String,
			FilePath:  nullStringToString(filePath),
			IsWrite:   isWrite == 1,
		}

		heartbeats = append(heartbeats, hb)
		ids = append(ids, id)
	}

	return heartbeats, ids, rows.Err()
}

func (q *Queue) MarkSynced(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := q.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE heartbeats SET synced = 1 WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.Exec(id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (q *Queue) DeleteSynced() (int64, error) {
	result, err := q.db.Exec("DELETE FROM heartbeats WHERE synced = 1")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (q *Queue) Count() (int, error) {
	var count int
	err := q.db.QueryRow("SELECT COUNT(*) FROM heartbeats WHERE synced = 0").Scan(&count)
	return count, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

type DayStats struct {
	Date     string `json:"date"`
	Duration string `json:"duration"`
	Minutes  int    `json:"minutes"`
}

type Stats struct {
	TotalMinutes int            `json:"total_minutes"`
	TotalTime    string         `json:"total_time"`
	Days         []DayStats     `json:"days"`
	ByProject    map[string]int `json:"by_project"`
	ByLanguage   map[string]int `json:"by_language"`
	ByEditor     map[string]int `json:"by_editor"`
	Heartbeats   int            `json:"heartbeats"`
}

func (q *Queue) GetStats(since time.Time) (*Stats, error) {
	stats := &Stats{
		Days:       []DayStats{},
		ByProject:  make(map[string]int),
		ByLanguage: make(map[string]int),
		ByEditor:   make(map[string]int),
	}

	rows, err := q.db.Query(`
		SELECT timestamp, project, language, editor
		FROM heartbeats
		WHERE timestamp >= ?
		ORDER BY timestamp ASC
	`, since.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var heartbeats []struct {
		timestamp time.Time
		project   string
		language  string
		editor    string
	}

	for rows.Next() {
		var ts string
		var project, language, editor sql.NullString

		if err := rows.Scan(&ts, &project, &language, &editor); err != nil {
			return nil, err
		}

		timestamp, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			timestamp, _ = time.Parse(time.RFC3339, ts)
		}

		heartbeats = append(heartbeats, struct {
			timestamp time.Time
			project   string
			language  string
			editor    string
		}{
			timestamp: timestamp,
			project:   nullStringToString(project),
			language:  nullStringToString(language),
			editor:    editor.String,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats.Heartbeats = len(heartbeats)

	if len(heartbeats) == 0 {
		stats.TotalTime = "0m"
		return stats, nil
	}

	dayMinutes := make(map[string]int)
	heartbeatInterval := 2 // Assume each heartbeat represents ~2 minutes of activity

	for _, hb := range heartbeats {
		day := hb.timestamp.Format("2006-01-02")
		dayMinutes[day] += heartbeatInterval

		if hb.project != "" {
			stats.ByProject[hb.project] += heartbeatInterval
		}
		if hb.language != "" {
			stats.ByLanguage[hb.language] += heartbeatInterval
		}
		if hb.editor != "" {
			stats.ByEditor[hb.editor] += heartbeatInterval
		}

		stats.TotalMinutes += heartbeatInterval
	}

	for day, minutes := range dayMinutes {
		stats.Days = append(stats.Days, DayStats{
			Date:     day,
			Minutes:  minutes,
			Duration: formatMinutes(minutes),
		})
	}

	stats.TotalTime = formatMinutes(stats.TotalMinutes)

	return stats, nil
}

func formatMinutes(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}
