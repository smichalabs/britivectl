terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}


# ── Route 53 ───────────────────────────────────────────────────────────────────

resource "aws_route53_zone" "main" {
  name = var.domain
}

resource "aws_route53_record" "apex" {
  zone_id = aws_route53_zone.main.zone_id
  name    = var.domain
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.docs.domain_name
    zone_id                = aws_cloudfront_distribution.docs.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "acm_validation" {
  for_each = {
    for dvo in aws_acm_certificate.docs.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      type   = dvo.resource_record_type
      record = dvo.resource_record_value
    }
  }

  zone_id = aws_route53_zone.main.zone_id
  name    = each.value.name
  type    = each.value.type
  ttl     = 300
  records = [each.value.record]
}

# ── S3 bucket ──────────────────────────────────────────────────────────────────

resource "aws_s3_bucket" "docs" {
  bucket = var.bucket_name
}

resource "aws_s3_bucket_public_access_block" "docs" {
  bucket                  = aws_s3_bucket.docs.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Apex redirect: visitors hitting https://smichalabs.dev/ (no path) get a
# meta-refresh redirect to the docs path. Without this object the bucket
# returns 403 AccessDenied because there is no key at the root. Replace the
# content with a real homepage when one exists.
resource "aws_s3_object" "apex_index" {
  bucket        = aws_s3_bucket.docs.id
  key           = "index.html"
  content_type  = "text/html; charset=utf-8"
  cache_control = "max-age=300"
  content       = <<-EOT
    <!doctype html>
    <html lang="en">
    <head>
    <meta charset="utf-8">
    <meta http-equiv="refresh" content="0; url=/utils/bctl/">
    <link rel="canonical" href="https://smichalabs.dev/utils/bctl/">
    <title>smichalabs</title>
    </head>
    <body>
    <p>Redirecting to <a href="/utils/bctl/">/utils/bctl/</a>...</p>
    </body>
    </html>
  EOT
}

resource "aws_s3_bucket_server_side_encryption_configuration" "docs" {
  bucket = aws_s3_bucket.docs.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
    bucket_key_enabled = true
  }
}

# ── CloudFront OAC ─────────────────────────────────────────────────────────────

resource "aws_cloudfront_origin_access_control" "docs" {
  name                              = "${var.bucket_name}-oac"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_s3_bucket_policy" "docs" {
  bucket = aws_s3_bucket.docs.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowCloudFront"
        Effect = "Allow"
        Principal = {
          Service = "cloudfront.amazonaws.com"
        }
        Action   = "s3:GetObject"
        Resource = "${aws_s3_bucket.docs.arn}/*"
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.docs.arn
          }
        }
      },
      {
        Sid       = "DenyInsecureTransport"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource  = [aws_s3_bucket.docs.arn, "${aws_s3_bucket.docs.arn}/*"]
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      }
    ]
  })
}

# ── ACM certificate (must be us-east-1 for CloudFront) ─────────────────────────

resource "aws_acm_certificate" "docs" {
  domain_name       = var.domain
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

# ── ACM certificate validation (waits until DNS records are added) ─────────────

resource "aws_acm_certificate_validation" "docs" {
  certificate_arn         = aws_acm_certificate.docs.arn
  validation_record_fqdns = [for r in aws_route53_record.acm_validation : r.fqdn]
}

# ── CloudFront function: rewrite /path/to/dir → /path/to/dir/index.html ────────

resource "aws_cloudfront_function" "rewrite_index" {
  name    = "rewrite-index"
  runtime = "cloudfront-js-2.0"
  publish = true
  code    = <<-EOT
    async function handler(event) {
      var request = event.request;
      var uri = request.uri;
      if (uri.endsWith('/')) {
        request.uri += 'index.html';
      } else if (!uri.includes('.')) {
        // Redirect to the trailing-slash URL instead of silently rewriting.
        // Without the redirect the browser URL bar still shows /utils/bctl
        // (no slash), so it treats bctl as a file in /utils/ and resolves
        // relative CSS/JS paths against /utils/ instead of /utils/bctl/.
        return {
          statusCode: 301,
          statusDescription: 'Moved Permanently',
          headers: { location: { value: uri + '/' } }
        };
      }
      return request;
    }
  EOT
}

# ── CloudFront response headers (security) ─────────────────────────────────────

resource "aws_cloudfront_response_headers_policy" "security" {
  name = "docs-security-headers"

  security_headers_config {
    strict_transport_security {
      access_control_max_age_sec = 63072000
      include_subdomains         = true
      preload                    = true
      override                   = true
    }

    content_type_options {
      override = true
    }

    frame_options {
      frame_option = "DENY"
      override     = true
    }

    referrer_policy {
      referrer_policy = "strict-origin-when-cross-origin"
      override        = true
    }

    content_security_policy {
      content_security_policy = "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; font-src 'self' data:"
      override                = true
    }
  }
}

# ── CloudFront distribution ────────────────────────────────────────────────────

resource "aws_cloudfront_distribution" "docs" {
  enabled     = true
  aliases     = [var.domain]
  price_class = "PriceClass_100" # US + Europe only — cheapest

  origin {
    domain_name              = aws_s3_bucket.docs.bucket_regional_domain_name
    origin_id                = "s3"
    origin_access_control_id = aws_cloudfront_origin_access_control.docs.id
  }

  default_cache_behavior {
    target_origin_id           = "s3"
    viewer_protocol_policy     = "redirect-to-https"
    response_headers_policy_id = aws_cloudfront_response_headers_policy.security.id
    allowed_methods            = ["GET", "HEAD"]
    cached_methods             = ["GET", "HEAD"]
    compress                   = true

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.rewrite_index.arn
    }

    forwarded_values {
      query_string = false
      cookies { forward = "none" }
    }

    min_ttl     = 0
    default_ttl = 300
    max_ttl     = 3600
  }

  # Map upstream errors to the styled 404 page with a real 404 status.
  #
  # Why status 404 (not 200):
  # response_code applies to every CloudFront response that hits this
  # mapping, including asset requests (CSS/JS/images). If we return 200,
  # a missing CSS file gets HTML content with a 200 OK status, and the
  # browser tries to parse HTML as CSS -- silently failing and rendering
  # the page unstyled. Returning 404 lets the browser cleanly skip a
  # failed asset and shows search engines the correct signal for missing
  # pages.
  #
  # Why both 403 and 404 are mapped:
  # S3 with blocked public ACLs returns 403 AccessDenied (not 404) for
  # any missing key. Without the 403 mapping the user sees the raw S3
  # AccessDenied XML page. The 404 mapping is defensive in case anything
  # else upstream ever surfaces a literal 404.
  #
  # error_caching_min_ttl = 0 stops an edge POP from caching a transient
  # error response (e.g. during a deploy window) for the default ~10s.
  custom_error_response {
    error_code            = 404
    response_code         = 404
    response_page_path    = "/404.html"
    error_caching_min_ttl = 0
  }

  custom_error_response {
    error_code            = 403
    response_code         = 404
    response_page_path    = "/404.html"
    error_caching_min_ttl = 0
  }

  restrictions {
    geo_restriction { restriction_type = "none" }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.docs.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }
}

# ── IAM role for GitHub Actions (OIDC) ────────────────────────────────────────

data "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"
}

resource "aws_iam_role" "docs_deploy" {
  name = "bctl-docs-deploy"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = data.aws_iam_openid_connect_provider.github.arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringLike = {
          "token.actions.githubusercontent.com:sub" = "repo:smichalabs/britivectl:ref:refs/heads/main"
        }
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "docs_deploy" {
  name = "bctl-docs-deploy"
  role = aws_iam_role.docs_deploy.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:DeleteObject", "s3:ListBucket"]
        Resource = [aws_s3_bucket.docs.arn, "${aws_s3_bucket.docs.arn}/*"]
      },
      {
        Effect   = "Allow"
        Action   = "cloudfront:CreateInvalidation"
        Resource = aws_cloudfront_distribution.docs.arn
      }
    ]
  })
}

# ── Alerting (SNS + CloudWatch) ────────────────────────────────────────────────

resource "aws_sns_topic" "alerts" {
  name = "docs-infra-alerts"
}

resource "aws_sns_topic_subscription" "email" {
  for_each  = toset(var.alert_emails)
  topic_arn = aws_sns_topic.alerts.arn
  protocol  = "email"
  endpoint  = each.value
}

resource "aws_cloudwatch_metric_alarm" "cf_5xx" {
  alarm_name          = "docs-cloudfront-5xx-errors"
  alarm_description   = "CloudFront 5xx error rate exceeded 5% for 10 minutes"
  namespace           = "AWS/CloudFront"
  metric_name         = "5xxErrorRate"
  statistic           = "Average"
  period              = 300
  evaluation_periods  = 2
  threshold           = 5
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    DistributionId = aws_cloudfront_distribution.docs.id
    Region         = "Global"
  }

  alarm_actions = [aws_sns_topic.alerts.arn]
  ok_actions    = [aws_sns_topic.alerts.arn]
}

resource "aws_cloudwatch_metric_alarm" "cf_4xx" {
  alarm_name          = "docs-cloudfront-4xx-anomaly"
  alarm_description   = "CloudFront 4xx error rate exceeded 25% for 15 minutes"
  namespace           = "AWS/CloudFront"
  metric_name         = "4xxErrorRate"
  statistic           = "Average"
  period              = 300
  evaluation_periods  = 3
  threshold           = 25
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    DistributionId = aws_cloudfront_distribution.docs.id
    Region         = "Global"
  }

  alarm_actions = [aws_sns_topic.alerts.arn]
  ok_actions    = [aws_sns_topic.alerts.arn]
}
