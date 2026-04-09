variable "domain" {
  description = "Root domain for the docs site"
  type        = string
  default     = "smichalabs.dev"
}

variable "bucket_name" {
  description = "S3 bucket name for docs content"
  type        = string
  default     = "smichalabs-docs"
}

variable "docs_path" {
  description = "S3 key prefix where docs are stored (e.g. utils/bctl)"
  type        = string
  default     = "utils/bctl"
}
