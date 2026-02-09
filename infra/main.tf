terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }

    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }

  backend "s3" {
    endpoints = {
      s3 = "https://nyc3.digitaloceanspaces.com"
    }

    bucket = "glizzus-tf-state"
    key    = "sound-off/infra/terraform.tfstate"

    skip_credentials_validation = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
    skip_s3_checksum            = true
    region                      = "us-east-1"
  }
}

variable "cloudflare_token" {
  type      = string
  sensitive = true
}

variable "do_token" {
  type      = string
  sensitive = true
}

provider "cloudflare" {
  api_token = var.cloudflare_token
}

provider "digitalocean" {
  token = var.do_token
}

resource "digitalocean_ssh_key" "infra_ssh_key" {
  name       = "infra-ssh-key"
  public_key = file(pathexpand("~/.ssh/soundoff-infra.pub"))
}

resource "digitalocean_droplet" "infra_droplet" {
  name     = "infra-droplet"
  region   = "nyc3"
  size     = "s-2vcpu-2gb"
  image    = "fedora-42-x64"
  ssh_keys = [digitalocean_ssh_key.infra_ssh_key.id]
  tags     = ["sound-off:infra"]
}

resource "digitalocean_kubernetes_cluster" "soundoff_cluster" {
  name    = "soundoff-cluster"
  region  = "nyc3"
  version = "1.34.1-do.3"
  ha      = false

  node_pool {
    name       = "default"
    size       = "s-1vcpu-2gb"
    node_count = 1
    auto_scale = false
  }
}

resource "digitalocean_firewall" "infra_firewall" {
  name        = "infra-firewall"
  droplet_ids = [digitalocean_droplet.infra_droplet.id]

  inbound_rule {
    protocol              = "tcp"
    port_range            = "5432"
    source_kubernetes_ids = [digitalocean_kubernetes_cluster.soundoff_cluster.id]
  }

  inbound_rule {
    protocol              = "tcp"
    port_range            = "6379"
    source_kubernetes_ids = [digitalocean_kubernetes_cluster.soundoff_cluster.id]
  }

  inbound_rule {
    protocol              = "tcp"
    port_range            = "9000"
    source_kubernetes_ids = [digitalocean_kubernetes_cluster.soundoff_cluster.id]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "53"
    destination_addresses = ["0.0.0.0/0"]
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "443"
    destination_addresses = ["0.0.0.0/0"]
  }
}

resource "cloudflare_record" "infra_domains" {
  for_each = toset([
    "database.soundoff.glizzus.net",
    "bucket.soundoff.glizzus.net",
    "redis.soundoff.glizzus.net"
  ])

  zone_id = "5160a6971536e107f477a6b4e6f08e86"
  name    = each.value
  content = digitalocean_droplet.infra_droplet.ipv4_address
  type    = "A"
}

resource "digitalocean_project" "soundoff_project" {
  name        = "Sound Off"
  purpose     = "Web Application"
  description = "Annoying and punctual Discord bot"
  environment = "production"

  resources = [
    digitalocean_droplet.infra_droplet.urn,
    digitalocean_kubernetes_cluster.soundoff_cluster.urn,
  ]
}
