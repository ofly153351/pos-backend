package storage

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

	"pos-backend/internal/idgen"
)

type LogoStorage interface {
	SaveLogo(file *multipart.FileHeader) (string, error)
	DeleteLogo(logoURL string) error
}

// ── Local ────────────────────────────────────────────────────────────────────

type LocalLogoStorage struct {
	baseDir        string
	publicBasePath string
	subDir         string
}

func NewLocalLogoStorage(baseDir, publicBasePath, subDir string) LocalLogoStorage {
	return LocalLogoStorage{
		baseDir:        baseDir,
		publicBasePath: strings.TrimRight(publicBasePath, "/"),
		subDir:         subDir,
	}
}

func (s LocalLogoStorage) SaveLogo(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", nil
	}
	dir := filepath.Join(s.baseDir, s.subDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".bin"
	}
	filename := fmt.Sprintf("%08d%s", idgen.NextInt(), ext)
	dst, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return "", err
	}
	defer dst.Close()
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return s.publicBasePath + "/" + s.subDir + "/" + filename, nil
}

func (s LocalLogoStorage) DeleteLogo(logoURL string) error {
	objectPath, ok := safeAssetPath(logoURL, s.publicBasePath, s.subDir)
	if !ok {
		return nil
	}
	dir := filepath.Join(s.baseDir, s.subDir)
	filePath := filepath.Join(dir, filepath.Base(objectPath))
	rel, err := filepath.Rel(dir, filePath)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return nil
	}
	if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// ── MinIO ────────────────────────────────────────────────────────────────────

type MinIOLogoStorage struct {
	client    *minio.Client
	bucket    string
	publicURL string
	subDir    string

	// ensureBucket bookkeeping. Only a successful bucket check is cached; a
	// failure (e.g. MinIO down when the API started) is retried on every call.
	mu       sync.Mutex
	verified bool
}

func NewMinIOLogoStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicURL, subDir string) *MinIOLogoStorage {
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
		subDir:    subDir,
	}
}

func (s *MinIOLogoStorage) SaveLogo(file *multipart.FileHeader) (string, error) {
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
	objectName := fmt.Sprintf("%s/%08d%s", s.subDir, idgen.NextInt(), ext)
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
		context.Background(), s.bucket, objectName, src, file.Size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", err
	}
	return s.publicURL + "/" + s.bucket + "/" + objectName, nil
}

func (s *MinIOLogoStorage) DeleteLogo(logoURL string) error {
	objectPath, ok := safeAssetPath(logoURL, s.publicURL+"/"+s.bucket, s.subDir)
	if !ok {
		return nil
	}
	return s.client.RemoveObject(context.Background(), s.bucket, objectPath, minio.RemoveObjectOptions{})
}

// ensureBucket lazily creates the bucket on first use. A failure is NOT cached:
// MinIO may come up after the API server starts (docker compose up order), so a
// once-failed attempt must retry on the next call. Only success is remembered.
func (s *MinIOLogoStorage) ensureBucket(ctx context.Context) error {
	s.mu.Lock()
	verified := s.verified
	s.mu.Unlock()
	if verified {
		return nil
	}
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err // transient (MinIO down?) — retry on next call
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return err // transient — retry on next call
		}
	}
	s.mu.Lock()
	s.verified = true
	s.mu.Unlock()
	return nil
}

// ── safeAssetPath ─────────────────────────────────────────────────────────────
// Returns the relative object path (subDir/filename) if logoURL is a valid
// asset under publicBasePath/subDir, otherwise (_, false).

func safeAssetPath(rawURL, publicBasePath, subDir string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	publicBasePath = strings.TrimRight(strings.TrimSpace(publicBasePath), "/")
	if rawURL == "" || publicBasePath == "" || subDir == "" {
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
		if !assetURL.IsAbs() || !baseURL.IsAbs() ||
			!strings.EqualFold(assetURL.Scheme, baseURL.Scheme) ||
			!strings.EqualFold(assetURL.Host, baseURL.Host) {
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
	if len(parts) != 2 || parts[0] != subDir {
		return "", false
	}
	name := parts[1]
	if name == "" || name == "." || name == ".." ||
		strings.HasPrefix(strings.ToLower(name), "default") ||
		strings.HasPrefix(strings.ToLower(name), "shared") {
		return "", false
	}
	return parts[0] + "/" + name, true
}
