package warehouse_receipt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type AttachmentStorage interface {
	SaveAttachment(file *multipart.FileHeader) (url, mimeType, originalName string, size int64, err error)
	SaveAttachmentBytes(fileName, mimeType string, data []byte) (url string, err error)
	DeleteByURL(url string) error
}

type NoopAttachmentStorage struct{}

func (NoopAttachmentStorage) SaveAttachment(file *multipart.FileHeader) (string, string, string, int64, error) {
	return "", "", "", 0, ErrReceiptAttachmentStorage
}

func (NoopAttachmentStorage) SaveAttachmentBytes(fileName, mimeType string, data []byte) (string, error) {
	return "", ErrReceiptAttachmentStorage
}

func (NoopAttachmentStorage) DeleteByURL(url string) error { return nil }

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
	meta, err := validateAttachmentFile(file)
	if err != nil {
		return "", "", "", 0, err
	}
	src, err := file.Open()
	if err != nil {
		return "", "", "", 0, err
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		return "", "", "", 0, err
	}
	url, err := s.SaveAttachmentBytes(meta.Name, meta.MimeType, data)
	if err != nil {
		return "", "", "", 0, err
	}
	return url, meta.MimeType, meta.Name, meta.Size, nil
}

func (s *MinIOAttachmentStorage) SaveAttachmentBytes(fileName, mimeType string, data []byte) (string, error) {
	meta, err := validateAttachmentPayload(fileName, mimeType, int64(len(data)))
	if err != nil {
		return "", err
	}
	if err := s.ensureBucket(context.Background()); err != nil {
		return "", err
	}
	objectName := fmt.Sprintf("warehouse-receipts/%s%s", newAttachmentToken(), meta.Ext)
	if _, err := s.client.PutObject(context.Background(), s.bucket, objectName, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: meta.MimeType}); err != nil {
		return "", err
	}
	return s.publicURL + "/" + s.bucket + "/" + objectName, nil
}

func (s *MinIOAttachmentStorage) DeleteByURL(url string) error {
	prefix := s.publicURL + "/" + s.bucket + "/"
	if !strings.HasPrefix(url, prefix) {
		return nil
	}
	objectName := strings.TrimPrefix(url, prefix)
	if strings.TrimSpace(objectName) == "" {
		return nil
	}
	return s.client.RemoveObject(context.Background(), s.bucket, objectName, minio.RemoveObjectOptions{})
}

type attachmentMeta struct {
	Name     string
	MimeType string
	Size     int64
	Ext      string
}

func validateAttachmentFile(file *multipart.FileHeader) (attachmentMeta, error) {
	if file == nil {
		return attachmentMeta{}, ErrReceiptAttachmentRequired
	}
	return validateAttachmentPayload(file.Filename, file.Header.Get("Content-Type"), file.Size)
}

func validateAttachmentPayload(fileName, mimeType string, size int64) (attachmentMeta, error) {
	if size <= 0 || size > attachmentMaxBytes {
		return attachmentMeta{}, ErrReceiptAttachmentSize
	}
	contentType := strings.ToLower(strings.TrimSpace(mimeType))
	ext := normalizeExt(fileName, contentType)
	if ext == "" {
		return attachmentMeta{}, ErrReceiptAttachmentType
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
			return attachmentMeta{}, ErrReceiptAttachmentType
		}
	}
	if contentType != "application/pdf" && contentType != "image/jpeg" && contentType != "image/png" {
		return attachmentMeta{}, ErrReceiptAttachmentType
	}
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "attachment" + ext
	}
	if filepath.Ext(strings.ToLower(name)) == "" {
		name = name + ext
	}
	return attachmentMeta{Name: name, MimeType: contentType, Size: size, Ext: ext}, nil
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
