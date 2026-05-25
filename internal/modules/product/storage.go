package product

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ImageStorage interface {
	SaveProductImage(file *multipart.FileHeader) (string, error)
	DeleteProductImage(imageURL string) error
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

func (s LocalImageStorage) DeleteProductImage(imageURL string) error {
	objectPath, ok := safeAssetPath(imageURL, s.publicBasePath, "products")
	if !ok {
		return nil
	}
	productsDir := filepath.Join(s.baseDir, "products")
	filePath := filepath.Join(productsDir, filepath.Base(objectPath))
	rel, err := filepath.Rel(productsDir, filePath)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return nil
	}
	if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *MinIOImageStorage) DeleteProductImage(imageURL string) error {
	objectPath, ok := safeAssetPath(imageURL, s.publicURL+"/"+s.bucket, "products")
	if !ok {
		return nil
	}
	return s.client.RemoveObject(context.Background(), s.bucket, objectPath, minio.RemoveObjectOptions{})
}

func safeAssetPath(rawURL, publicBasePath, assetDir string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	publicBasePath = strings.TrimRight(strings.TrimSpace(publicBasePath), "/")
	if rawURL == "" || publicBasePath == "" || assetDir == "" {
		return "", false
	}

	assetURL, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	baseURL, err := url.Parse(publicBasePath)
	if err != nil {
		return "", false
	}

	assetPath := assetURL.EscapedPath()
	if assetPath == "" {
		assetPath = rawURL
	}
	if unescaped, err := url.PathUnescape(assetPath); err == nil {
		assetPath = unescaped
	}
	basePath := baseURL.EscapedPath()
	if basePath == "" {
		basePath = publicBasePath
	}
	if unescaped, err := url.PathUnescape(basePath); err == nil {
		basePath = unescaped
	}

	if assetURL.IsAbs() || baseURL.IsAbs() {
		if !assetURL.IsAbs() || !baseURL.IsAbs() || !strings.EqualFold(assetURL.Scheme, baseURL.Scheme) || !strings.EqualFold(assetURL.Host, baseURL.Host) {
			return "", false
		}
	}

	assetPath = path.Clean("/" + strings.TrimLeft(assetPath, "/"))
	basePath = path.Clean("/" + strings.TrimLeft(basePath, "/"))
	rel := strings.TrimPrefix(assetPath, basePath)
	if rel == assetPath || !strings.HasPrefix(rel, "/") {
		return "", false
	}
	rel = strings.TrimPrefix(path.Clean(rel), "/")
	parts := strings.Split(rel, "/")
	if len(parts) != 2 || parts[0] != assetDir {
		return "", false
	}
	name := parts[1]
	if name == "" || name == "." || name == ".." || strings.HasPrefix(strings.ToLower(name), "default") || strings.HasPrefix(strings.ToLower(name), "shared") {
		return "", false
	}
	return parts[0] + "/" + name, true
}
