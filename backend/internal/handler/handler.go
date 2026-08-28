// Package handler 提供 Gin 的 HTTP 处理器与输入校验。
package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"electrical-ledger/internal/annex"
	"electrical-ledger/internal/model"
	"electrical-ledger/internal/store"
)

// Handler 聚合各路由的处理逻辑。
type Handler struct {
	store *store.Store
	annex *annex.Service
}

// New 构造 Handler。
func New(st *store.Store, svc *annex.Service) *Handler {
	return &Handler{store: st, annex: svc}
}

// maxImages 每实体图片上限。
const maxImages = 9

// ---------- 公共辅助 ----------

func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		badRequest(c, "请求体格式错误: "+err.Error())
		return false
	}
	return true
}

func pathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "无效的 id")
		return 0, false
	}
	return id, true
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func notFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, gin.H{"error": msg})
}

func serverError(c *gin.Context, err error) {
	log.Printf("[server-error] %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// normalizeImageIDs 校验并返回去重后的图片 id 列表（≤9 且均存在）。
func (h *Handler) normalizeImageIDs(ids []int64) ([]int64, error) {
	ids = unique(ids)
	if len(ids) > maxImages {
		return nil, fmt.Errorf("图片最多 %d 张", maxImages)
	}
	missing, err := h.store.AnnexIDsExist(ids)
	if err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("附件不存在: %v", missing)
	}
	return ids, nil
}

func unique(in []int64) []int64 {
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

func requireName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("名称不能为空")
	}
	return nil
}

// respondStoreErr 统一处理 store 层错误。
func respondStoreErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		notFound(c, "记录不存在")
	case errors.Is(err, store.ErrDuplicateName):
		badRequest(c, "名称已存在")
	case errors.Is(err, store.ErrInvalid):
		badRequest(c, err.Error())
	default:
		serverError(c, err)
	}
}

// ---------- 配电室 ----------

// ListRooms GET /api/rooms
func (h *Handler) ListRooms(c *gin.Context) {
	rooms, err := h.store.ListRooms()
	if err != nil {
		serverError(c, err)
		return
	}
	ok(c, rooms)
}

// CreateRoom POST /api/rooms
func (h *Handler) CreateRoom(c *gin.Context) {
	var in model.RoomInput
	if !bindJSON(c, &in) {
		return
	}
	if err := requireName(in.Name); err != nil {
		badRequest(c, err.Error())
		return
	}
	ids, err := h.normalizeImageIDs(in.ImageIDs)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	in.ImageIDs = ids
	room, err := h.store.CreateRoom(in)
	if err != nil {
		respondStoreErr(c, err)
		return
	}
	ok(c, room)
}

// UpdateRoom PUT /api/rooms/:id
func (h *Handler) UpdateRoom(c *gin.Context) {
	id, okID := pathID(c)
	if !okID {
		return
	}
	var in model.RoomInput
	if !bindJSON(c, &in) {
		return
	}
	if err := requireName(in.Name); err != nil {
		badRequest(c, err.Error())
		return
	}
	ids, err := h.normalizeImageIDs(in.ImageIDs)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	in.ImageIDs = ids
	room, err := h.store.UpdateRoom(id, in)
	if err != nil {
		respondStoreErr(c, err)
		return
	}
	ok(c, room)
}

// DeleteRoom DELETE /api/rooms/:id
func (h *Handler) DeleteRoom(c *gin.Context) {
	id, okID := pathID(c)
	if !okID {
		return
	}
	if err := h.store.DeleteRoom(id); err != nil {
		respondStoreErr(c, err)
		return
	}
	ok(c, gin.H{"deleted": id})
}

// ---------- 配电柜 ----------

// ListCabinets GET /api/cabinets?room_id=
func (h *Handler) ListCabinets(c *gin.Context) {
	roomID, _ := strconv.ParseInt(c.Query("room_id"), 10, 64)
	list, err := h.store.ListCabinets(roomID)
	if err != nil {
		serverError(c, err)
		return
	}
	ok(c, list)
}

// CreateCabinet POST /api/cabinets
func (h *Handler) CreateCabinet(c *gin.Context) {
	var in model.CabinetInput
	if !bindJSON(c, &in) {
		return
	}
	if in.RoomID <= 0 {
		badRequest(c, "请选择所属配电室")
		return
	}
	if err := requireName(in.Name); err != nil {
		badRequest(c, err.Error())
		return
	}
	ids, err := h.normalizeImageIDs(in.ImageIDs)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	in.ImageIDs = ids
	cab, err := h.store.CreateCabinet(in)
	if err != nil {
		respondStoreErr(c, err)
		return
	}
	ok(c, cab)
}

// UpdateCabinet PUT /api/cabinets/:id
func (h *Handler) UpdateCabinet(c *gin.Context) {
	id, okID := pathID(c)
	if !okID {
		return
	}
	var in model.CabinetInput
	if !bindJSON(c, &in) {
		return
	}
	if in.RoomID <= 0 {
		badRequest(c, "请选择所属配电室")
		return
	}
	if err := requireName(in.Name); err != nil {
		badRequest(c, err.Error())
		return
	}
	ids, err := h.normalizeImageIDs(in.ImageIDs)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	in.ImageIDs = ids
	cab, err := h.store.UpdateCabinet(id, in)
	if err != nil {
		respondStoreErr(c, err)
		return
	}
	ok(c, cab)
}

// DeleteCabinet DELETE /api/cabinets/:id
func (h *Handler) DeleteCabinet(c *gin.Context) {
	id, okID := pathID(c)
	if !okID {
		return
	}
	if err := h.store.DeleteCabinet(id); err != nil {
		respondStoreErr(c, err)
		return
	}
	ok(c, gin.H{"deleted": id})
}

// ---------- 台账记录 ----------

// ListEquipment GET /api/equipment?room_id=&cabinet_id=&keyword=&page=&page_size=
func (h *Handler) ListEquipment(c *gin.Context) {
	roomID, _ := strconv.ParseInt(c.Query("room_id"), 10, 64)
	cabinetID, _ := strconv.ParseInt(c.Query("cabinet_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.store.ListEquipment(store.EquipmentFilter{
		RoomID:    roomID,
		CabinetID: cabinetID,
		Keyword:   c.Query("keyword"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		serverError(c, err)
		return
	}
	ok(c, result)
}

// GetEquipment GET /api/equipment/:id
func (h *Handler) GetEquipment(c *gin.Context) {
	id, okID := pathID(c)
	if !okID {
		return
	}
	eq, err := h.store.GetEquipment(id)
	if err != nil {
		respondStoreErr(c, err)
		return
	}
	ok(c, eq)
}

// CreateEquipment POST /api/equipment
func (h *Handler) CreateEquipment(c *gin.Context) {
	var in model.EquipmentInput
	if !bindJSON(c, &in) {
		return
	}
	if in.RoomID <= 0 {
		badRequest(c, "请选择所属配电室")
		return
	}
	if err := requireName(in.Name); err != nil {
		badRequest(c, err.Error())
		return
	}
	ids, err := h.normalizeImageIDs(in.ImageIDs)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	in.ImageIDs = ids
	eq, err := h.store.CreateEquipment(in)
	if err != nil {
		respondStoreErr(c, err)
		return
	}
	ok(c, eq)
}

// UpdateEquipment PUT /api/equipment/:id
func (h *Handler) UpdateEquipment(c *gin.Context) {
	id, okID := pathID(c)
	if !okID {
		return
	}
	var in model.EquipmentInput
	if !bindJSON(c, &in) {
		return
	}
	if in.RoomID <= 0 {
		badRequest(c, "请选择所属配电室")
		return
	}
	if err := requireName(in.Name); err != nil {
		badRequest(c, err.Error())
		return
	}
	ids, err := h.normalizeImageIDs(in.ImageIDs)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	in.ImageIDs = ids
	eq, err := h.store.UpdateEquipment(id, in)
	if err != nil {
		respondStoreErr(c, err)
		return
	}
	ok(c, eq)
}

// DeleteEquipment DELETE /api/equipment/:id
func (h *Handler) DeleteEquipment(c *gin.Context) {
	id, okID := pathID(c)
	if !okID {
		return
	}
	if err := h.store.DeleteEquipment(id); err != nil {
		respondStoreErr(c, err)
		return
	}
	ok(c, gin.H{"deleted": id})
}

// ---------- 附件 ----------

func annexURL(id int64) string {
	return "/api/annex/" + strconv.FormatInt(id, 10) + "/file"
}

// UploadAnnex POST /api/annex/upload （multipart 字段名 file）
func (h *Handler) UploadAnnex(c *gin.Context) {
	header, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "缺少上传文件（字段名 file）")
		return
	}
	uid, ext, err := h.annex.SaveFile(header)
	if err != nil {
		switch {
		case errors.Is(err, annex.ErrFileTooLarge), errors.Is(err, annex.ErrBadExt):
			badRequest(c, err.Error())
		default:
			serverError(c, err)
		}
		return
	}
	a, err := h.store.InsertAnnex(model.Annex{
		UUID:         uid,
		OriginalName: filepath.Base(header.Filename),
		Ext:          ext,
		MimeType:     annex.MimeFor(ext),
		Size:         header.Size,
	})
	if err != nil {
		// 入库失败时回滚磁盘文件
		_ = h.annex.DeleteFile(model.Annex{UUID: uid, Ext: ext})
		serverError(c, err)
		return
	}
	a.URL = annexURL(a.ID)
	ok(c, a)
}

// GetAnnex GET /api/annex/:id
func (h *Handler) GetAnnex(c *gin.Context) {
	id, okID := pathID(c)
	if !okID {
		return
	}
	a, err := h.store.GetAnnex(id)
	if err != nil {
		respondStoreErr(c, err)
		return
	}
	a.URL = annexURL(a.ID)
	ok(c, a)
}

// ServeAnnexFile GET /api/annex/:id/file
func (h *Handler) ServeAnnexFile(c *gin.Context) {
	id, okID := pathID(c)
	if !okID {
		return
	}
	a, err := h.store.GetAnnex(id)
	if err != nil {
		respondStoreErr(c, err)
		return
	}
	path := h.annex.FilePath(a)
	if _, err := os.Stat(path); err != nil {
		notFound(c, "图片文件不存在")
		return
	}
	if mime := a.MimeType; mime != "" {
		c.Header("Content-Type", mime)
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.File(path)
}

// RecomputeAnnex POST /api/annex/recompute —— 随时可重算引用次数
func (h *Handler) RecomputeAnnex(c *gin.Context) {
	if err := h.store.RecomputeAllCounts(); err != nil {
		serverError(c, err)
		return
	}
	ok(c, gin.H{"recomputed": true})
}

// CleanupAnnex POST /api/annex/cleanup —— 清理 ref_count=0 的孤儿附件（行 + 磁盘文件）
func (h *Handler) CleanupAnnex(c *gin.Context) {
	if err := h.store.RecomputeAllCounts(); err != nil {
		serverError(c, err)
		return
	}
	orphans, err := h.store.ListOrphanAnnexes()
	if err != nil {
		serverError(c, err)
		return
	}
	type cleaned struct {
		ID           int64  `json:"id"`
		OriginalName string `json:"original_name"`
	}
	var deleted []cleaned
	for _, a := range orphans {
		_ = h.annex.DeleteFile(a)
		if err := h.store.DeleteAnnex(a.ID); err != nil {
			serverError(c, err)
			return
		}
		deleted = append(deleted, cleaned{ID: a.ID, OriginalName: a.OriginalName})
	}
	ok(c, gin.H{"cleaned": deleted, "count": len(deleted)})
}
