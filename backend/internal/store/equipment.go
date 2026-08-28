package store

import (
	"database/sql"
	"errors"
	"strings"

	"electrical-ledger/internal/model"
)

// EquipmentFilter 台账列表过滤条件。
type EquipmentFilter struct {
	RoomID    int64
	CabinetID int64 // 0 表示不过滤
	Keyword   string
	Page      int
	PageSize  int
}

// ListEquipment 按条件查询台账记录（JOIN 出房间/柜名，附带图片 id），返回分页结果。
func (s *Store) ListEquipment(f EquipmentFilter) (model.EquipmentListResult, error) {
	where, args := equipmentWhere(f)
	countQuery := "SELECT COUNT(*) FROM equipment e" + where
	var total int64
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return model.EquipmentListResult{}, err
	}

	page, size := normalizePage(f.Page, f.PageSize)
	query := `SELECT e.id, e.room_id, r.name, e.cabinet_id, COALESCE(c.name,''),
	                 e.name, e.model, e.manufacturer, e.quantity, e.remark,
	                 e.created_at, e.updated_at
	          FROM equipment e
	          JOIN rooms r ON r.id = e.room_id
	          LEFT JOIN cabinets c ON c.id = e.cabinet_id` + where +
		" ORDER BY e.id DESC LIMIT ? OFFSET ?"
	queryArgs := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return model.EquipmentListResult{}, err
	}
	defer rows.Close()

	items := make([]model.Equipment, 0, size)
	for rows.Next() {
		var e model.Equipment
		if err := rows.Scan(
			&e.ID, &e.RoomID, &e.RoomName, &e.CabinetID, &e.CabinetName,
			&e.Name, &e.Model, &e.Manufacturer, &e.Quantity, &e.Remark,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return model.EquipmentListResult{}, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return model.EquipmentListResult{}, err
	}
	for i := range items {
		ids, err := s.annexIDsForTargetQuery(s.db, string(model.TargetEquipment), items[i].ID)
		if err != nil {
			return model.EquipmentListResult{}, err
		}
		items[i].ImageIDs = ids
	}
	return model.EquipmentListResult{Items: items, Total: total, Page: page, PageSize: size}, nil
}

func equipmentWhere(f EquipmentFilter) (string, []any) {
	var conds []string
	var args []any
	if f.RoomID != 0 {
		conds = append(conds, "e.room_id = ?")
		args = append(args, f.RoomID)
	}
	if f.CabinetID != 0 {
		conds = append(conds, "e.cabinet_id = ?")
		args = append(args, f.CabinetID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + escapeLike(kw) + "%"
		conds = append(conds, "(e.name LIKE ? ESCAPE '\\' OR e.model LIKE ? ESCAPE '\\')")
		args = append(args, like, like)
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// escapeLike 转义 LIKE 通配符。
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	return page, size
}

// GetEquipment 按 id 查询单条台账。
func (s *Store) GetEquipment(id int64) (model.Equipment, error) {
	var e model.Equipment
	err := s.db.QueryRow(
		`SELECT e.id, e.room_id, r.name, e.cabinet_id, COALESCE(c.name,''),
		        e.name, e.model, e.manufacturer, e.quantity, e.remark,
		        e.created_at, e.updated_at
		 FROM equipment e
		 JOIN rooms r ON r.id = e.room_id
		 LEFT JOIN cabinets c ON c.id = e.cabinet_id
		 WHERE e.id = ?`, id,
	).Scan(
		&e.ID, &e.RoomID, &e.RoomName, &e.CabinetID, &e.CabinetName,
		&e.Name, &e.Model, &e.Manufacturer, &e.Quantity, &e.Remark,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Equipment{}, ErrNotFound
	}
	if err != nil {
		return model.Equipment{}, err
	}
	ids, err := s.annexIDsForTargetQuery(s.db, string(model.TargetEquipment), e.ID)
	if err != nil {
		return model.Equipment{}, err
	}
	e.ImageIDs = ids
	return e, nil
}

// CreateEquipment 新建台账记录并写入图片引用。
func (s *Store) CreateEquipment(in model.EquipmentInput) (model.Equipment, error) {
	if err := s.validateEquipmentInput(in); err != nil {
		return model.Equipment{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return model.Equipment{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		`INSERT INTO equipment(room_id, cabinet_id, name, model, manufacturer, quantity, remark, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		in.RoomID, in.CabinetID, strings.TrimSpace(in.Name), strings.TrimSpace(in.Model),
		strings.TrimSpace(in.Manufacturer), in.Quantity, in.Remark, now(), now(),
	)
	if err != nil {
		return model.Equipment{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Equipment{}, err
	}
	affected, err := s.replaceRefsTx(tx, string(model.TargetEquipment), id, uniqueInt64(in.ImageIDs))
	if err != nil {
		return model.Equipment{}, err
	}
	if err := recomputeCounts(tx, affected); err != nil {
		return model.Equipment{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Equipment{}, err
	}
	return s.GetEquipment(id)
}

// UpdateEquipment 更新台账记录并替换图片引用。
func (s *Store) UpdateEquipment(id int64, in model.EquipmentInput) (model.Equipment, error) {
	if err := s.validateEquipmentInput(in); err != nil {
		return model.Equipment{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return model.Equipment{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		`UPDATE equipment SET room_id = ?, cabinet_id = ?, name = ?, model = ?, manufacturer = ?,
		        quantity = ?, remark = ?, updated_at = ? WHERE id = ?`,
		in.RoomID, in.CabinetID, strings.TrimSpace(in.Name), strings.TrimSpace(in.Model),
		strings.TrimSpace(in.Manufacturer), in.Quantity, in.Remark, now(), id,
	)
	if err != nil {
		return model.Equipment{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.Equipment{}, ErrNotFound
	}
	affected, err := s.replaceRefsTx(tx, string(model.TargetEquipment), id, uniqueInt64(in.ImageIDs))
	if err != nil {
		return model.Equipment{}, err
	}
	if err := recomputeCounts(tx, affected); err != nil {
		return model.Equipment{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Equipment{}, err
	}
	return s.GetEquipment(id)
}

// DeleteEquipment 删除台账记录并清理其图片引用。
func (s *Store) DeleteEquipment(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	affected, err := s.removeRefsTx(tx, string(model.TargetEquipment), id)
	if err != nil {
		return err
	}
	res, err := tx.Exec("DELETE FROM equipment WHERE id = ?", id)
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

// validateEquipmentInput 校验房间/柜存在且一致、数量非负。
func (s *Store) validateEquipmentInput(in model.EquipmentInput) error {
	if err := s.ensureRoomExists(in.RoomID); err != nil {
		return err
	}
	if in.CabinetID != nil {
		var owner int64
		err := s.db.QueryRow("SELECT room_id FROM cabinets WHERE id = ?", *in.CabinetID).Scan(&owner)
		if errors.Is(err, sql.ErrNoRows) {
			return invalidf("配电柜不存在: %d", *in.CabinetID)
		}
		if err != nil {
			return err
		}
		if owner != in.RoomID {
			return invalidf("配电柜与配电室不匹配")
		}
	}
	if in.Quantity < 0 {
		return invalidf("数量不能为负数")
	}
	return nil
}
