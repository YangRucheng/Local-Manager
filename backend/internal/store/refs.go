package store

import (
	"database/sql"
	"fmt"
)

// refTarget 一次引用关系的目标。
type refTarget struct {
	kind string // model.TargetRoom / TargetCabinet / TargetEquipment 的字符串值
	id   int64
}

// replaceRefsTx 在事务内用 annexIDs（按顺序）替换 targetType+targetID 的全部引用，
// 返回受影响（旧 ∪ 新）的 annex id，调用方随后对其重算 ref_count。
func (s *Store) replaceRefsTx(tx *sql.Tx, targetType string, targetID int64, annexIDs []int64) ([]int64, error) {
	old, err := s.annexIDsForTargetQuery(tx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(
		"DELETE FROM annex_ref WHERE target_type = ? AND target_id = ?", targetType, targetID,
	); err != nil {
		return nil, err
	}
	for i, id := range annexIDs {
		if _, err := tx.Exec(
			"INSERT INTO annex_ref(annex_id, target_type, target_id, position, created_at) VALUES(?,?,?,?,?)",
			id, targetType, targetID, i, now(),
		); err != nil {
			return nil, err
		}
	}
	return uniqueInt64(append(old, annexIDs...)), nil
}

// removeRefsTx 在事务内删除目标全部引用，返回被删引用涉及的 annex id（供重算）。
func (s *Store) removeRefsTx(tx *sql.Tx, targetType string, targetID int64) ([]int64, error) {
	ids, err := s.annexIDsForTargetQuery(tx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(
		"DELETE FROM annex_ref WHERE target_type = ? AND target_id = ?", targetType, targetID,
	); err != nil {
		return nil, err
	}
	return ids, nil
}

// removeRefsForTargetsTx 批量删除多个目标的引用，返回全部受影响 annex id。
func (s *Store) removeRefsForTargetsTx(tx *sql.Tx, targets []refTarget) ([]int64, error) {
	var affected []int64
	for _, t := range targets {
		ids, err := s.removeRefsTx(tx, t.kind, t.id)
		if err != nil {
			return nil, err
		}
		affected = append(affected, ids...)
	}
	return uniqueInt64(affected), nil
}

// annexIDsForTargetQuery 返回某目标按 position 排序的 annex id。
func (s *Store) annexIDsForTargetQuery(q interface {
	Query(query string, args ...any) (*sql.Rows, error)
}, targetType string, targetID int64) ([]int64, error) {
	rows, err := q.Query(
		"SELECT annex_id FROM annex_ref WHERE target_type = ? AND target_id = ? ORDER BY position",
		targetType, targetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// recomputeCounts 对指定 annex id 重算 ref_count（传入 旧∪新 集合）。
func recomputeCounts(ex execer, annexIDs []int64) error {
	ids := uniqueInt64(annexIDs)
	if len(ids) == 0 {
		return nil
	}
	args := make([]any, 0, len(ids)+1)
	args = append(args, now())
	for _, id := range ids {
		args = append(args, id)
	}
	_, err := ex.Exec(fmt.Sprintf(
		`UPDATE annex SET ref_count = (SELECT COUNT(*) FROM annex_ref WHERE annex_ref.annex_id = annex.id), updated_at = ?
		 WHERE id IN (%s)`,
		placeholders(len(ids)),
	), args...)
	return err
}

// RecomputeAllCounts 全量重算所有 annex 的引用次数（启动时与手动触发使用）。
func (s *Store) RecomputeAllCounts() error {
	_, err := s.db.Exec(
		`UPDATE annex SET ref_count = (SELECT COUNT(*) FROM annex_ref WHERE annex_ref.annex_id = annex.id), updated_at = ?`,
		now(),
	)
	return err
}
