// Package annex 负责附件的磁盘存储：按 uuid 重命名落盘、删除、路径解析。
package annex

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"electrical-ledger/internal/model"
)

// MaxFileSize 单张图片上限。
const MaxFileSize = 10 << 20 // 10 MB

// allowedExt 允许的图片扩展名 → MIME 类型。
var allowedExt = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
}

// 领域错误。
var (
	ErrFileTooLarge = errors.New("图片大小不能超过 10MB")
	ErrBadExt       = errors.New("仅支持 JPG/PNG/WebP/GIF 图片")
)

// Service 附件磁盘服务。
type Service struct {
	AnnexDir string
}

// SaveFile 将上传文件以 uuid 重命名保存到 AnnexDir，返回 (uuid, ext)。
func (s *Service) SaveFile(header *multipart.FileHeader) (uid, ext string, err error) {
	if header.Size > MaxFileSize {
		return "", "", ErrFileTooLarge
	}
	ext = strings.ToLower(filepath.Ext(header.Filename))
	if _, ok := allowedExt[ext]; !ok {
		return "", "", ErrBadExt
	}
	src, err := header.Open()
	if err != nil {
		return "", "", fmt.Errorf("打开上传文件: %w", err)
	}
	defer src.Close()

	uid = uuid.NewString()
	fileName := uid + ext
	if err := os.MkdirAll(s.AnnexDir, 0o755); err != nil {
		return "", "", fmt.Errorf("创建附件目录: %w", err)
	}
	dst, err := os.Create(filepath.Join(s.AnnexDir, fileName))
	if err != nil {
		return "", "", fmt.Errorf("创建附件文件: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", "", fmt.Errorf("写入附件文件: %w", err)
	}
	return uid, ext, nil
}

// FilePath 返回附件对应的磁盘绝对/相对路径。
func (s *Service) FilePath(a model.Annex) string {
	return filepath.Join(s.AnnexDir, a.UUID+a.Ext)
}

// MimeFor 返回扩展名对应的 MIME（未知返回空串）。
func MimeFor(ext string) string {
	return allowedExt[strings.ToLower(ext)]
}

// DeleteFile 删除磁盘文件，文件不存在视为成功。
func (s *Service) DeleteFile(a model.Annex) error {
	err := os.Remove(s.FilePath(a))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
