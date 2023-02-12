terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "2.26.0"
    }
  }
}

variable "discord_token" {
  description = "Discord API token."
  type        = string
  sensitive   = true
}

variable "discord_app_public_key" {
  description = "Discord application public key."
  type        = string
}

resource "digitalocean_app" "discord_leetcode" {
  spec {
    name   = "discord-leetcode"
    region = "nyc1"

    alert {
      rule = "DEPLOYMENT_FAILED"
    }

    function {
      name       = "interaction"
      source_dir = "functions"

      github {
        repo           = "sudomateo/discord-leetcode"
        branch         = "main"
        deploy_on_push = true
      }

      env {
        key   = "DISCORD_TOKEN"
        value = var.discord_token
        scope = "RUN_TIME"
        type  = "SECRET"
      }

      env {
        key   = "DISCORD_APP_PUBLIC_KEY"
        value = var.discord_app_public_key
        scope = "RUN_TIME"
        type  = "GENERAL"
      }

      routes {
        path = "/"
      }
    }
  }
}
