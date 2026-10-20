data "cloudflare_zone" "main" {
  name = var.cloudflare_zone
}

resource "digitalocean_vpc" "main" {
  name     = "${var.project_name}-vpc"
  region   = var.region
  ip_range = "10.20.0.0/20"
}

resource "digitalocean_droplet" "app" {
  name       = "${var.project_name}-app"
  image      = "ubuntu-24-04-x64"
  size       = "s-4vcpu-8gb"
  region     = var.region
  vpc_uuid   = digitalocean_vpc.main.id
  ssh_keys   = [var.ssh_key_fingerprint]
  monitoring = true
  user_data = templatefile("${path.module}/cloud-init.yaml", {
    deploy_ssh_public_key = var.deploy_ssh_public_key
  })

  tags = [var.project_name, "production"]
}

resource "digitalocean_database_cluster" "mysql" {
  name       = "${var.project_name}-mysql"
  engine     = "mysql"
  version    = "8"
  size       = var.mysql_size
  region     = var.region
  node_count = 1
  tags       = [var.project_name, "production"]

  maintenance_window {
    day  = "sunday"
    hour = "08:00"
  }
}

resource "digitalocean_database_db" "main" {
  cluster_id = digitalocean_database_cluster.mysql.id
  name       = "video_server"
}

resource "digitalocean_database_user" "app" {
  cluster_id = digitalocean_database_cluster.mysql.id
  name       = "video_app"
}

resource "digitalocean_database_cluster" "redis" {
  name       = "${var.project_name}-redis"
  engine     = "redis"
  version    = "7"
  size       = var.redis_size
  region     = var.region
  node_count = 1
  tags       = [var.project_name, "production"]

  maintenance_window {
    day  = "sunday"
    hour = "09:00"
  }
}

resource "digitalocean_database_firewall" "mysql" {
  cluster_id = digitalocean_database_cluster.mysql.id
  rule {
    type  = "droplet"
    value = digitalocean_droplet.app.id
  }
}

resource "digitalocean_database_firewall" "redis" {
  cluster_id = digitalocean_database_cluster.redis.id
  rule {
    type  = "droplet"
    value = digitalocean_droplet.app.id
  }
}

resource "digitalocean_spaces_bucket" "media" {
  name   = "${var.project_name}-media-${var.region}"
  region = var.region
  acl    = "private"

  versioning {
    enabled = true
  }
}

resource "digitalocean_spaces_bucket" "backup" {
  name   = "${var.project_name}-backup-${var.backup_region}"
  region = var.backup_region
  acl    = "private"

  versioning {
    enabled = true
  }
}

resource "digitalocean_firewall" "app" {
  name = "${var.project_name}-firewall"

  droplet_ids = [digitalocean_droplet.app.id]

  dynamic "inbound_rule" {
    for_each = toset(var.ssh_allowed_ips)
    content {
      protocol         = "tcp"
      port_range       = "22"
      source_addresses = [inbound_rule.value]
    }
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = var.cloudflare_ingress_cidrs
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = var.cloudflare_ingress_cidrs
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "icmp"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}

resource "cloudflare_record" "app" {
  zone_id = data.cloudflare_zone.main.id
  name    = trimsuffix(var.app_domain, ".${var.cloudflare_zone}")
  value   = digitalocean_droplet.app.ipv4_address
  type    = "A"
  proxied = true
  ttl     = 1
}

resource "cloudflare_ruleset" "application_rate_limits" {
  zone_id = data.cloudflare_zone.main.id
  name    = "${var.project_name} application rate limits"
  kind    = "zone"
  phase   = "http_ratelimit"

  rules {
    action      = "block"
    description = "Protect login endpoint"
    expression  = "(http.request.uri.path eq \"/api/v1/users/login\")"
    ratelimit {
      characteristics     = ["ip.src"]
      period              = 60
      requests_per_period = 10
      mitigation_timeout  = 300
    }
  }

  rules {
    action      = "block"
    description = "Protect registration endpoint"
    expression  = "(http.request.uri.path eq \"/api/v1/users/register\")"
    ratelimit {
      characteristics     = ["ip.src"]
      period              = 60
      requests_per_period = 5
      mitigation_timeout  = 600
    }
  }

  rules {
    action      = "managed_challenge"
    description = "Protect upload endpoint"
    expression  = "(http.request.uri.path eq \"/api/v1/videos/upload\")"
    ratelimit {
      characteristics     = ["ip.src"]
      period              = 60
      requests_per_period = 20
      mitigation_timeout  = 300
    }
  }
}
