package main

import (
    "log"
    "net/http"
    "os"
    "path/filepath"
)

func main() {
    port := "3000"
    
    // 获取当前工作目录
    wd, err := os.Getwd()
    if err != nil {
        log.Fatal("无法获取当前工作目录:", err)
    }
    
    // 处理当在frontend目录中运行的情况
    // 如果当前目录是frontend，则返回上级目录
    if filepath.Base(wd) == "frontend" {
        wd = filepath.Dir(wd)
    }
    
    // 设置静态文件服务目录
    staticDir := filepath.Join(wd, "frontend")
    
    // 检查目录是否存在
    if _, err := os.Stat(staticDir); os.IsNotExist(err) {
        log.Fatalf("前端目录不存在: %s", staticDir)
    }
    
    log.Printf("正在端口 %s 上启动前端服务器", port)
    log.Printf("前端文件目录: %s", staticDir)
    log.Printf("访问地址: http://localhost:%s", port)
    
    // 提供静态文件服务
    http.Handle("/", http.FileServer(http.Dir(staticDir)))
    
    log.Fatal(http.ListenAndServe(":"+port, nil))
}