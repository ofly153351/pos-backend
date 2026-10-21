package rustfs

import (
	"context"
	"fmt"

	s3client "github.com/minio/minio-go/v7"
)

// EnsurePublicBucket creates the configured bucket and makes object reads
// available to browsers. RustFS exposes the standard S3 bucket-policy API.
func EnsurePublicBucket(ctx context.Context, client *s3client.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, s3client.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, bucket)
	return client.SetBucketPolicy(ctx, bucket, policy)
}
