#!/usr/bin/env bash
# setup-github-secrets.sh -- sets GitHub Actions secrets for the britivectl repo
# Usage: AWS_PROFILE=terraform ./scripts/setup-github-secrets.sh
set -euo pipefail

REPO="smichalabs/britivectl"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()    { echo -e "${CYAN}  →${NC} $*"; }
success() { echo -e "${GREEN}  ✓${NC} $*"; }
fatal()   { echo -e "${RED}  ✗${NC} $*" >&2; exit 1; }

echo ""
echo -e "${BOLD}bctl -- GitHub secrets setup${NC}"
echo ""

command -v gh >/dev/null 2>&1 || fatal "gh CLI not found. Install: brew install gh"
gh auth status >/dev/null 2>&1 || fatal "Not logged into gh. Run: gh auth login"
command -v aws >/dev/null 2>&1 || fatal "aws CLI not found. Install: brew install awscli"

# ── Resolve AWS values ─────────────────────────────────────────────────────────

info "Resolving AWS resource values..."

TERRAFORM_ROLE_ARN=$(aws iam get-role --role-name bctl-terraform \
  --query 'Role.Arn' --output text 2>/dev/null) \
  || fatal "Could not find bctl-terraform role. Run bootstrap-aws.sh first."
success "Terraform role: ${TERRAFORM_ROLE_ARN}"

DOCS_ROLE_ARN=$(aws iam get-role --role-name bctl-docs-deploy \
  --query 'Role.Arn' --output text 2>/dev/null) \
  || fatal "Could not find bctl-docs-deploy role. Run terraform apply first."
success "Docs deploy role: ${DOCS_ROLE_ARN}"

DIST_ID=$(aws cloudfront list-distributions \
  --query "DistributionList.Items[0].Id" --output text 2>/dev/null) \
  || fatal "Could not list CloudFront distributions."
success "CloudFront distribution: ${DIST_ID}"

# ── Set secrets ────────────────────────────────────────────────────────────────

info "Setting GitHub secrets on ${REPO}..."

gh secret set AWS_TERRAFORM_ROLE_ARN   --repo "${REPO}" --body "${TERRAFORM_ROLE_ARN}"
success "AWS_TERRAFORM_ROLE_ARN"

gh secret set AWS_DOCS_ROLE_ARN        --repo "${REPO}" --body "${DOCS_ROLE_ARN}"
success "AWS_DOCS_ROLE_ARN"

gh secret set DOCS_BUCKET              --repo "${REPO}" --body "smichalabs-docs"
success "DOCS_BUCKET"

gh secret set DOCS_CF_DISTRIBUTION_ID  --repo "${REPO}" --body "${DIST_ID}"
success "DOCS_CF_DISTRIBUTION_ID"

# ── GitHub PAT for publishing to britivectl-releases ──────────────────────────

if [[ -z "${RELEASES_GITHUB_TOKEN:-}" ]]; then
  printf "  RELEASES_GITHUB_TOKEN (PAT with contents:write on britivectl-releases): "
  read -rs RELEASES_GITHUB_TOKEN </dev/tty
  echo ""
fi

gh secret set RELEASES_GITHUB_TOKEN   --repo "${REPO}" --body "${RELEASES_GITHUB_TOKEN}"
success "RELEASES_GITHUB_TOKEN"

# ── Clean up old secrets ──────────────────────────────────────────────────────

info "Removing old static key secrets (if they exist)..."
gh secret delete TF_AWS_ACCESS_KEY_ID     --repo "${REPO}" 2>/dev/null && success "Removed TF_AWS_ACCESS_KEY_ID" || true
gh secret delete TF_AWS_SECRET_ACCESS_KEY --repo "${REPO}" 2>/dev/null && success "Removed TF_AWS_SECRET_ACCESS_KEY" || true

echo ""
echo -e "${BOLD}==> All secrets set. CI will use OIDC for AWS authentication.${NC}"
echo ""
