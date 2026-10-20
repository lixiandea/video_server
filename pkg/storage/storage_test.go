package storage

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lixiandea/video_server/internal/config"
)

func TestLocalStorageLifecycle(t *testing.T) {
	root := t.TempDir()
	service, err := NewStorageService(&config.StorageConfig{
		Backend:     "local",
		VideoDir:    filepath.Join(root, "videos"),
		TemplateDir: filepath.Join(root, "templates"),
		TempDir:     filepath.Join(root, "temp"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.OutputDir() != filepath.Join(root, "videos") {
		t.Fatalf("local output dir must be video dir, got %q", service.OutputDir())
	}
	if err := service.SaveVideo("video-1", bytes.NewBufferString("fixture")); err != nil {
		t.Fatal(err)
	}
	if !service.VideoExists(service.SourceKey("video-1")) {
		t.Fatal("expected uploaded video to exist")
	}
	materialized, err := service.Materialize(context.Background(), service.SourceKey("video-1"))
	if err != nil || materialized == "" {
		t.Fatalf("materialize failed: path=%q err=%v", materialized, err)
	}
	if err := service.DeleteVideoAssets(context.Background(), "video-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(materialized); !os.IsNotExist(err) {
		t.Fatalf("expected local video to be deleted, stat err=%v", err)
	}
}

func TestObjectStorageRequiresCompleteConfiguration(t *testing.T) {
	root := t.TempDir()
	_, err := NewStorageService(&config.StorageConfig{
		Backend:     "s3",
		VideoDir:    filepath.Join(root, "videos"),
		TemplateDir: filepath.Join(root, "templates"),
		TempDir:     filepath.Join(root, "temp"),
	})
	if err == nil {
		t.Fatal("expected incomplete S3 config to fail")
	}
}
