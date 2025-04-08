terraform {
  required_providers {
    github = {
        source  = "integrations/github"
        version = "6.4.0"
    }  
  }

  backend "s3" {
    endpoints = {
      s3 = "https://nyc3.digitaloceanspaces.com"
    }

    bucket = "glizzus-tf-state"
    key    = "sound-off/github/terraform.tfstate"

    skip_credentials_validation = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
    skip_s3_checksum            = true
    region                      = "us-east-1"
  }
}

resource "github_repository" "sound_off_repo" {
  name = "sound-off"
  visibility = "public"

  lifecycle {
    prevent_destroy = true
  }
}
