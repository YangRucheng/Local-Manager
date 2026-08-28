// Package main 本地电气台账入口。
package main

import (
	"embed"
	"io/fs"
	"log"

	"electrical-ledger/internal/annex"
	"electrical-ledger/internal/config"
	"electrical-ledger/internal/handler"
	"electrical-ledger/internal/router"
	"electrical-ledger/internal/store"
)

//go:embed all:webdist
var embeddedDist embed.FS

func main() {
	cfg := config.FromEnv()

	st, err := store.Open(cfg.DBPath())
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer st.Close()

	if err := st.Migrate(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	// 启动时全量重算引用次数，保证 ref_count 与 annex_ref 始终一致
	if err := st.RecomputeAllCounts(); err != nil {
		log.Printf("启动重算引用次数失败: %v", err)
	}

	annexSvc := &annex.Service{AnnexDir: cfg.AnnexDir()}
	h := handler.New(st, annexSvc)

	var dist fs.FS = embeddedDist
	r := router.New(h, dist)

	log.Printf("本地电气台账已启动: http://127.0.0.1:%s  (数据目录: %s)", cfg.Port, cfg.DataDir)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
