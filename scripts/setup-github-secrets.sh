#!/usr/bin/env bash
# setup-github-secrets.sh — sets GitHub Actions secrets for the britivectl repo
# Usage: TF_AWS_ACCESS_KEY_ID=xxx TF_AWS_SECRET_ACCESS_KEY=xxx ./scripts/setup-github-secrets.sh
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
echo -e "${BOLD}bctl — GitHub secrets setup${NC}"
echo ""

command -v gh >/dev/null 2>&1 || fatal "gh CLI not found. Install: brew install gh"
gh auth status >/dev/null 2>&1 || fatal "Not logged into gh. Run: gh auth login"

# ── Terraform CLI credentials (for infra workflow) ─────────────────────────────

if [[ -z "${TF_AWS_ACCESS_KEY_ID:-}" ]]; then
  printf "  TF_AWS_ACCESS_KEY_ID (terraform-cli key): "
  read -r TF_AWS_ACCESS_KEY_ID </dev/tty
fi

if [[ -z "${TF_AWS_SECRET_ACCESS_KEY:-}" ]]; then
  printf "  TF_AWS_SECRET_ACCESS_KEY: "
  read -rs TF_AWS_SECRET_ACCESS_KEY </dev/tty
  echo ""
fi

# ── Resolve AWS values ─────────────────────────────────────────────────────────

info "Resolving AWS resource values..."

DIST_ID=$(AWS_ACCESS_KEY_ID="${TF_AWS_ACCESS_KEY_ID}" \
          AWS_SECRET_ACCESS_KEY="${TF_AWS_SECRET_ACCESS_KEY}" \
          AWS_DEFAULT_REGION=us-east-1 \
          aws cloudfront list-distributions \
          --query "DistributionList.Items[0].Id" --output text)

ROLE_ARN=$(AWS_ACCESS_KEY_ID="${TF_AWS_ACCESS_KEY_ID}" \
           AWS_SECRET_ACCESS_KEY="${TF_AWS_SECRET_ACCESS_KEY}" \
           AWS_DEFAULT_REGION=us-east-1 \
           aws iam get-role --role-name bctl-docs-deploy \
           --query 'Role.Arn' --output text)

# ── Set secrets ────────────────────────────────────────────────────────────────

info "Setting GitHub secrets on ${REPO}..."

gh secret set TF_AWS_ACCESS_KEY_ID     --repo "${REPO}" --body "${TF_AWS_ACCESS_KEY_ID}"
success "TF_AWS_ACCESS_KEY_ID"

gh secret set TF_AWS_SECRET_ACCESS_KEY --repo "${REPO}" --body "${TF_AWS_SECRET_ACCESS_KEY}"
success "TF_AWS_SECRET_ACCESS_KEY"

gh secret set DOCS_BUCKET              --repo "${REPO}" --body "smichalabs-docs"
success "DOCS_BUCKET"

gh secret set DOCS_CF_DISTRIBUTION_ID  --repo "${REPO}" --body "${DIST_ID}"
success "DOCS_CF_DISTRIBUTION_ID"

gh secret set AWS_DOCS_ROLE_ARN        --repo "${REPO}" --body "${ROLE_ARN}"
success "AWS_DOCS_ROLE_ARN"

# ── GitHub PAT for publishing to britivectl-releases ──────────────────────────

if [[ -z "${RELEASES_GITHUB_TOKEN:-}" ]]; then
  printf "  RELEASES_GITHUB_TOKEN (PAT with contents:write on britivectl-releases): "
  read -rs RELEASES_GITHUB_TOKEN </dev/tty
  echo ""
fi

gh secret set RELEASES_GITHUB_TOKEN   --repo "${REPO}" --body "${RELEASES_GITHUB_TOKEN}"
success "RELEASES_GITHUB_TOKEN"

echo ""
echo -e "${BOLD}==> All secrets set. You can now merge feat/docs-site → main.${NC}"
echo ""
