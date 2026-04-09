terraform {
  backend "s3" {
    bucket = "smichalabs-terraform-state"
    key    = "docs/terraform.tfstate"
    region = "us-east-1"
  }
}
