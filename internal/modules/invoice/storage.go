package invoice

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const maxProofFileSize = 10 * 1024 * 1024 // 10MB

type PaymentProofStorage interface {
	SavePaymentProof(file *multipart.FileHeader) (url, mimeType, originalName string, err error)
}

type NoopPaymentProofStorage struct{}

func (NoopPaymentProofStorage) SavePaymentProof(file *multipart.FileHeader) (string, string, string, error) {
	return "", "", "", nil
}

type MinIOPaymentProofStorage struct {
	client    *minio.Client
	bucket    string
	publicURL string
	once      sync.Once
	onceErr   error
}

func NewMinIOPaymentProofStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicURL string) *MinIOPaymentProofStorage {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		panic(err)
	}

	return &MinIOPaymentProofStorage{
		client:    client,
		bucket:    bucket,
		publicURL: strings.TrimRight(publicURL, "/"),
	}
}

func (s *MinIOPaymentProofStorage) SavePaymentProof(file *multipart.FileHeader) (string, string, string, error) {
	if file == nil {
		return "", "", "", nil
	}
	if file.Size <= 0 || file.Size > maxProofFileSize {
		return "", "", "", ErrInvalidProofFileSize
	}

	contentType := strings.TrimSpace(file.Header.Get("Content-Type"))
	if !isAllowedProofContentType(contentType) {
		return "", "", "", ErrInvalidProofFileType
	}
	if err := s.ensureBucket(context.Background()); err != nil {
		return "", "", "", err
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = extByContentType(contentType)
	}
	if ext == "" {
		ext = ".bin"
	}

	objectName := fmt.Sprintf("invoice-payments/%s%s", newID(), ext)
	src, err := file.Open()
	if err != nil {
		return "", "", "", err
	}
	defer src.Close()

	_, err = s.client.PutObject(
		context.Background(),
		s.bucket,
		objectName,
		src,
		file.Size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", "", "", err
	}

	return s.publicURL + "/" + s.bucket + "/" + objectName, contentType, strings.TrimSpace(file.Filename), nil
}

func (s *MinIOPaymentProofStorage) ensureBucket(ctx context.Context) error {
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

func isAllowedProofContentType(contentType string) bool {
	switch strings.ToLower(contentType) {
	case "application/pdf", "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func extByContentType(contentType string) string {
	switch strings.ToLower(contentType) {
	case "application/pdf":
		return ".pdf"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
