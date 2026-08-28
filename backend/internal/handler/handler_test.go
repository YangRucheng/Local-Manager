package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"electrical-ledger/internal/annex"
	"electrical-ledger/internal/handler"
	"electrical-ledger/internal/model"
	"electrical-ledger/internal/router"
	"electrical-ledger/internal/store"
)

// tinyPNG 1x1 像素 PNG。
var tinyPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
	0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
}

type testEnv struct {
	engine   *gin.Engine
	store    *store.Store
	annexDir string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "app.db"))
	if err != nil {
		t.Fatalf("打开测试库失败: %v", err)
	}
	if err := st.Migrate(); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	svc := &annex.Service{AnnexDir: filepath.Join(dir, "annex")}
	engine := router.New(handler.New(st, svc), nil) // 测试仅 API
	t.Cleanup(func() { _ = st.Close() })
	return &testEnv{engine: engine, store: st, annexDir: filepath.Join(dir, "annex")}
}

func doJSON(t *testing.T, env *testEnv, method, path string, body any) (*httptest.ResponseRecorder, any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("编码请求失败: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)
	var parsed any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &parsed)
	}
	return rec, parsed
}

func uploadAnnex(t *testing.T, env *testEnv, filename string, content []byte) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("创建表单失败: %v", err)
	}
	_, _ = fw.Write(content)
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/annex/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)
	var out map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}
	return rec, out
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asInt(v any) int64 {
	if f, ok := v.(float64); ok {
		return int64(f)
	}
	return 0
}

func TestRoomAPI(t *testing.T) {
	env := newTestEnv(t)

	rec, _ := doJSON(t, env, "POST", "/api/rooms", model.RoomInput{Name: "一号配电室"})
	if rec.Code != http.StatusOK {
		t.Fatalf("创建配电室失败: %d %s", rec.Code, rec.Body.String())
	}

	// 空名称 → 400
	rec, _ = doJSON(t, env, "POST", "/api/rooms", model.RoomInput{Name: "  "})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("空名称应 400，得到 %d", rec.Code)
	}

	// 重名 → 400
	rec, out := doJSON(t, env, "POST", "/api/rooms", model.RoomInput{Name: "一号配电室"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("重名应 400，得到 %d", rec.Code)
	}
	if msg := fmt.Sprint(asMap(out)["error"]); !strings.Contains(msg, "已存在") {
		t.Fatalf("重名错误信息不符: %s", msg)
	}

	// 列表
	rec, out = doJSON(t, env, "GET", "/api/rooms", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("列表失败: %d", rec.Code)
	}
	list, ok := out.([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("期望 1 个配电室，得到 %v", out)
	}
}

func TestEquipmentAPIValidation(t *testing.T) {
	env := newTestEnv(t)
	rec, out := doJSON(t, env, "POST", "/api/rooms", model.RoomInput{Name: "一号配电室"})
	if rec.Code != http.StatusOK {
		t.Fatalf("创建房间失败")
	}
	roomID := asInt(asMap(out)["id"])

	// 创建柜
	rec, out = doJSON(t, env, "POST", "/api/cabinets", model.CabinetInput{RoomID: roomID, Name: "G01"})
	if rec.Code != http.StatusOK {
		t.Fatalf("创建柜失败: %s", rec.Body.String())
	}
	cabID := asInt(asMap(out)["id"])

	// 柜与房间不匹配：柜属于 roomID，但记录挂在另一个房间
	rec, out = doJSON(t, env, "POST", "/api/rooms", model.RoomInput{Name: "二号配电室"})
	room2ID := asInt(asMap(out)["id"])
	rec, _ = doJSON(t, env, "POST", "/api/equipment", model.EquipmentInput{
		RoomID: room2ID, CabinetID: &cabID, Name: "断路器",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("柜与房间不匹配应 400，得到 %d %s", rec.Code, rec.Body.String())
	}

	// 10 张图 → 400（先造 10 个附件）
	var ids []int64
	for i := 0; i < 10; i++ {
		a := model.Annex{
			UUID:         "test-" + strconv.FormatInt(int64(i), 10) + "-" + uuid.NewString(),
			OriginalName: "a.png", Ext: ".png", MimeType: "image/png", Size: 1,
		}
		ins, err := env.store.InsertAnnex(a)
		if err != nil {
			t.Fatalf("插入附件失败: %v", err)
		}
		ids = append(ids, ins.ID)
	}
	rec, _ = doJSON(t, env, "POST", "/api/equipment", model.EquipmentInput{
		RoomID: roomID, Name: "断路器", ImageIDs: ids,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("10 张图应 400，得到 %d %s", rec.Code, rec.Body.String())
	}

	// 数量为负 → 400
	rec, _ = doJSON(t, env, "POST", "/api/equipment", model.EquipmentInput{
		RoomID: roomID, Name: "断路器", Quantity: -1,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("负数数量应 400，得到 %d", rec.Code)
	}

	// 正常创建 + 搜索
	rec, out = doJSON(t, env, "POST", "/api/equipment", model.EquipmentInput{
		RoomID: roomID, CabinetID: &cabID, Name: "断路器", Model: "DZ47-63", Quantity: 2, ImageIDs: ids[:3],
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("正常创建失败: %s", rec.Body.String())
	}
	rec, out = doJSON(t, env, "GET", "/api/equipment?keyword=DZ47", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("搜索失败: %d", rec.Code)
	}
	if total := asInt(asMap(out)["total"]); total != 1 {
		t.Fatalf("搜索期望 1 条，得到 %v", total)
	}
}

func TestUploadAnnex(t *testing.T) {
	env := newTestEnv(t)

	rec, out := uploadAnnex(t, env, "photo.png", tinyPNG)
	if rec.Code != http.StatusOK {
		t.Fatalf("上传失败: %d %s", rec.Code, rec.Body.String())
	}
	id := asInt(out["id"])
	uuidStr := fmt.Sprint(out["uuid"])
	if err := uuid.Validate(uuidStr); err != nil {
		t.Fatalf("uuid 格式不符: %s", uuidStr)
	}

	// 磁盘文件确为 uuid 命名且存在
	entries, err := os.ReadDir(env.annexDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("附件目录应恰好 1 个文件，得到 %d (%v)", len(entries), err)
	}
	if entries[0].Name() != uuidStr+".png" {
		t.Fatalf("文件名应为 <uuid>.png，得到 %s", entries[0].Name())
	}

	// 下载
	req := httptest.NewRequest("GET", "/api/annex/"+strconv.FormatInt(id, 10)+"/file", nil)
	rec2 := httptest.NewRecorder()
	env.engine.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("下载图片失败: %d", rec2.Code)
	}
	if !bytes.Equal(rec2.Body.Bytes(), tinyPNG) {
		t.Fatalf("下载内容不符")
	}

	// 非法扩展名 → 400
	rec, _ = uploadAnnex(t, env, "evil.exe", []byte("MZ"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("非法扩展名应 400，得到 %d", rec.Code)
	}

	// 列表接口：未引用 → ref_count=0、references 为空
	rec, listOut := doJSON(t, env, "GET", "/api/annex", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("附件列表失败: %d", rec.Code)
	}
	items, _ := asMap(listOut)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("期望 1 个附件，得到 %d", len(items))
	}
	first, _ := items[0].(map[string]any)
	if asInt(first["ref_count"]) != 0 {
		t.Fatalf("未引用附件 ref_count 应为 0")
	}
	if refs, ok := first["references"].([]any); ok && len(refs) != 0 {
		t.Fatalf("未引用附件 references 应为空")
	}
}

func TestAnnexListAndRecompute(t *testing.T) {
	env := newTestEnv(t)
	rec, out := doJSON(t, env, "POST", "/api/rooms", model.RoomInput{Name: "一号配电室"})
	if rec.Code != http.StatusOK {
		t.Fatalf("创建房间失败")
	}
	roomID := asInt(asMap(out)["id"])

	// 上传两张图，其中一张被房间引用
	rec, up1 := uploadAnnex(t, env, "referenced.png", tinyPNG)
	if rec.Code != http.StatusOK {
		t.Fatalf("上传失败: %d", rec.Code)
	}
	refID := asInt(up1["id"])
	rec, up2 := uploadAnnex(t, env, "free.png", tinyPNG)
	if rec.Code != http.StatusOK {
		t.Fatalf("上传失败: %d", rec.Code)
	}
	freeID := asInt(up2["id"])

	rec, out = doJSON(t, env, "PUT", "/api/rooms/"+strconv.FormatInt(roomID, 10), model.RoomInput{
		Name: "一号配电室", ImageIDs: []int64{refID},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("更新房间图片失败: %s", rec.Body.String())
	}

	// 重算接口
	rec, _ = doJSON(t, env, "POST", "/api/annex/recompute", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("重算失败: %d", rec.Code)
	}

	// 列表：ref_count 1 与 0；被引用者 references 含配电室
	rec, out = doJSON(t, env, "GET", "/api/annex?page_size=50", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("附件列表失败: %d", rec.Code)
	}
	if total := asInt(asMap(out)["total"]); total != 2 {
		t.Fatalf("期望 2 个附件，得到 %v", asMap(out)["total"])
	}
	byID := map[int64]map[string]any{}
	for _, it := range asMap(out)["items"].([]any) {
		m := it.(map[string]any)
		byID[asInt(m["id"])] = m
	}
	if rc := asInt(byID[refID]["ref_count"]); rc != 1 {
		t.Fatalf("被引用附件 ref_count 应为 1，得到 %d", rc)
	}
	if rc := asInt(byID[freeID]["ref_count"]); rc != 0 {
		t.Fatalf("未引用附件 ref_count 应为 0，得到 %d", rc)
	}
	refs, _ := byID[refID]["references"].([]any)
	if len(refs) != 1 {
		t.Fatalf("被引用附件应有 1 条引用，得到 %d", len(refs))
	}
	if kind := asMap(refs[0])["kind"]; fmt.Sprint(kind) != "room" {
		t.Fatalf("引用类型应为 room，得到 %v", kind)
	}

	// 关键字过滤
	rec, out = doJSON(t, env, "GET", "/api/annex?keyword=free", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("关键字过滤失败: %d", rec.Code)
	}
	if total := asInt(asMap(out)["total"]); total != 1 {
		t.Fatalf("关键字过滤期望 1 个，得到 %v", asMap(out)["total"])
	}
}
