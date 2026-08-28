package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"electrical-ledger/internal/model"
)

// ListRooms 返回全部配电室（含柜数与图片），按名称排序。
func (s *Store) ListRooms() ([]model.Room, error) {
	rows, err := s.db.Query(
		`SELECT r.id, r.name, r.remark, r.created_at, r.updated_at,
		        (SELECT COUNT(*) FROM cabinets c WHERE c.room_id = r.id) AS cabinet_count
		 FROM rooms r ORDER BY r.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Room{}
	for rows.Next() {
		var r model.Room
		if err := rows.Scan(&r.ID, &r.Name, &r.Remark, &r.CreatedAt, &r.UpdatedAt, &r.CabinetCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		ids, err := s.annexIDsForTargetQuery(s.db, string(model.TargetRoom), out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].ImageIDs = ids
	}
	return out, nil
}

// GetRoom 按 id 查询配电室。
func (s *Store) GetRoom(id int64) (model.Room, error) {
	var r model.Room
	err := s.db.QueryRow(
		`SELECT r.id, r.name, r.remark, r.created_at, r.updated_at,
		        (SELECT COUNT(*) FROM cabinets c WHERE c.room_id = r.id) AS cabinet_count
		 FROM rooms r WHERE r.id = ?`, id,
	).Scan(&r.ID, &r.Name, &r.Remark, &r.CreatedAt, &r.UpdatedAt, &r.CabinetCount)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Room{}, ErrNotFound
	}
	if err != nil {
		return model.Room{}, err
	}
	ids, err := s.annexIDsForTargetQuery(s.db, string(model.TargetRoom), r.ID)
	if err != nil {
		return model.Room{}, err
	}
	r.ImageIDs = ids
	return r, nil
}

// CreateRoom 新建配电室并写入图片引用。
func (s *Store) CreateRoom(in model.RoomInput) (model.Room, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return model.Room{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		"INSERT INTO rooms(name, remark, created_at, updated_at) VALUES(?,?,?,?)",
		strings.TrimSpace(in.Name), in.Remark, now(), now(),
	)
	if err != nil {
		return model.Room{}, mapErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Room{}, err
	}
	affected, err := s.replaceRefsTx(tx, string(model.TargetRoom), id, uniqueInt64(in.ImageIDs))
	if err != nil {
		return model.Room{}, err
	}
	if err := recomputeCounts(tx, affected); err != nil {
		return model.Room{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Room{}, err
	}
	return s.GetRoom(id)
}

// UpdateRoom 更新配电室名称/备注/图片。
func (s *Store) UpdateRoom(id int64, in model.RoomInput) (model.Room, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return model.Room{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		"UPDATE rooms SET name = ?, remark = ?, updated_at = ? WHERE id = ?",
		strings.TrimSpace(in.Name), in.Remark, now(), id,
	)
	if err != nil {
		return model.Room{}, mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.Room{}, ErrNotFound
	}
	affected, err := s.replaceRefsTx(tx, string(model.TargetRoom), id, uniqueInt64(in.ImageIDs))
	if err != nil {
		return model.Room{}, err
	}
	if err := recomputeCounts(tx, affected); err != nil {
		return model.Room{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Room{}, err
	}
	return s.GetRoom(id)
}

// DeleteRoom 删除配电室：级联删除其配电柜与台账记录，并清理全部引用、重算计数。
// 返回受影响 annex id（已重算），供上层决定是否清理孤儿。
func (s *Store) DeleteRoom(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 收集子目标 id
	var targets []refTarget
	targets = append(targets, refTarget{kind: string(model.TargetRoom), id: id})

	cabRows, err := tx.Query("SELECT id FROM cabinets WHERE room_id = ?", id)
	if err != nil {
		return err
	}
	var cabIDs []int64
	for cabRows.Next() {
		var cid int64
		if err := cabRows.Scan(&cid); err != nil {
			cabRows.Close()
			return err
		}
		cabIDs = append(cabIDs, cid)
	}
	cabRows.Close()
	if err := cabRows.Err(); err != nil {
		return err
	}
	for _, cid := range cabIDs {
		targets = append(targets, refTarget{kind: string(model.TargetCabinet), id: cid})
	}

	eqRows, err := tx.Query("SELECT id FROM equipment WHERE room_id = ?", id)
	if err != nil {
		return err
	}
	var eqIDs []int64
	for eqRows.Next() {
		var eid int64
		if err := eqRows.Scan(&eid); err != nil {
			eqRows.Close()
			return err
		}
		eqIDs = append(eqIDs, eid)
	}
	eqRows.Close()
	if err := eqRows.Err(); err != nil {
		return err
	}
	for _, eid := range eqIDs {
		targets = append(targets, refTarget{kind: string(model.TargetEquipment), id: eid})
	}

	affected, err := s.removeRefsForTargetsTx(tx, targets)
	if err != nil {
		return err
	}
	// 显式删除子记录（外键 CASCADE 兜底，显式删除保证引用清理顺序一致）
	if len(eqIDs) > 0 {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM equipment WHERE id IN (%s)", placeholders(len(eqIDs))), toAny(eqIDs)...); err != nil {
			return err
		}
	}
	if len(cabIDs) > 0 {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM cabinets WHERE id IN (%s)", placeholders(len(cabIDs))), toAny(cabIDs)...); err != nil {
			return err
		}
	}
	res, err := tx.Exec("DELETE FROM rooms WHERE id = ?", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err := recomputeCounts(tx, affected); err != nil {
		return err
	}
	return tx.Commit()
}

func toAny(ids []int64) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}
