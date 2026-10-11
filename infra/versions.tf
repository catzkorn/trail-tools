terraform {
  required_version = ">= 1.14"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 8.6"
    }
  }

  # The state bucket is created once by hand, see the README.
  backend "gcs" {
    bucket = "trail-tools-444904-tfstate"
    prefix = "trail-tools"
  }
}

provider "google" {
  project = var.project
  region  = var.region
  # Budgets are billing account resources, and their API needs a project
  # to bill requests to.
  user_project_override = true
  billing_project       = var.project
}

data "google_project" "this" {}
