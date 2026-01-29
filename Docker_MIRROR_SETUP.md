# Docker 国内镜像加速配置指南

由于国内访问Docker Hub速度较慢，建议配置国内镜像加速器以提高拉取镜像的速度。

## macOS (Docker Desktop) 配置方法

对于 macOS 上的 Docker Desktop 用户，请按以下步骤配置：

1. 打开 Docker Desktop 应用程序
2. 点击右上角的鲸鱼图标，选择 "Preferences..."
3. 在左侧菜单中选择 "Docker Engine"
4. 在右侧编辑框中替换原有内容为以下配置：

```json
{
  "builder": {
    "gc": {
      "enabled": true,
      "defaultKeepStorage": "20GB"
    }
  },
  "experimental": false,
  "features": {
    "buildkit": true
  },
  "registry-mirrors": [
    "https://hub-mirror.c.163.com",
    "https://docker.mirrors.ustc.edu.cn",
    "https://ccr.ccs.tencentyun.com",
    "https://mirror.ccs.tencentyun.com",
    "https://registry.docker-cn.com"
  ]
}
```

5. 点击 "Apply & Restart" 按钮重启 Docker Desktop

## Linux 配置方法

对于 Linux 系统用户，可以编辑 `/etc/docker/daemon.json` 文件（如果没有此文件则创建）：

```bash
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json <<-'EOF'
{
  "registry-mirrors": [
    "https://hub-mirror.c.163.com",
    "https://docker.mirrors.ustc.edu.cn",
    "https://ccr.ccs.tencentyun.com",
    "https://mirror.ccs.tencentyun.com",
    "https://registry.docker-cn.com"
  ]
}
EOF
sudo systemctl daemon-reload
sudo systemctl restart docker
```

## 验证配置

配置完成后，可以通过以下命令验证镜像加速器是否生效：

```bash
docker info
```

如果配置成功，在输出信息中会显示 `Registry Mirrors` 部分，列出配置的镜像地址。

## 推荐使用的镜像加速器

- 网易云：`https://hub-mirror.c.163.com`
- 中科大：`https://docker.mirrors.ustc.edu.cn`
- 腾讯云：`https://ccr.ccs.tencentyun.com` 和 `https://mirror.ccs.tencentyun.com`
- Docker中国：`https://registry.docker-cn.com`

这些镜像源可以显著提升Docker镜像的拉取速度。

## 在你的项目中使用

你的项目使用了以下镜像：
- mysql:8.0
- redis:alpine

配置镜像加速后，这些官方镜像的拉取速度将显著提升。

要启动你的视频服务器项目，请使用以下命令：

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 停止所有服务
docker-compose down
```

如果在配置过程中遇到任何问题，请参考上述步骤重新配置Docker镜像加速器。