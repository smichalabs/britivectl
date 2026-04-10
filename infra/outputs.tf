output "cloudfront_domain" {
  description = "Add this as a CNAME in Namecheap for smichalabs.dev (@ record)"
  value       = aws_cloudfront_distribution.docs.domain_name
}

output "docs_url" {
  description = "Your docs will be live at this URL"
  value       = "https://${var.domain}/${var.docs_path}"
}

output "acm_validation_records" {
  description = "Add these DNS records in Namecheap to validate the ACM certificate"
  value = {
    for dvo in aws_acm_certificate.docs.domain_validation_options : dvo.domain_name => {
      type  = dvo.resource_record_type
      name  = dvo.resource_record_name
      value = dvo.resource_record_value
    }
  }
}

output "github_actions_role_arn" {
  description = "Set this as AWS_DOCS_ROLE_ARN secret in the GitHub repo"
  value       = aws_iam_role.docs_deploy.arn
}

output "docs_bucket" {
  description = "Set this as DOCS_BUCKET secret in the GitHub repo"
  value       = aws_s3_bucket.docs.bucket
}

output "cloudfront_distribution_id" {
  description = "Set this as DOCS_CF_DISTRIBUTION_ID secret in the GitHub repo"
  value       = aws_cloudfront_distribution.docs.id
}

output "nameservers" {
  description = "Set these as custom nameservers in Namecheap"
  value       = aws_route53_zone.main.name_servers
}

output "sns_alerts_topic_arn" {
  description = "SNS topic ARN for infrastructure alerts"
  value       = aws_sns_topic.alerts.arn
}
