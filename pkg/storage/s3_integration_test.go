package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lixiandea/video_server/internal/config"
)

func TestS3CompatibleLifecycle(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT is not configured")
	}
	accessKey := os.Getenv("S3_TEST_ACCESS_KEY")
	secretKey := os.Getenv("S3_TEST_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Fatal("S3 test credentials are required")
	}

	suffix := time.Now().UnixNano()
	primary := fmt.Sprintf("video-test-%d", suffix)
	backup := fmt.Sprintf("video-backup-%d", suffix)
	client, err := newS3Client(endpoint, "us-east-1", primary, accessKey, secretKey, false)
	if err != nil {
		t.Fatal(err)
	}
	backupClient, err := newS3Client(endpoint, "us-east-1", backup, accessKey, secretKey, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		client *s3.Client
		bucket string
	}{{client, primary}, {backupClient, backup}} {
		if _, err := item.client.CreateBucket(context.Background(), &s3.CreateBucketInput{
			Bucket: aws.String(item.bucket),
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_ = cleanupBucket(client, primary)
		_ = cleanupBucket(backupClient, backup)
	})

	root := t.TempDir()
	service, err := NewStorageService(&config.StorageConfig{
		Backend:           "s3",
		VideoDir:          filepath.Join(root, "videos"),
		TemplateDir:       filepath.Join(root, "templates"),
		TempDir:           filepath.Join(root, "temp"),
		S3Endpoint:        endpoint,
		S3Region:          "us-east-1",
		S3Bucket:          primary,
		S3AccessKey:       accessKey,
		S3SecretKey:       secretKey,
		S3UseSSL:          false,
		S3BackupEndpoint:  endpoint,
		S3BackupRegion:    "us-east-1",
		S3BackupBucket:    backup,
		S3BackupAccessKey: accessKey,
		S3BackupSecretKey: secretKey,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := service.SaveVideo("video-1", bytes.NewBufferString("source")); err != nil {
		t.Fatal(err)
	}
	materialized, err := service.Materialize(context.Background(), service.SourceKey("video-1"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(materialized)
	if err != nil || string(data) != "source" {
		t.Fatalf("unexpected materialized object: %q err=%v", data, err)
	}

	output := filepath.Join(root, "output.mp4")
	if err := os.WriteFile(output, []byte("video"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := service.UploadFile(context.Background(), "videos/video-1/output.mp4", output, "video/mp4"); err != nil {
		t.Fatal(err)
	}
	url, err := service.PresignGet(context.Background(), "videos/video-1/output.mp4", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || string(body) != "video" {
		t.Fatalf("unexpected presigned response: status=%d body=%q", response.StatusCode, body)
	}

	if err := service.CopyVideoToBackup(context.Background(), "video-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := backupClient.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(backup),
		Key:    aws.String("videos/video-1/output.mp4"),
	}); err != nil {
		t.Fatalf("backup object missing: %v", err)
	}
	if err := service.DeleteVideoAssets(context.Background(), "video-1"); err != nil {
		t.Fatal(err)
	}
}

func cleanupBucket(client *s3.Client, bucket string) error {
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return err
		}
		for _, object := range page.Contents {
			_, _ = client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    object.Key,
			})
		}
	}
	_, err := client.DeleteBucket(context.Background(), &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	return err
}
