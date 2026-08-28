package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

// AnnexFilter 附件列表过滤条件。
type AnnexFilter struct {
	Keyword  string
	Page     int
	PageSize int
}

// ListAnnexes 分页查询附件（可按文件名模糊搜索，按 id 倒序）。
func (s *Store) ListAnnexes(f AnnexFilter) (model.AnnexListResult, error) {
	where := ""
	args := []any{}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		where = " WHERE original_name LIKE ? ESCAPE '\\'"
		args = append(args, "%"+escapeLike(kw)+"%")
	}
	var total int64
	if err := s.db.QueryRow("SELECT COUNT(*) FROM annex"+where, args...).Scan(&total); err != nil {
		return model.AnnexListResult{}, err
	}

	page, size := normalizePage(f.Page, f.PageSize)
	query := `SELECT id, uuid, original_name, ext, mime_type, size, ref_count, created_at, updated_at
	          FROM annex` + where + " ORDER BY id DESC LIMIT ? OFFSET ?"
	queryArgs := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return model.AnnexListResult{}, err
	}
	defer rows.Close()

	items := make([]model.Annex, 0, size)
	for rows.Next() {
		var a model.Annex
		if err := rows.Scan(&a.ID, &a.UUID, &a.OriginalName, &a.Ext, &a.MimeType, &a.Size, &a.RefCount, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return model.AnnexListResult{}, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return model.AnnexListResult{}, err
	}
	for i := range items {
		refs, err := s.AnnexReferences(items[i].ID)
		if err != nil {
			return model.AnnexListResult{}, err
		}
		items[i].References = refs
	}
	return model.AnnexListResult{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// AnnexReferences 返回某附件被引用的目标列表（按 position 排序）。
func (s *Store) AnnexReferences(annexID int64) ([]model.AnnexRefInfo, error) {
	rows, err := s.db.Query(
		`SELECT r.target_type, r.target_id,
		        CASE r.target_type
		          WHEN 'room'      THEN (SELECT name FROM rooms WHERE id = r.target_id)
		          WHEN 'cabinet'   THEN (SELECT name FROM cabinets WHERE id = r.target_id)
		          WHEN 'equipment' THEN (SELECT name FROM equipment WHERE id = r.target_id)
		          ELSE ''
		        END
		 FROM annex_ref r WHERE r.annex_id = ? ORDER BY r.position`,
		annexID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AnnexRefInfo
	for rows.Next() {
		var ref model.AnnexRefInfo
		if err := rows.Scan(&ref.Kind, &ref.ID, &ref.Name); err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, rows.Err()
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
