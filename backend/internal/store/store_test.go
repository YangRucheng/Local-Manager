package store

import (
	"errors"
	"path/filepath"
	"testing"

	"electrical-ledger/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "app.db"))
	if err != nil {
		t.Fatalf("打开测试库失败: %v", err)
	}
	if err := st.Migrate(); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func mustAnnex(t *testing.T, st *Store, name string) model.Annex {
	t.Helper()
	a, err := st.InsertAnnex(model.Annex{
		UUID:         "uuid-" + name,
		OriginalName: name + ".png",
		Ext:          ".png",
		MimeType:     "image/png",
		Size:         10,
	})
	if err != nil {
		t.Fatalf("插入附件 %s 失败: %v", name, err)
	}
	return a
}

func refCount(t *testing.T, st *Store, id int64) int {
	t.Helper()
	a, err := st.GetAnnex(id)
	if err != nil {
		t.Fatalf("查询附件 %d 失败: %v", id, err)
	}
	return a.RefCount
}

func TestRoomCRUDAndDuplicate(t *testing.T) {
	st := newTestStore(t)

	r1, err := st.CreateRoom(model.RoomInput{Name: "一号配电室", Remark: "一层"})
	if err != nil {
		t.Fatalf("创建配电室失败: %v", err)
	}
	if r1.Name != "一号配电室" {
		t.Fatalf("名称不符: %s", r1.Name)
	}

	if _, err := st.CreateRoom(model.RoomInput{Name: "一号配电室"}); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("期望名称冲突，得到: %v", err)
	}

	r2, err := st.CreateRoom(model.RoomInput{Name: "二号配电室"})
	if err != nil {
		t.Fatalf("创建配电室失败: %v", err)
	}
	rooms, err := st.ListRooms()
	if err != nil {
		t.Fatalf("列出配电室失败: %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("期望 2 个配电室，得到 %d", len(rooms))
	}

	upd, err := st.UpdateRoom(r2.ID, model.RoomInput{Name: "二号配电室（改）", Remark: "二层"})
	if err != nil {
		t.Fatalf("更新配电室失败: %v", err)
	}
	if upd.Remark != "二层" {
		t.Fatalf("备注未更新: %q", upd.Remark)
	}

	if err := st.DeleteRoom(r1.ID); err != nil {
		t.Fatalf("删除配电室失败: %v", err)
	}
	if _, err := st.GetRoom(r1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("删除后应不存在，得到: %v", err)
	}
}

func TestCabinetCRUDAndDeleteKeepsEquipment(t *testing.T) {
	st := newTestStore(t)
	r, _ := st.CreateRoom(model.RoomInput{Name: "一号配电室"})

	c, err := st.CreateCabinet(model.CabinetInput{RoomID: r.ID, Name: "G01"})
	if err != nil {
		t.Fatalf("创建配电柜失败: %v", err)
	}
	if _, err := st.CreateCabinet(model.CabinetInput{RoomID: r.ID, Name: "G01"}); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("期望柜名冲突，得到: %v", err)
	}

	eq, err := st.CreateEquipment(model.EquipmentInput{
		RoomID: r.ID, CabinetID: &c.ID, Name: "断路器", Model: "DZ47-63",
	})
	if err != nil {
		t.Fatalf("创建台账失败: %v", err)
	}

	// 删除柜：记录保留、cabinet_id 置空
	if err := st.DeleteCabinet(c.ID); err != nil {
		t.Fatalf("删除配电柜失败: %v", err)
	}
	got, err := st.GetEquipment(eq.ID)
	if err != nil {
		t.Fatalf("删除柜后记录应保留: %v", err)
	}
	if got.CabinetID != nil {
		t.Fatalf("删除柜后 cabinet_id 应置空，得到 %v", *got.CabinetID)
	}
}

func TestEquipmentFilterSearchPagination(t *testing.T) {
	st := newTestStore(t)
	r, _ := st.CreateRoom(model.RoomInput{Name: "一号配电室"})
	r2, _ := st.CreateRoom(model.RoomInput{Name: "二号配电室"})
	c, _ := st.CreateCabinet(model.CabinetInput{RoomID: r.ID, Name: "G01"})

	// 10 条：5 条在 r/G01，5 条在 r2
	for i := 0; i < 5; i++ {
		_, err := st.CreateEquipment(model.EquipmentInput{
			RoomID: r.ID, CabinetID: &c.ID, Name: "断路器A", Model: "DZ47", Manufacturer: "施耐德", Quantity: i,
		})
		if err != nil {
			t.Fatalf("创建台账失败: %v", err)
		}
		_, err = st.CreateEquipment(model.EquipmentInput{
			RoomID: r2.ID, Name: "交流接触器B", Model: "CJX2", Quantity: 1,
		})
		if err != nil {
			t.Fatalf("创建台账失败: %v", err)
		}
	}

	// 房间过滤
	res, err := st.ListEquipment(EquipmentFilter{RoomID: r.ID, PageSize: 50})
	if err != nil {
		t.Fatalf("过滤失败: %v", err)
	}
	if res.Total != 5 {
		t.Fatalf("房间过滤期望 5，得到 %d", res.Total)
	}

	// 柜过滤
	res, _ = st.ListEquipment(EquipmentFilter{RoomID: r.ID, CabinetID: c.ID, PageSize: 50})
	if res.Total != 5 {
		t.Fatalf("柜过滤期望 5，得到 %d", res.Total)
	}

	// 关键词搜索名称
	res, _ = st.ListEquipment(EquipmentFilter{Keyword: "接触器", PageSize: 50})
	if res.Total != 5 {
		t.Fatalf("名称搜索期望 5，得到 %d", res.Total)
	}

	// 关键词搜索型号
	res, _ = st.ListEquipment(EquipmentFilter{Keyword: "cjx", PageSize: 50})
	if res.Total != 5 {
		t.Fatalf("型号搜索期望 5，得到 %d", res.Total)
	}

	// 关键词含通配符不放大匹配
	res, _ = st.ListEquipment(EquipmentFilter{Keyword: "%", PageSize: 50})
	if res.Total != 0 {
		t.Fatalf("通配符搜索期望 0，得到 %d", res.Total)
	}

	// 分页
	res, _ = st.ListEquipment(EquipmentFilter{Page: 2, PageSize: 3})
	if len(res.Items) != 3 || res.Total != 10 {
		t.Fatalf("分页期望第2页3条/共10条，得到 %d/%d", len(res.Items), res.Total)
	}

	// 房间名/柜名已 JOIN 出来
	if res.Items[0].RoomName == "" {
		t.Fatalf("缺少房间名")
	}
}

func TestAnnexRefCountLifecycle(t *testing.T) {
	st := newTestStore(t)

	a1 := mustAnnex(t, st, "a1")
	a2 := mustAnnex(t, st, "a2")
	a3 := mustAnnex(t, st, "a3")
	a4 := mustAnnex(t, st, "a4")

	// 房间引用 a1,a2 → 各自 ref_count=1
	r, err := st.CreateRoom(model.RoomInput{Name: "一号配电室", ImageIDs: []int64{a1.ID, a2.ID}})
	if err != nil {
		t.Fatalf("创建房间失败: %v", err)
	}
	if refCount(t, st, a1.ID) != 1 || refCount(t, st, a2.ID) != 1 {
		t.Fatalf("房间引用后计数错误: a1=%d a2=%d", refCount(t, st, a1.ID), refCount(t, st, a2.ID))
	}

	// 柜引用 a2（共享）→ a2=2
	c, err := st.CreateCabinet(model.CabinetInput{RoomID: r.ID, Name: "G01", ImageIDs: []int64{a2.ID}})
	if err != nil {
		t.Fatalf("创建柜失败: %v", err)
	}
	if refCount(t, st, a2.ID) != 2 {
		t.Fatalf("共享后 a2 计数错误: %d", refCount(t, st, a2.ID))
	}

	// 台账引用 a1,a3,a4 → a1=2, a3=1, a4=1
	cid := c.ID
	eq, err := st.CreateEquipment(model.EquipmentInput{
		RoomID: r.ID, CabinetID: &cid, Name: "断路器", ImageIDs: []int64{a1.ID, a3.ID, a4.ID},
	})
	if err != nil {
		t.Fatalf("创建台账失败: %v", err)
	}
	if refCount(t, st, a1.ID) != 2 || refCount(t, st, a3.ID) != 1 || refCount(t, st, a4.ID) != 1 {
		t.Fatalf("台账引用后计数错误: a1=%d a3=%d a4=%d", refCount(t, st, a1.ID), refCount(t, st, a3.ID), refCount(t, st, a4.ID))
	}

	// 更新台账移除 a3 → a3=0 成为孤儿
	cid2 := c.ID
	if _, err := st.UpdateEquipment(eq.ID, model.EquipmentInput{
		RoomID: r.ID, CabinetID: &cid2, Name: "断路器", ImageIDs: []int64{a1.ID, a4.ID},
	}); err != nil {
		t.Fatalf("更新台账失败: %v", err)
	}
	if refCount(t, st, a3.ID) != 0 {
		t.Fatalf("移除后 a3 计数应为 0，得到 %d", refCount(t, st, a3.ID))
	}

	// 删除台账 → a1,a4 各减 1 → a1=1,a4=0
	if err := st.DeleteEquipment(eq.ID); err != nil {
		t.Fatalf("删除台账失败: %v", err)
	}
	if refCount(t, st, a1.ID) != 1 || refCount(t, st, a4.ID) != 0 {
		t.Fatalf("删除台账后计数错误: a1=%d a4=%d", refCount(t, st, a1.ID), refCount(t, st, a4.ID))
	}

	// 全量重算后数值不变
	if err := st.RecomputeAllCounts(); err != nil {
		t.Fatalf("重算失败: %v", err)
	}
	if refCount(t, st, a1.ID) != 1 || refCount(t, st, a2.ID) != 2 || refCount(t, st, a3.ID) != 0 {
		t.Fatalf("重算后计数错误")
	}

	// 附件列表：总数 4；a2 被房间+柜引用（2 处），a3 无引用
	res, err := st.ListAnnexes(AnnexFilter{PageSize: 50})
	if err != nil {
		t.Fatalf("列出附件失败: %v", err)
	}
	if res.Total != 4 {
		t.Fatalf("期望 4 个附件，得到 %d", res.Total)
	}
	refs, err := st.AnnexReferences(a2.ID)
	if err != nil {
		t.Fatalf("查询附件引用失败: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("a2 应被 2 处引用，得到 %d", len(refs))
	}
	refsA3, err := st.AnnexReferences(a3.ID)
	if err != nil {
		t.Fatalf("查询附件引用失败: %v", err)
	}
	if len(refsA3) != 0 {
		t.Fatalf("a3 应无引用，得到 %d", len(refsA3))
	}
	// 关键字过滤（按文件名）
	res, err = st.ListAnnexes(AnnexFilter{Keyword: "a3", PageSize: 50})
	if err != nil {
		t.Fatalf("关键字过滤失败: %v", err)
	}
	if res.Total != 1 {
		t.Fatalf("关键字过滤期望 1 个，得到 %d", res.Total)
	}
}

func TestDeleteRoomCascades(t *testing.T) {
	st := newTestStore(t)
	a1 := mustAnnex(t, st, "a1")
	a2 := mustAnnex(t, st, "a2")
	a3 := mustAnnex(t, st, "a3")

	r, _ := st.CreateRoom(model.RoomInput{Name: "一号配电室", ImageIDs: []int64{a1.ID}})
	c, _ := st.CreateCabinet(model.CabinetInput{RoomID: r.ID, Name: "G01", ImageIDs: []int64{a2.ID}})
	cid := c.ID
	eq, err := st.CreateEquipment(model.EquipmentInput{RoomID: r.ID, CabinetID: &cid, Name: "断路器", ImageIDs: []int64{a3.ID}})
	if err != nil {
		t.Fatalf("创建台账失败: %v", err)
	}

	if err := st.DeleteRoom(r.ID); err != nil {
		t.Fatalf("删除房间失败: %v", err)
	}

	// 全部子记录与引用被清理，附件成为孤儿
	if _, err := st.GetRoom(r.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("房间应已删除")
	}
	if _, err := st.GetCabinet(c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("柜应已删除")
	}
	if _, err := st.GetEquipment(eq.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("台账应已删除")
	}
	for _, id := range []int64{a1.ID, a2.ID, a3.ID} {
		if refCount(t, st, id) != 0 {
			t.Fatalf("删除房间后附件 %d 应无引用", id)
		}
	}
}
