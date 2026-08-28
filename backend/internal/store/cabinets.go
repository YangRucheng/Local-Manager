package store

import (
	"database/sql"
	"errors"
	"strings"

	"electrical-ledger/internal/model"
)

// ListCabinets 返回配电柜列表；roomID 非 0 时仅返回该配电室下的柜。
func (s *Store) ListCabinets(roomID int64) ([]model.Cabinet, error) {
	query := `SELECT c.id, c.room_id, r.name, c.name, c.remark, c.created_at, c.updated_at
	          FROM cabinets c JOIN rooms r ON r.id = c.room_id`
	args := []any{}
	if roomID != 0 {
		query += " WHERE c.room_id = ?"
		args = append(args, roomID)
	}
	query += " ORDER BY c.id"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Cabinet{}
	for rows.Next() {
		var c model.Cabinet
		if err := rows.Scan(&c.ID, &c.RoomID, &c.RoomName, &c.Name, &c.Remark, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		ids, err := s.annexIDsForTargetQuery(s.db, string(model.TargetCabinet), out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].ImageIDs = ids
	}
	return out, nil
}

// GetCabinet 按 id 查询配电柜。
func (s *Store) GetCabinet(id int64) (model.Cabinet, error) {
	var c model.Cabinet
	err := s.db.QueryRow(
		`SELECT c.id, c.room_id, r.name, c.name, c.remark, c.created_at, c.updated_at
		 FROM cabinets c JOIN rooms r ON r.id = c.room_id WHERE c.id = ?`, id,
	).Scan(&c.ID, &c.RoomID, &c.RoomName, &c.Name, &c.Remark, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Cabinet{}, ErrNotFound
	}
	if err != nil {
		return model.Cabinet{}, err
	}
	ids, err := s.annexIDsForTargetQuery(s.db, string(model.TargetCabinet), c.ID)
	if err != nil {
		return model.Cabinet{}, err
	}
	c.ImageIDs = ids
	return c, nil
}

// CreateCabinet 新建配电柜并写入图片引用。
func (s *Store) CreateCabinet(in model.CabinetInput) (model.Cabinet, error) {
	if err := s.ensureRoomExists(in.RoomID); err != nil {
		return model.Cabinet{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return model.Cabinet{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		"INSERT INTO cabinets(room_id, name, remark, created_at, updated_at) VALUES(?,?,?,?,?)",
		in.RoomID, strings.TrimSpace(in.Name), in.Remark, now(), now(),
	)
	if err != nil {
		return model.Cabinet{}, mapErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Cabinet{}, err
	}
	affected, err := s.replaceRefsTx(tx, string(model.TargetCabinet), id, uniqueInt64(in.ImageIDs))
	if err != nil {
		return model.Cabinet{}, err
	}
	if err := recomputeCounts(tx, affected); err != nil {
		return model.Cabinet{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Cabinet{}, err
	}
	return s.GetCabinet(id)
}

// UpdateCabinet 更新配电柜。
func (s *Store) UpdateCabinet(id int64, in model.CabinetInput) (model.Cabinet, error) {
	if err := s.ensureRoomExists(in.RoomID); err != nil {
		return model.Cabinet{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return model.Cabinet{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		"UPDATE cabinets SET room_id = ?, name = ?, remark = ?, updated_at = ? WHERE id = ?",
		in.RoomID, strings.TrimSpace(in.Name), in.Remark, now(), id,
	)
	if err != nil {
		return model.Cabinet{}, mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.Cabinet{}, ErrNotFound
	}
	affected, err := s.replaceRefsTx(tx, string(model.TargetCabinet), id, uniqueInt64(in.ImageIDs))
	if err != nil {
		return model.Cabinet{}, err
	}
	if err := recomputeCounts(tx, affected); err != nil {
		return model.Cabinet{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Cabinet{}, err
	}
	return s.GetCabinet(id)
}

// DeleteCabinet 删除配电柜：其下台账记录保留但 cabinet_id 置空（外键 SET NULL），
// 仅清理该柜自身的图片引用。
func (s *Store) DeleteCabinet(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	affected, err := s.removeRefsTx(tx, string(model.TargetCabinet), id)
	if err != nil {
		return err
	}
	res, err := tx.Exec("DELETE FROM cabinets WHERE id = ?", id)
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

func (s *Store) ensureRoomExists(roomID int64) error {
	var one int
	err := s.db.QueryRow("SELECT 1 FROM rooms WHERE id = ?", roomID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return invalidf("配电室不存在: %d", roomID)
	}
	return err
}
