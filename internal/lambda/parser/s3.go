package parser

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3GetAPI adapts aws-sdk GetObject for ObjectReader.
type S3GetAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type s3Reader struct {
	api S3GetAPI
}

// NewS3ObjectReader returns an ObjectReader backed by S3.
func NewS3ObjectReader(api S3GetAPI) ObjectReader {
	return s3Reader{api: api}
}

func (r s3Reader) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	out, err := r.api.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read s3 body: %w", err)
	}
	return data, nil
}

// NoopObjectReader fails all reads (misconfigured Lambda).
type NoopObjectReader struct{}

func (NoopObjectReader) GetObject(context.Context, string, string) ([]byte, error) {
	return nil, fmt.Errorf("s3 object reader not configured")
}
