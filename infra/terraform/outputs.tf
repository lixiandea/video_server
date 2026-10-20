output "droplet_ipv4" {
  value = digitalocean_droplet.app.ipv4_address
}

output "mysql_uri" {
  value     = digitalocean_database_cluster.mysql.uri
  sensitive = true
}

output "mysql_host" {
  value = digitalocean_database_cluster.mysql.host
}

output "mysql_port" {
  value = digitalocean_database_cluster.mysql.port
}

output "redis_uri" {
  value     = digitalocean_database_cluster.redis.uri
  sensitive = true
}

output "primary_bucket" {
  value = digitalocean_spaces_bucket.media.name
}

output "backup_bucket" {
  value = digitalocean_spaces_bucket.backup.name
}

output "app_url" {
  value = "https://${var.app_domain}"
}
