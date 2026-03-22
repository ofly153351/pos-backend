package product

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

type ImageStorage interface {
	SaveProductImage(file *multipart.FileHeader) (string, error)
}

type LocalImageStorage struct {
	baseDir        string
	publicBasePath string
}

func NewLocalImageStorage(baseDir, publicBasePath string) LocalImageStorage {
	return LocalImageStorage{
		baseDir:        baseDir,
		publicBasePath: publicBasePath,
	}
}

func (s LocalImageStorage) SaveProductImage(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", nil
	}

	imageDir := filepath.Join(s.baseDir, "products")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".bin"
	}

	filename := fmt.Sprintf("%s%s", newID(), ext)
	dstPath := filepath.Join(imageDir, filename)

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

	return strings.TrimRight(s.publicBasePath, "/") + "/products/" + filename, nil
}
