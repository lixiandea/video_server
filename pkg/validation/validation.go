package validation

import (
    "fmt"
    "mime/multipart"
    "path/filepath"
    "regexp"
    "strings"
)

// ValidateUsername validates username format
func ValidateUsername(username string) error {
    if len(username) < 3 || len(username) > 30 {
        return fmt.Errorf("username must be between 3 and 30 characters")
    }
    
    matched, err := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
    if err != nil {
        return err
    }
    if !matched {
        return fmt.Errorf("username can only contain letters, numbers, and underscores")
    }
    
    return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
    if len(password) < 6 {
        return fmt.Errorf("password must be at least 6 characters")
    }
    
    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
    hasDigit := regexp.MustCompile(`\d`).MatchString(password)
    
    if !(hasUpper && hasLower && hasDigit) {
        return fmt.Errorf("password must contain at least one uppercase letter, lowercase letter, and digit")
    }
    
    return nil
}

// ValidateVideoFile validates uploaded video file
func ValidateVideoFile(fileHeader *multipart.FileHeader) error {
    // Check file extension
    ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
    validExts := []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".webm"}
    
    isValidExt := false
    for _, validExt := range validExts {
        if ext == validExt {
            isValidExt = true
            break
        }
    }
    
    if !isValidExt {
        return fmt.Errorf("invalid file type. Allowed types: %v", validExts)
    }
    
    // Check file size (using 500MB as max for example)
    if fileHeader.Size > 500*1024*1024 { // 500MB
        return fmt.Errorf("file too large. Maximum size is 500MB")
    }
    
    return nil
}

// ValidateVideoName validates video name
func ValidateVideoName(name string) error {
    if len(name) == 0 {
        return fmt.Errorf("video name cannot be empty")
    }
    
    if len(name) > 200 {
        return fmt.Errorf("video name too long. Maximum 200 characters")
    }
    
    return nil
}