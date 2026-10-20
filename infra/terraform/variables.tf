variable "digitalocean_token" {
  type      = string
  sensitive = true
}

variable "cloudflare_api_token" {
  type      = string
  sensitive = true
}

variable "cloudflare_zone" {
  type        = string
  description = "Cloudflare 根域名，例如 example.com"
}

variable "app_domain" {
  type        = string
  description = "应用域名，例如 video.example.com"
}

variable "region" {
  type    = string
  default = "nyc3"
}

variable "backup_region" {
  type    = string
  default = "sfo3"
}

variable "ssh_key_fingerprint" {
  type = string
}

variable "deploy_ssh_public_key" {
  type        = string
  description = "专用 deploy 用户的 SSH 公钥"
}

variable "ssh_allowed_ips" {
  type = list(string)
}

variable "project_name" {
  type    = string
  default = "video-server"
}

variable "mysql_size" {
  type    = string
  default = "db-s-2vcpu-4gb"
}

variable "redis_size" {
  type    = string
  default = "db-s-1vcpu-1gb"
}

variable "cloudflare_ingress_cidrs" {
  type = list(string)
  default = [
    "173.245.48.0/20",
    "103.21.244.0/22",
    "103.22.200.0/22",
    "103.31.4.0/22",
    "141.101.64.0/18",
    "108.162.192.0/18",
    "190.93.240.0/20",
    "188.114.96.0/20",
    "197.234.240.0/22",
    "198.41.128.0/17",
    "162.158.0.0/15",
    "104.16.0.0/13",
    "104.24.0.0/14",
    "172.64.0.0/13",
    "131.0.72.0/22",
    "2400:cb00::/32",
    "2606:4700::/32",
    "2803:f800::/32",
    "2405:b500::/32",
    "2405:8100::/32",
    "2a06:98c0::/29",
    "2c0f:f248::/32"
  ]
}
