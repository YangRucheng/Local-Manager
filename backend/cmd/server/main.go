// Package main 本地电气台账入口。
package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

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

	// 启动即确保数据目录与附件目录存在
	for _, dir := range []string{cfg.DataDir, cfg.AnnexDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("创建目录失败 %s: %v", dir, err)
		}
	}

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
		log.Printf("[告警] 启动重算引用次数失败: %v", err)
	}

	annexSvc := &annex.Service{AnnexDir: cfg.AnnexDir()}
	h := handler.New(st, annexSvc)

	var dist fs.FS = embeddedDist
	r := router.New(h, dist)

	printBanner(cfg)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

// printBanner 打印直观的启动信息。
func printBanner(cfg config.Config) {
	log.Println("===============================================")
	log.Println("  本地电气台账")
	log.Println("-----------------------------------------------")
	log.Printf("  监听地址:  http://127.0.0.1:%s", cfg.Port)
	log.Printf("  数据目录:  %s", cfg.DataDir)
	log.Printf("  附件目录:  %s", cfg.AnnexDir())
	log.Printf("  数据库文件: %s", cfg.DBPath())
	log.Println("===============================================")
	log.Printf("[启动] 服务就绪，请用浏览器打开 http://127.0.0.1:%s", cfg.Port)
}
