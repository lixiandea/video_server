// admin 提供生产环境的一次性管理员初始化命令。
//
//	ADMIN_PASSWORD='...' admin ensure --username ops
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/lixiandea/video_server/internal/config"
	"github.com/lixiandea/video_server/internal/models"
	"github.com/lixiandea/video_server/pkg/auth"
	"github.com/lixiandea/video_server/pkg/database"
	"github.com/lixiandea/video_server/pkg/validation"
	"gorm.io/gorm"
)

func main() {
	flags := flag.NewFlagSet("admin", flag.ExitOnError)
	username := flags.String("username", "", "管理员用户名")
	if len(os.Args) < 2 || os.Args[1] != "ensure" {
		fmt.Fprintln(os.Stderr, "用法: ADMIN_PASSWORD='<密码>' admin ensure --username <用户名>")
		os.Exit(2)
	}
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	if err := validation.ValidateUsername(*username); err != nil {
		fatal(err)
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if path := os.Getenv("ADMIN_PASSWORD_FILE"); password == "" && path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			fatal(fmt.Errorf("read ADMIN_PASSWORD_FILE: %w", err))
		}
		password = strings.TrimSpace(string(content))
	}
	if err := validation.ValidatePassword(password); err != nil {
		fatal(fmt.Errorf("ADMIN_PASSWORD is invalid: %w", err))
	}

	cfg := config.LoadConfig()
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		fatal(err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		fatal(err)
	}

	var user models.User
	result := db.Where("login_name = ?", *username).First(&user)
	switch {
	case result.Error == nil:
		if err := db.Model(&user).Updates(map[string]interface{}{
			"role": "admin",
			"pwd":  hash,
		}).Error; err != nil {
			fatal(err)
		}
		fmt.Printf("管理员已更新: %s\n", *username)
	case errors.Is(result.Error, gorm.ErrRecordNotFound):
		user = models.User{
			LoginName: *username,
			Pwd:       hash,
			Role:      "admin",
		}
		if err := db.Create(&user).Error; err != nil {
			fatal(err)
		}
		fmt.Printf("管理员已创建: %s\n", *username)
	default:
		fatal(result.Error)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "admin:", err)
	os.Exit(1)
}
