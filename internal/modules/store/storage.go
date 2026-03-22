package store

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

type LogoStorage interface {
	SaveStoreLogo(file *multipart.FileHeader) (string, error)
}

type LocalLogoStorage struct {
	baseDir        string
	publicBasePath string
}

func NewLocalLogoStorage(baseDir, publicBasePath string) LocalLogoStorage {
	return LocalLogoStorage{
		baseDir:        baseDir,
		publicBasePath: publicBasePath,
	}
}

func (s LocalLogoStorage) SaveStoreLogo(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", nil
	}

	logoDir := filepath.Join(s.baseDir, "logos")
	if err := os.MkdirAll(logoDir, 0o755); err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".bin"
	}

	filename := fmt.Sprintf("%s%s", newHexID(), ext)
	dstPath := filepath.Join(logoDir, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return strings.TrimRight(s.publicBasePath, "/") + "/logos/" + filename, nil
}
