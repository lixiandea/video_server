package storage

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    
    "github.com/lixiandea/video_server/internal/config"
)

type StorageService struct {
    cfg *config.StorageConfig
}

func NewStorageService(cfg *config.StorageConfig) *StorageService {
    // Ensure directories exist
    dirs := []string{cfg.VideoDir, cfg.TemplateDir, cfg.TempDir}
    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            panic(fmt.Sprintf("Failed to create directory %s: %v", dir, err))
        }
    }
    
    return &StorageService{cfg: cfg}
}

func (s *StorageService) SaveVideo(filename string, data io.Reader) error {
    // Sanitize filename to prevent directory traversal
    cleanFilename := filepath.Base(filename)
    filepath := filepath.Join(s.cfg.VideoDir, cleanFilename)
    
    file, err := os.Create(filepath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    _, err = io.Copy(file, data)
    return err
}

func (s *StorageService) GetVideoPath(filename string) string {
    cleanFilename := filepath.Base(filename)
    return filepath.Join(s.cfg.VideoDir, cleanFilename)
}

func (s *StorageService) DeleteVideo(filename string) error {
    cleanFilename := filepath.Base(filename)
    filepath := filepath.Join(s.cfg.VideoDir, cleanFilename)
    
    return os.Remove(filepath)
}

func (s *StorageService) VideoExists(filename string) bool {
    cleanFilename := filepath.Base(filename)
    filepath := filepath.Join(s.cfg.VideoDir, cleanFilename)
    
    _, err := os.Stat(filepath)
    return !os.IsNotExist(err)
}

func (s *StorageService) GetVideoInfo(filename string) (os.FileInfo, error) {
    cleanFilename := filepath.Base(filename)
    filepath := filepath.Join(s.cfg.VideoDir, cleanFilename)
    
    return os.Stat(filepath)
}

func (s *StorageService) GetTemplatePath(filename string) string {
    cleanFilename := filepath.Base(filename)
    return filepath.Join(s.cfg.TemplateDir, cleanFilename)
}