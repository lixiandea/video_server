package validation

import (
    "bytes"
    "errors"
    "fmt"
    "io"
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

// emailPattern 只做基本的形状校验（本地部分@域名.顶级域），不追求 RFC 完整性。
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// ValidateEmail 校验邮箱格式。空字符串表示未填写，直接视为通过。
func ValidateEmail(email string) error {
    email = strings.TrimSpace(email)
    if email == "" {
        return nil
    }
    if len(email) > 255 {
        return fmt.Errorf("email must be at most 255 characters")
    }
    if !emailPattern.MatchString(email) {
        return fmt.Errorf("invalid email format")
    }

    return nil
}

// ValidateBio 校验个人简介长度。空字符串表示未填写，直接视为通过。
func ValidateBio(bio string) error {
    if len([]rune(bio)) > 200 {
        return fmt.Errorf("bio must be at most 200 characters")
    }

    return nil
}

// ErrFileTooLarge 表示上传文件超过允许的最大体积
var ErrFileTooLarge = errors.New("file too large")

// MaxVideoSize 是未显式配置时的兜底体积上限
const MaxVideoSize int64 = 50 * 1024 * 1024 // 50MB

var (
    allowedVideoExts = []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".webm"}
    asfMagic         = []byte{0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9, 0x00, 0xAA, 0x00, 0x62, 0xCE, 0x6C}

    // videoMagicChecks 依据容器文件头（magic number）判断真实格式
    videoMagicChecks = []func([]byte) bool{
        func(b []byte) bool { return len(b) >= 12 && string(b[4:8]) == "ftyp" },                                       // MP4 / MOV / 3GP
        func(b []byte) bool { return len(b) >= 4 && b[0] == 0x1A && b[1] == 0x45 && b[2] == 0xDF && b[3] == 0xA3 },    // Matroska / WebM
        func(b []byte) bool { return len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "AVI " },         // AVI
        func(b []byte) bool { return len(b) >= 3 && string(b[0:3]) == "FLV" },                                        // FLV
        func(b []byte) bool { return len(b) >= len(asfMagic) && bytes.Equal(b[:len(asfMagic)], asfMagic) },           // ASF / WMV
    }
)

// ValidateVideoFile 同时校验扩展名、体积与文件头，避免仅凭后缀放行任意内容。
func ValidateVideoFile(fileHeader *multipart.FileHeader, maxSize int64) error {
    if fileHeader == nil {
        return fmt.Errorf("missing uploaded file")
    }

    ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
    if !containsString(allowedVideoExts, ext) {
        return fmt.Errorf("invalid file type. Allowed types: %v", allowedVideoExts)
    }

    if maxSize <= 0 {
        maxSize = MaxVideoSize
    }
    if fileHeader.Size > maxSize {
        return fmt.Errorf("%w: maximum size is %d bytes", ErrFileTooLarge, maxSize)
    }

    return validateVideoContent(fileHeader)
}

// validateVideoContent 读取文件头并匹配受支持的视频容器签名。
func validateVideoContent(fileHeader *multipart.FileHeader) error {
    file, err := fileHeader.Open()
    if err != nil {
        return fmt.Errorf("cannot open uploaded file: %w", err)
    }
    defer file.Close()

    head := make([]byte, 16)
    n, err := io.ReadFull(file, head)
    if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
        return fmt.Errorf("cannot read uploaded file: %w", err)
    }
    head = head[:n]

    for _, matches := range videoMagicChecks {
        if matches(head) {
            return nil
        }
    }

    return fmt.Errorf("file content does not match a supported video format")
}

func containsString(list []string, target string) bool {
    for _, item := range list {
        if item == target {
            return true
        }
    }
    return false
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
