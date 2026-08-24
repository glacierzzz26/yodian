// Package migrations 内嵌迁移 SQL，由 store.MigrateUp 启动时执行
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
