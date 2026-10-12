terraform {
  required_version = ">= 1.15.5, < 2.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.68.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "2.8.1"
    }
  }
}

provider "aws" {
  region = var.aws_region
  default_tags {
    tags = { Application = "npt-shortenlink", Environment = "prod" }
  }
}
