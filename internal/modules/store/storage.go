package store

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type LogoStorage interface {
	SaveStoreLogo(file *multipart.FileHeader) (string, error)
}

type LocalLogoStorage struct {
	baseDir        string
	publicBasePath string
}

type MinIOLogoStorage struct {
	client    *minio.Client
	bucket    string
	publicURL string
	once      sync.Once
	onceErr   error
}

func NewLocalLogoStorage(baseDir, publicBasePath string) LocalLogoStorage {
	return LocalLogoStorage{
		baseDir:        baseDir,
		publicBasePath: publicBasePath,
	}
}

func NewMinIOLogoStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicURL string) *MinIOLogoStorage {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		panic(err)
	}

	return &MinIOLogoStorage{
		client:    client,
		bucket:    bucket,
		publicURL: strings.TrimRight(publicURL, "/"),
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

func (s *MinIOLogoStorage) SaveStoreLogo(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", nil
	}
	if err := s.ensureBucket(context.Background()); err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".bin"
	}

	filename := fmt.Sprintf("logos/%s%s", newHexID(), ext)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s.client.PutObject(
		context.Background(),
		s.bucket,
		filename,
		src,
		file.Size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", err
	}

	return s.publicURL + "/" + s.bucket + "/" + filename, nil
}

func (s *MinIOLogoStorage) ensureBucket(ctx context.Context) error {
	s.once.Do(func() {
		exists, err := s.client.BucketExists(ctx, s.bucket)
		if err != nil {
			s.onceErr = err
			return
		}
		if exists {
			return
		}
		s.onceErr = s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
	})
	return s.onceErr
}
