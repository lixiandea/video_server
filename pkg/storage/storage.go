package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lixiandea/video_server/internal/config"
)

// StorageService 对本地磁盘和 S3 兼容对象存储提供统一接口。
type StorageService struct {
	cfg          *config.StorageConfig
	client       *s3.Client
	backupClient *s3.Client
}

func NewStorageService(cfg *config.StorageConfig) (*StorageService, error) {
	dirs := []string{cfg.TempDir}
	if !strings.EqualFold(cfg.Backend, "s3") {
		dirs = append(dirs, cfg.VideoDir, cfg.TemplateDir)
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	service := &StorageService{cfg: cfg}
	if !service.IsObjectStorage() {
		return service, nil
	}
	client, err := newS3Client(cfg.S3Endpoint, cfg.S3Region, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3UseSSL)
	if err != nil {
		return nil, err
	}
	service.client = client
	if cfg.S3BackupBucket != "" {
		backupClient, err := newS3Client(
			cfg.S3BackupEndpoint,
			cfg.S3BackupRegion,
			cfg.S3BackupBucket,
			cfg.S3BackupAccessKey,
			cfg.S3BackupSecretKey,
			cfg.S3UseSSL,
		)
		if err != nil {
			return nil, err
		}
		service.backupClient = backupClient
	}
	return service, nil
}

func newS3Client(endpoint, region, bucket, accessKey, secretKey string, useSSL bool) (*s3.Client, error) {
	if endpoint == "" || region == "" || bucket == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("incomplete S3 configuration")
	}
	scheme := "https://"
	if !useSSL {
		scheme = "http://"
	}
	baseEndpoint := endpoint
	if !strings.HasPrefix(baseEndpoint, "http://") && !strings.HasPrefix(baseEndpoint, "https://") {
		baseEndpoint = scheme + baseEndpoint
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 configuration: %w", err)
	}
	return s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(baseEndpoint)
		options.UsePathStyle = true
	}), nil
}

func (s *StorageService) IsObjectStorage() bool {
	return strings.EqualFold(s.cfg.Backend, "s3")
}

func (s *StorageService) TempDir() string {
	return s.cfg.TempDir
}

func (s *StorageService) OutputDir() string {
	if s.IsObjectStorage() {
		return s.cfg.TempDir
	}
	return s.cfg.VideoDir
}

func (s *StorageService) Healthy(ctx context.Context) error {
	if !s.IsObjectStorage() {
		_, err := os.Stat(s.cfg.TempDir)
		return err
	}
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.cfg.S3Bucket)})
	return err
}

// SourceKey 返回上传源文件在存储层中的标识。
func (s *StorageService) SourceKey(videoID string) string {
	if s.IsObjectStorage() {
		return "source/" + filepath.Base(videoID)
	}
	return s.GetVideoPath(videoID)
}

func (s *StorageService) SaveVideo(filename string, data io.Reader) error {
	if !s.IsObjectStorage() {
		return s.saveLocal(filename, data)
	}
	key := "source/" + filepath.Base(filename)
	tempPath := filepath.Join(s.cfg.TempDir, "upload-"+filepath.Base(filename))
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(tempFile, data); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to spool upload: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}
	defer os.Remove(tempPath)
	return s.UploadFile(context.Background(), key, tempPath, "application/octet-stream")
}

func (s *StorageService) saveLocal(filename string, data io.Reader) error {
	file, err := os.Create(s.GetVideoPath(filename))
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, data)
	return err
}

// Materialize 把对象下载到临时目录；本地存储直接返回现有路径。
func (s *StorageService) Materialize(ctx context.Context, key string) (string, error) {
	if !s.IsObjectStorage() {
		if _, err := os.Stat(key); err != nil {
			return "", err
		}
		return key, nil
	}
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("failed to download object %s: %w", key, err)
	}
	defer output.Body.Close()

	dest := filepath.Join(s.cfg.TempDir, "download-"+filepath.Base(key))
	file, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(file, output.Body); err != nil {
		file.Close()
		return "", fmt.Errorf("failed to materialize object: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return dest, nil
}

func (s *StorageService) UploadFile(ctx context.Context, key, localPath, contentType string) error {
	if !s.IsObjectStorage() {
		return nil
	}
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.S3Bucket),
		Key:         aws.String(strings.TrimPrefix(key, "/")),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload %s: %w", key, err)
	}
	return nil
}

func (s *StorageService) UploadDir(ctx context.Context, prefix, dir string) error {
	if !s.IsObjectStorage() {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		contentType := "application/octet-stream"
		switch filepath.Ext(path) {
		case ".m3u8":
			contentType = "application/vnd.apple.mpegurl"
		case ".ts":
			contentType = "video/mp2t"
		}
		return s.UploadFile(ctx, filepath.Join(prefix, filepath.ToSlash(relative)), path, contentType)
	})
}

func (s *StorageService) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if !s.IsObjectStorage() {
		return "", fmt.Errorf("presigned URLs require object storage")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	presignClient := s3.NewPresignClient(s.client)
	result, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("failed to presign object: %w", err)
	}
	return result.URL, nil
}

func (s *StorageService) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if !s.IsObjectStorage() {
		return os.Open(key)
	}
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}

func (s *StorageService) VideoExists(filename string) bool {
	if !s.IsObjectStorage() {
		_, err := os.Stat(s.GetVideoPath(filename))
		return !os.IsNotExist(err)
	}
	_, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(strings.TrimPrefix(filename, "/")),
	})
	return err == nil
}

func (s *StorageService) DeleteVideoAssets(ctx context.Context, videoID string) error {
	if !s.IsObjectStorage() {
		pattern := filepath.Join(s.cfg.VideoDir, filepath.Base(videoID)+"*")
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			_ = os.RemoveAll(match)
		}
		return nil
	}
	for _, prefix := range []string{"source/" + videoID, "videos/" + videoID + "/"} {
		if err := s.deleteObjects(ctx, s.client, s.cfg.S3Bucket, prefix); err != nil {
			return err
		}
	}
	return nil
}

func (s *StorageService) DeleteObjectsBefore(ctx context.Context, prefix string, cutoff time.Time) error {
	if !s.IsObjectStorage() {
		return nil
	}
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.cfg.S3Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, object := range page.Contents {
			if object.Key == nil || object.LastModified == nil || !object.LastModified.Before(cutoff) {
				continue
			}
			if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(s.cfg.S3Bucket),
				Key:    object.Key,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *StorageService) CopyVideoToBackup(ctx context.Context, videoID string) error {
	if !s.IsObjectStorage() {
		return nil
	}
	if s.backupClient == nil {
		return fmt.Errorf("backup object storage is not configured")
	}
	prefix := "videos/" + videoID + "/"
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.cfg.S3Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list backup candidates: %w", err)
		}
		for _, object := range page.Contents {
			if object.Key == nil {
				continue
			}
			source, err := s.client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(s.cfg.S3Bucket),
				Key:    object.Key,
			})
			if err != nil {
				return err
			}
			tempFile, createErr := os.CreateTemp(s.cfg.TempDir, "replicate-*")
			if createErr != nil {
				source.Body.Close()
				return createErr
			}
			_, copyErr := io.Copy(tempFile, source.Body)
			closeErr := source.Body.Close()
			if copyErr != nil {
				tempFile.Close()
				os.Remove(tempFile.Name())
				return copyErr
			}
			if err := tempFile.Close(); err != nil {
				os.Remove(tempFile.Name())
				return err
			}
			file, openErr := os.Open(tempFile.Name())
			if openErr != nil {
				os.Remove(tempFile.Name())
				return openErr
			}
			_, putErr := s.backupClient.PutObject(ctx, &s3.PutObjectInput{
				Bucket:      aws.String(s.cfg.S3BackupBucket),
				Key:         object.Key,
				Body:        file,
				ContentType: source.ContentType,
			})
			file.Close()
			os.Remove(tempFile.Name())
			if putErr != nil {
				return fmt.Errorf("failed to copy %s to backup bucket: %w", *object.Key, putErr)
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
	return nil
}

func (s *StorageService) deleteObjects(ctx context.Context, client *s3.Client, bucket, prefix string) error {
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, object := range page.Contents {
			if object.Key == nil {
				continue
			}
			if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    object.Key,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *StorageService) GetVideoPath(filename string) string {
	return filepath.Join(s.cfg.VideoDir, filepath.Base(filename))
}

func (s *StorageService) GetTemplatePath(filename string) string {
	return filepath.Join(s.cfg.TemplateDir, filepath.Base(filename))
}

func (s *StorageService) GetVideoInfo(filename string) (os.FileInfo, error) {
	return os.Stat(s.GetVideoPath(filename))
}
