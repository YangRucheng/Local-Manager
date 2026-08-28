// Package store 提供 sqlite 数据访问：打开、迁移与各表的类型化操作。
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	// 纯 Go 实现的 SQLite 驱动：CGO_ENABLED=0 即可交叉编译各平台静态单文件。
	_ "modernc.org/sqlite"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("记录不存在")

// ErrDuplicateName 名称唯一性冲突。
var ErrDuplicateName = errors.New("名称已存在")

// ErrInvalid 业务校验失败（应返回 400）。
var ErrInvalid = errors.New("参数不合法")

// invalidf 构造带 ErrInvalid 标记的校验错误。
func invalidf(format string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), ErrInvalid)
}

// Store 封装 sqlite 数据库。
type Store struct {
	db *sql.DB
}

// Open 打开（必要时创建）数据目录下的 sqlite 数据库并启用 WAL。
func Open(dbPath string) (*Store, error) {
	if dir := dirOf(dbPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据目录: %w", err)
		}
	}
	dsn := "file:" + dbPath +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}
	// 本地单用户场景，单连接最稳妥，避免写锁冲突。
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	return &Store{db: db}, nil
}

func dirOf(path string) string {
	i := strings.LastIndexByte(path, '/')
	if i <= 0 {
		return ""
	}
	return path[:i]
}

// Close 关闭数据库。
func (s *Store) Close() error { return s.db.Close() }

// DB 暴露底层句柄，供需要原始事务的调用方使用。
func (s *Store) DB() *sql.DB { return s.db }

// Migrate 建表（幂等）。使用 PRAGMA foreign_keys=on 依赖外键级联。
func (s *Store) Migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS rooms (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    remark     TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS cabinets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id    INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    remark     TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(room_id, name)
);
CREATE TABLE IF NOT EXISTS equipment (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id      INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    cabinet_id   INTEGER REFERENCES cabinets(id) ON DELETE SET NULL,
    name         TEXT NOT NULL,
    model        TEXT NOT NULL DEFAULT '',
    manufacturer TEXT NOT NULL DEFAULT '',
    quantity     INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    remark       TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS annex (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid          TEXT NOT NULL UNIQUE,
    original_name TEXT NOT NULL,
    ext           TEXT NOT NULL DEFAULT '',
    mime_type     TEXT NOT NULL DEFAULT '',
    size          INTEGER NOT NULL DEFAULT 0,
    ref_count     INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS annex_ref (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    annex_id    INTEGER NOT NULL REFERENCES annex(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,
    target_id   INTEGER NOT NULL,
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL,
    UNIQUE(annex_id, target_type, target_id, position)
);
CREATE INDEX IF NOT EXISTS idx_annex_ref_target ON annex_ref(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_annex_ref_annex  ON annex_ref(annex_id);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("迁移数据库: %w", err)
	}
	return nil
}

func now() string { return time.Now().Format("2006-01-02 15:04:05") }

// isUniqueViolation 判断是否 sqlite 唯一约束冲突。
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// mapErr 将底层错误转换为领域错误（重复名 → ErrDuplicateName）。
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if isUniqueViolation(err) {
		return ErrDuplicateName
	}
	return err
}

// execer 抽象 sql.DB 与 sql.Tx，便于事务内外复用。
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// uniqueInt64 去重并保持顺序。
func uniqueInt64(in []int64) []int64 {
	seen := make(map[int64]struct{}, len(in))
	out := make([]int64, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// placeholders 生成 "?,?,?" 形式的占位符串。
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
