// Package router 装配 Gin 路由并托管前端静态资源。
package router

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"electrical-ledger/internal/handler"
)

// New 构造 gin.Engine：挂载 /api 路由；distFS 非空时托管前端（SPA fallback）。
func New(h *handler.Handler, distFS fs.FS) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	api := r.Group("/api")
	{
		// 配电室
		api.GET("/rooms", h.ListRooms)
		api.POST("/rooms", h.CreateRoom)
		api.PUT("/rooms/:id", h.UpdateRoom)
		api.DELETE("/rooms/:id", h.DeleteRoom)
		// 配电柜
		api.GET("/cabinets", h.ListCabinets)
		api.POST("/cabinets", h.CreateCabinet)
		api.PUT("/cabinets/:id", h.UpdateCabinet)
		api.DELETE("/cabinets/:id", h.DeleteCabinet)
		// 台账记录
		api.GET("/equipment", h.ListEquipment)
		api.GET("/equipment/:id", h.GetEquipment)
		api.POST("/equipment", h.CreateEquipment)
		api.PUT("/equipment/:id", h.UpdateEquipment)
		api.DELETE("/equipment/:id", h.DeleteEquipment)
		// 附件
		api.POST("/annex/upload", h.UploadAnnex)
		api.GET("/annex/:id", h.GetAnnex)
		api.GET("/annex/:id/file", h.ServeAnnexFile)
		api.POST("/annex/recompute", h.RecomputeAnnex)
		api.POST("/annex/cleanup", h.CleanupAnnex)
	}

	if distFS != nil {
		mountStatic(r, distFS)
	} else {
		r.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "接口不存在"})
		})
	}
	return r
}

// mountStatic 以 SPA 方式托管前端产物：命中文件返回文件，否则回退 index.html。
func mountStatic(r *gin.Engine, dist fs.FS) {
	sub, err := fs.Sub(dist, "webdist")
	if err != nil {
		// webdist 缺失时仅提供 API
		return
	}
	fileServer := http.FileServer(http.FS(sub))
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "接口不存在"})
			return
		}
		p := strings.TrimPrefix(c.Request.URL.Path, "/")
		if p != "" {
			if f, err := sub.Open(p); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		index, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.String(http.StatusNotFound, "前端未构建：请先运行 make frontend")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})
}
