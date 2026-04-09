variable "domain" {
  description = "Custom domain for the docs site (e.g. bctl.smichalabs.dev)"
  type        = string
  default     = "bctl.smichalabs.dev"
}

variable "bucket_name" {
  description = "S3 bucket name for docs content"
  type        = string
  default     = "bctl-smichalabs-docs"
}
