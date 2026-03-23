package product

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

type ImageStorage interface {
	SaveProductImage(file *multipart.FileHeader) (string, error)
}

type LocalImageStorage struct {
	baseDir        string
	publicBasePath string
}

type MinIOImageStorage struct {
	client    *minio.Client
	bucket    string
	publicURL string
	once      sync.Once
	onceErr   error
}

func NewLocalImageStorage(baseDir, publicBasePath string) LocalImageStorage {
	return LocalImageStorage{
		baseDir:        baseDir,
		publicBasePath: publicBasePath,
	}
}

func NewMinIOImageStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicURL string) *MinIOImageStorage {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		panic(err)
	}

	return &MinIOImageStorage{
		client:    client,
		bucket:    bucket,
		publicURL: strings.TrimRight(publicURL, "/"),
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

func (s *MinIOImageStorage) SaveProductImage(file *multipart.FileHeader) (string, error) {
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

	filename := fmt.Sprintf("products/%s%s", newID(), ext)

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

func (s *MinIOImageStorage) ensureBucket(ctx context.Context) error {
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
