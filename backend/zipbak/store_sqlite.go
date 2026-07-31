package zipbak

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	metaRowID     = 1
	schemaVersion = 4
)

// Store persists backup progress and the file manifest in SQLite.
type Store struct {
	path string
	db   *sql.DB
}

type metaRow struct {
	SourceDir      string
	StagingDir     string
	MaxPartBytes   int64
	NextFileIndex  int
	PendingZip     string
	PrefetchZip    string
	PartSerial     int
	PartNamePrefix string
	Done           bool
	FileCount      int
}

func OpenStore(statePath string) (*Store, error) {
	statePath = filepath.Clean(strings.TrimSpace(statePath))
	if statePath == "" {
		return nil, fmt.Errorf("state 路径为空")
	}
	if err := migrateLegacyStateFiles(statePath); err != nil {
		return nil, err
	}
	if err := migrateJSONIfNeeded(statePath); err != nil {
		return nil, err
	}
	return openStore(statePath)
}

func openStore(statePath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", sqliteDSN(statePath))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{path: statePath, db: db}
	if err := s.ensureSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func sqliteDSN(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	// modernc.org/sqlite URI: busy_timeout 避免 pack/init 写入时 status 立刻失败
	slash := filepath.ToSlash(abs)
	return "file:" + slash + "?_pragma=busy_timeout(15000)&_pragma=journal_mode(WAL)"
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) ensureSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_info (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			version INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS meta (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			source_dir TEXT NOT NULL,
			staging_dir TEXT NOT NULL,
			max_part_bytes INTEGER NOT NULL,
			next_file_index INTEGER NOT NULL DEFAULT 0,
			pending_zip TEXT NOT NULL DEFAULT '',
			prefetch_zip TEXT NOT NULL DEFAULT '',
			part_serial INTEGER NOT NULL DEFAULT 0,
			part_name_prefix TEXT NOT NULL DEFAULT '',
			done INTEGER NOT NULL DEFAULT 0,
			file_count INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS files (
			seq INTEGER PRIMARY KEY,
			rel_path TEXT NOT NULL UNIQUE,
			size_bytes INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_files_seq ON files(seq)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	var ver int
	err := s.db.QueryRow(`SELECT version FROM schema_info WHERE id = 1`).Scan(&ver)
	if err == sql.ErrNoRows {
		_, err = s.db.Exec(`INSERT INTO schema_info(id, version) VALUES (1, ?)`, schemaVersion)
		return err
	}
	if err != nil {
		return err
	}
	return s.applySchemaMigrations(ver)
}

func (s *Store) applySchemaMigrations(ver int) error {
	if ver > schemaVersion {
		return fmt.Errorf("不支持的 state 数据库版本 %d", ver)
	}
	if ver < 2 {
		if err := s.ensureFilesSizeColumn(); err != nil {
			return err
		}
		if _, err := s.db.Exec(`UPDATE schema_info SET version = 2 WHERE id = 1`); err != nil {
			return err
		}
		ver = 2
	}
	if ver < 3 {
		if err := s.ensurePrefetchZipColumn(); err != nil {
			return err
		}
		if _, err := s.db.Exec(`UPDATE schema_info SET version = 3 WHERE id = 1`); err != nil {
			return err
		}
		ver = 3
	}
	if ver < 4 {
		if err := s.ensurePartNamePrefixColumn(); err != nil {
			return err
		}
		if _, err := s.db.Exec(`UPDATE schema_info SET version = 4 WHERE id = 1`); err != nil {
			return err
		}
		ver = 4
	}
	if ver != schemaVersion {
		return fmt.Errorf("不支持的 state 数据库版本 %d", ver)
	}
	return nil
}

func (s *Store) ensureFilesSizeColumn() error {
	rows, err := s.db.Query(`PRAGMA table_info(files)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	hasSize := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "size_bytes" {
			hasSize = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if hasSize {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE files ADD COLUMN size_bytes INTEGER NOT NULL DEFAULT 0`)
	return err
}

func (s *Store) ensurePrefetchZipColumn() error {
	rows, err := s.db.Query(`PRAGMA table_info(meta)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	has := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "prefetch_zip" {
			has = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if has {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE meta ADD COLUMN prefetch_zip TEXT NOT NULL DEFAULT ''`)
	return err
}

func (s *Store) ensurePartNamePrefixColumn() error {
	rows, err := s.db.Query(`PRAGMA table_info(meta)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	has := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "part_name_prefix" {
			has = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if has {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE meta ADD COLUMN part_name_prefix TEXT NOT NULL DEFAULT ''`)
	return err
}

func (s *Store) Init(sourceDir, stagingDir, partPrefix string, files []ManifestEntry) error {
	sourceDir = filepath.Clean(sourceDir)
	stagingDir = filepath.Clean(stagingDir)
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM files`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM meta`); err != nil {
		return err
	}

	done := 0
	if len(files) == 0 {
		done = 1
	}
	_, err = tx.Exec(`INSERT INTO meta(id, source_dir, staging_dir, max_part_bytes, next_file_index, pending_zip, prefetch_zip, part_serial, part_name_prefix, done, file_count)
		VALUES (1, ?, ?, 0, 0, '', '', 0, ?, ?, ?)`,
		sourceDir, stagingDir, SanitizePartPrefix(partPrefix), done, len(files))
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO files(seq, rel_path, size_bytes) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, ent := range files {
		if _, err := stmt.Exec(i, ent.RelPath, ent.SizeBytes); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) loadMeta() (metaRow, error) {
	var m metaRow
	var done int
	err := s.db.QueryRow(`SELECT source_dir, staging_dir, max_part_bytes, next_file_index, pending_zip, prefetch_zip, part_serial, part_name_prefix, done, file_count
		FROM meta WHERE id = 1`).Scan(
		&m.SourceDir, &m.StagingDir, &m.MaxPartBytes, &m.NextFileIndex, &m.PendingZip, &m.PrefetchZip, &m.PartSerial, &m.PartNamePrefix, &done, &m.FileCount,
	)
	if err == sql.ErrNoRows {
		return m, fmt.Errorf("state 未初始化，请先 init")
	}
	if err != nil {
		return m, err
	}
	m.Done = done != 0
	return m, nil
}

func (s *Store) updateMeta(m metaRow) error {
	done := 0
	if m.Done {
		done = 1
	}
	_, err := s.db.Exec(`UPDATE meta SET next_file_index=?, pending_zip=?, prefetch_zip=?, part_serial=?, done=? WHERE id=1`,
		m.NextFileIndex, m.PendingZip, m.PrefetchZip, m.PartSerial, done)
	return err
}

func (s *Store) readStatus(maxPartBytes int64) (Status, error) {
	m, err := s.loadMeta()
	if err != nil {
		return Status{}, err
	}
	maxFile, err := s.maxManifestFileBytes()
	if err != nil {
		return Status{}, err
	}
	oversized, err := s.oversizedFileCount(m, maxPartBytes)
	if err != nil {
		return Status{}, err
	}
	return Status{
		TotalFiles:         m.FileCount,
		PackedFiles:        m.NextFileIndex,
		PendingZip:         m.PendingZip,
		PrefetchZip:        m.PrefetchZip,
		Done:               m.Done,
		NextFileIndex:      m.NextFileIndex,
		MaxFileBytes:       maxFile,
		OversizedFileCount: oversized,
	}, nil
}

func (s *Store) maxManifestFileBytes() (int64, error) {
	var max sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(size_bytes) FROM files`).Scan(&max)
	if err != nil {
		return 0, err
	}
	if max.Valid {
		return max.Int64, nil
	}
	return 0, nil
}

func (s *Store) oversizedFileCount(m metaRow, maxPartBytes int64) (int, error) {
	if maxPartBytes <= 0 {
		return 0, fmt.Errorf("max part size must be positive")
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM files WHERE size_bytes > ?`, maxPartBytes).Scan(&n); err != nil {
		return 0, err
	}
	rows, err := s.db.Query(`SELECT rel_path FROM files WHERE size_bytes = 0`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var rel string
		if err := rows.Scan(&rel); err != nil {
			return 0, err
		}
		size, err := fileSizeFromStore(0, m.SourceDir, rel)
		if err != nil {
			return 0, err
		}
		if size > maxPartBytes {
			n++
		}
	}
	return n, rows.Err()
}

func (s *Store) validateMeta(m metaRow) error {
	if m.SourceDir == "" || m.StagingDir == "" {
		return fmt.Errorf("state 缺少 source_dir 或 staging_dir")
	}
	return nil
}
