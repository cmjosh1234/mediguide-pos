package storage

import (
	"bytes"
	"context"
	"github.com/google/uuid"
	"io"
	"mediguide/internal/config"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

// Explicitly opt in against LOCAL test storage. Creates and deletes only a
// unique upload-tests object; never attaches anything to a guideline.
func TestMinioMultipartIntegration(t *testing.T) {
	endpoint := os.Getenv("MINIO_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("set MINIO_TEST_ENDPOINT for the local storage smoke test")
	}
	s, err := NewMinioStore(config.Config{S3Endpoint: endpoint, S3PublicEndpoint: endpoint, S3Bucket: "mediguide", S3AccessKey: "mediguide", S3SecretKey: "mediguide123"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	key := "upload-tests/" + uuid.NewString() + "/source.pdf"
	id, err := s.BeginUpload(ctx, key, "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.AbortUpload(context.Background(), key, id); _ = s.Delete(context.Background(), key) })
	// A genuinely expired signed URL must be rejected; the normal fresh signer
	// below then resumes the same multipart session successfully.
	expired, err := s.uploadClient.Presign(ctx, "PUT", s.bucket, key, time.Second, url.Values{"uploadId": {id}, "partNumber": {"1"}})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2100 * time.Millisecond)
	expiredRequest, _ := http.NewRequestWithContext(ctx, http.MethodPut, expired.String(), bytes.NewReader([]byte("expired")))
	expiredResponse, err := http.DefaultClient.Do(expiredRequest)
	if err != nil {
		t.Fatal("expired URL request failed")
	}
	expiredResponse.Body.Close()
	if expiredResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("expired URL status %d", expiredResponse.StatusCode)
	}
	content := append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("x"), 9<<20)...)
	for number, offset := 1, 0; offset < len(content); number, offset = number+1, offset+(8<<20) {
		url, err := s.SignUploadPart(ctx, key, id, number)
		if err != nil {
			t.Fatal(err)
		}
		preflight, _ := http.NewRequestWithContext(ctx, http.MethodOptions, url, nil)
		preflight.Header.Set("Origin", "http://localhost:3000")
		preflight.Header.Set("Access-Control-Request-Method", "PUT")
		cors, err := http.DefaultClient.Do(preflight)
		if err != nil {
			t.Fatal("CORS preflight failed")
		}
		cors.Body.Close()
		origin := cors.Header.Get("Access-Control-Allow-Origin")
		if cors.StatusCode >= 300 || (origin != "*" && origin != "http://localhost:3000") {
			t.Fatalf("CORS preflight not permitted: status %d", cors.StatusCode)
		}
		end := min(offset+(8<<20), len(content))
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(content[offset:end]))
		if err != nil {
			t.Fatal(err)
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal("signed PUT failed")
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatalf("signed PUT status %d", response.StatusCode)
		}
		if number == 1 {
			// Reconnect with a fresh storage client, as after a browser/process
			// interruption. Part one remains durable and is never PUT again.
			s, err = NewMinioStore(config.Config{S3Endpoint: endpoint, S3PublicEndpoint: endpoint, S3Bucket: "mediguide", S3AccessKey: "mediguide", S3SecretKey: "mediguide123"})
			if err != nil {
				t.Fatal(err)
			}
			stored, err := s.UploadParts(ctx, key, id)
			if err != nil || len(stored) != 1 || stored[0].Number != 1 || stored[0].Size != 8<<20 {
				t.Fatalf("resume lost completed part: %+v %v", stored, err)
			}
		}
	}
	parts, err := s.UploadParts(ctx, key, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 2 || parts[0].Size+parts[1].Size != int64(len(content)) {
		t.Fatalf("unexpected stored parts: %+v", parts)
	}
	if err = s.CompleteUpload(ctx, key, id, parts); err != nil {
		t.Fatal(err)
	}
	size, err := s.ObjectSize(ctx, key)
	if err != nil || size != int64(len(content)) {
		t.Fatalf("object size: %d %v", size, err)
	}
	reader, err := s.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || !bytes.Equal(actual, content) {
		t.Fatal("stored bytes differ")
	}
	if err = s.AbortUpload(ctx, key, id); err != nil {
		t.Fatal("abort must be idempotent after completion", err)
	}
}
