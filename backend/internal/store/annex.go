package store

import (
	"database/sql"
	"errors"
	"fmt"

	"electrical-ledger/internal/model"
)

// InsertAnnex 插入一条附件记录，返回带 ID 的完整记录。
func (s *Store) InsertAnnex(a model.Annex) (model.Annex, error) {
	res, err := s.db.Exec(
		`INSERT INTO annex(uuid, original_name, ext, mime_type, size, ref_count, created_at, updated_at)
		 VALUES(?,?,?,?,?,0,?,?)`,
		a.UUID, a.OriginalName, a.Ext, a.MimeType, a.Size, now(), now(),
	)
	if err != nil {
		return model.Annex{}, mapErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Annex{}, err
	}
	a.ID = id
	a.CreatedAt, a.UpdatedAt = now(), now()
	return a, nil
}

// GetAnnex 按 id 查询附件。
func (s *Store) GetAnnex(id int64) (model.Annex, error) {
	var a model.Annex
	err := s.db.QueryRow(
		`SELECT id, uuid, original_name, ext, mime_type, size, ref_count, created_at, updated_at
		 FROM annex WHERE id = ?`, id,
	).Scan(&a.ID, &a.UUID, &a.OriginalName, &a.Ext, &a.MimeType, &a.Size, &a.RefCount, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Annex{}, ErrNotFound
	}
	if err != nil {
		return model.Annex{}, err
	}
	return a, nil
}

// ListOrphanAnnexes 返回当前 ref_count = 0 的附件（建议先 RecomputeAllCounts 再调用）。
func (s *Store) ListOrphanAnnexes() ([]model.Annex, error) {
	rows, err := s.db.Query(
		`SELECT id, uuid, original_name, ext, mime_type, size, ref_count, created_at, updated_at
		 FROM annex WHERE ref_count = 0 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Annex
	for rows.Next() {
		var a model.Annex
		if err := rows.Scan(&a.ID, &a.UUID, &a.OriginalName, &a.Ext, &a.MimeType, &a.Size, &a.RefCount, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DeleteAnnex 删除附件记录（磁盘文件由 annex.Service 负责）。
func (s *Store) DeleteAnnex(id int64) error {
	res, err := s.db.Exec("DELETE FROM annex WHERE id = ?", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// AnnexIDsExist 返回不存在的附件 id 列表。
func (s *Store) AnnexIDsExist(ids []int64) ([]int64, error) {
	ids = uniqueInt64(ids)
	if len(ids) == 0 {
		return nil, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.db.Query(
		fmt.Sprintf("SELECT id FROM annex WHERE id IN (%s)", placeholders(len(ids))), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	exist := make(map[int64]struct{}, len(ids))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		exist[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var missing []int64
	for _, id := range ids {
		if _, ok := exist[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing, nil
}
