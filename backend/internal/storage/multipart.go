package storage

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
)

type UploadPart struct {
	Number int    `json:"number"`
	ETag   string `json:"etag"`
	Size   int64  `json:"size"`
}

// MultipartStore is optional; deployments without a public storage endpoint use
// the existing backend upload endpoint.
type MultipartStore interface {
	BeginUpload(context.Context, string, string) (string, error)
	SignUploadPart(context.Context, string, string, int) (string, error)
	UploadParts(context.Context, string, string) ([]UploadPart, error)
	CompleteUpload(context.Context, string, string, []UploadPart) error
	AbortUpload(context.Context, string, string) error
	ObjectSize(context.Context, string) (int64, error)
}

func (s *MinioStore) BeginUpload(ctx context.Context, key, mime string) (string, error) {
	if s.uploadClient == nil {
		return "", errors.New("direct uploads are not configured")
	}
	return (minio.Core{Client: s.client}).NewMultipartUpload(ctx, s.bucket, key, minio.PutObjectOptions{ContentType: mime})
}
func (s *MinioStore) SignUploadPart(ctx context.Context, key, id string, number int) (string, error) {
	if s.uploadClient == nil {
		return "", errors.New("direct uploads are not configured")
	}
	u, err := s.uploadClient.Presign(ctx, "PUT", s.bucket, key, 15*time.Minute, url.Values{"uploadId": {id}, "partNumber": {strconv.Itoa(number)}})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
func (s *MinioStore) UploadParts(ctx context.Context, key, id string) ([]UploadPart, error) {
	parts := []UploadPart{}
	marker := 0
	for {
		page, err := (minio.Core{Client: s.client}).ListObjectParts(ctx, s.bucket, key, id, marker, 1000)
		if err != nil {
			return nil, err
		}
		for _, p := range page.ObjectParts {
			parts = append(parts, UploadPart{Number: p.PartNumber, ETag: p.ETag, Size: p.Size})
		}
		if !page.IsTruncated {
			return parts, nil
		}
		marker = page.NextPartNumberMarker
	}
}
func (s *MinioStore) CompleteUpload(ctx context.Context, key, id string, parts []UploadPart) error {
	ordered := make([]minio.CompletePart, 0, len(parts))
	for _, p := range parts {
		ordered = append(ordered, minio.CompletePart{PartNumber: p.Number, ETag: p.ETag})
	}
	_, err := (minio.Core{Client: s.client}).CompleteMultipartUpload(ctx, s.bucket, key, id, ordered, minio.PutObjectOptions{})
	return err
}
func (s *MinioStore) AbortUpload(ctx context.Context, key, id string) error {
	err := (minio.Core{Client: s.client}).AbortMultipartUpload(ctx, s.bucket, key, id)
	if minio.ToErrorResponse(err).Code == "NoSuchUpload" {
		return nil
	}
	return err
}
func (s *MinioStore) ObjectSize(ctx context.Context, key string) (int64, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	return info.Size, err
}
