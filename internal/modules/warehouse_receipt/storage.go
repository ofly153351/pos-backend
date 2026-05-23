package warehouse_receipt

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type AttachmentStorage interface {
	SaveAttachment(file *multipart.FileHeader) (url, mimeType, originalName string, size int64, err error)
}

type NoopAttachmentStorage struct{}

func (NoopAttachmentStorage) SaveAttachment(file *multipart.FileHeader) (string, string, string, int64, error) {
	return "", "", "", 0, ErrReceiptAttachmentStorage
}

type MinIOAttachmentStorage struct {
	client    *minio.Client
	bucket    string
	publicURL string
	once      sync.Once
	onceErr   error
}

func NewMinIOAttachmentStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicURL string) *MinIOAttachmentStorage {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		panic(err)
	}
	return &MinIOAttachmentStorage{client: client, bucket: bucket, publicURL: strings.TrimRight(publicURL, "/")}
}

func (s *MinIOAttachmentStorage) SaveAttachment(file *multipart.FileHeader) (string, string, string, int64, error) {
	if file == nil {
		return "", "", "", 0, ErrReceiptAttachmentRequired
	}
	if file.Size <= 0 || file.Size > attachmentMaxBytes {
		return "", "", "", 0, ErrReceiptAttachmentSize
	}
	contentType := strings.ToLower(strings.TrimSpace(file.Header.Get("Content-Type")))
	ext := normalizeExt(file.Filename, contentType)
	if ext == "" {
		return "", "", "", 0, ErrReceiptAttachmentType
	}
	if contentType == "" || contentType == "application/octet-stream" {
		switch ext {
		case ".pdf":
			contentType = "application/pdf"
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		default:
			return "", "", "", 0, ErrReceiptAttachmentType
		}
	}
	if contentType != "application/pdf" && contentType != "image/jpeg" && contentType != "image/png" {
		return "", "", "", 0, ErrReceiptAttachmentType
	}
	if err := s.ensureBucket(context.Background()); err != nil {
		return "", "", "", 0, err
	}
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	objectName := fmt.Sprintf("warehouse-receipts/%s%s", newAttachmentToken(), ext)
	src, err := file.Open()
	if err != nil {
		return "", "", "", 0, err
	}
	defer src.Close()
	if _, err := s.client.PutObject(context.Background(), s.bucket, objectName, src, file.Size, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return "", "", "", 0, err
	}
	return s.publicURL + "/" + s.bucket + "/" + objectName, contentType, strings.TrimSpace(file.Filename), file.Size, nil
}

func (s *MinIOAttachmentStorage) ensureBucket(ctx context.Context) error {
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
