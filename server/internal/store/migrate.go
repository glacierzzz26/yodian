package store

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/yodian/server/migrations"
)

// MigrateUp 启动时执行 migrations/*.up.sql（按文件名序），记录到 schema_migrations。
// 同一文件在事务内执行 + 记录，失败整体回滚。down 文件保留供手工回滚（设计 15.5 可逆脚本）。
func MigrateUp(db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`).Error; err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	entries, err := fs.Glob(migrations.FS, "*.up.sql")
	if err != nil {
		return err
	}
	sort.Strings(entries)

	for _, e := range entries {
		version := strings.TrimSuffix(path.Base(e), ".up.sql")
		var cnt int64
		if err := db.Raw(`SELECT count(*) FROM schema_migrations WHERE version = ?`, version).Scan(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			slog.Debug("migration already applied", "version", version)
			continue
		}
		sqlBytes, err := migrations.FS.ReadFile(e)
		if err != nil {
			return err
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(sqlBytes)).Error; err != nil {
				return err
			}
			return tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version).Error
		}); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		slog.Info("migration applied", "version", version)
	}
	return nil
}
