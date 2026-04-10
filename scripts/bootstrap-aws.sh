#!/usr/bin/env bash
# bootstrap-aws.sh -- creates the OIDC provider and Terraform IAM role for CI
# Usage: AWS_ACCESS_KEY_ID=xxx AWS_SECRET_ACCESS_KEY=xxx ./scripts/bootstrap-aws.sh
# Or run interactively -- it will prompt for credentials if not set.
#
# This script is idempotent and only needs to be run once per AWS account.
# After running, set the AWS_TERRAFORM_ROLE_ARN GitHub secret and CI handles
# everything via OIDC -- no static keys needed.
set -euo pipefail

ROLE_NAME="bctl-terraform"
POLICY_NAME="bctl-terraform-infra"
STATE_BUCKET="smichalabs-terraform-state"
GITHUB_REPO="smichalabs/britivectl"
OIDC_URL="https://token.actions.githubusercontent.com"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()    { echo -e "${CYAN}  →${NC} $*"; }
success() { echo -e "${GREEN}  ✓${NC} $*"; }
warn()    { echo -e "${YELLOW}  !${NC} $*"; }
fatal()   { echo -e "${RED}  ✗${NC} $*" >&2; exit 1; }

echo ""
echo -e "${BOLD}bctl -- AWS bootstrap (OIDC)${NC}"
echo "  Creates the GitHub OIDC provider and Terraform IAM role for CI."
echo ""

# ── Prompt for root credentials if not in environment ─────────────────────────

if [[ -z "${AWS_ACCESS_KEY_ID:-}" ]]; then
  printf "  AWS root Access Key ID: "
  read -r AWS_ACCESS_KEY_ID </dev/tty
  export AWS_ACCESS_KEY_ID
fi

if [[ -z "${AWS_SECRET_ACCESS_KEY:-}" ]]; then
  printf "  AWS root Secret Access Key: "
  read -rs AWS_SECRET_ACCESS_KEY </dev/tty
  echo ""
  export AWS_SECRET_ACCESS_KEY
fi

export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-us-east-1}"

# Verify credentials work
info "Verifying root credentials..."
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text 2>/dev/null) \
  || fatal "Could not authenticate -- check your credentials"
success "Authenticated (account: ${ACCOUNT_ID})"
echo ""

# ── Create Terraform state bucket ─────────────────────────────────────────────

info "Creating Terraform state bucket '${STATE_BUCKET}'..."
if aws s3api head-bucket --bucket "${STATE_BUCKET}" 2>/dev/null; then
  warn "State bucket '${STATE_BUCKET}' already exists -- skipping"
else
  aws s3api create-bucket --bucket "${STATE_BUCKET}" --region us-east-1 >/dev/null
  aws s3api put-bucket-versioning --bucket "${STATE_BUCKET}" \
    --versioning-configuration Status=Enabled >/dev/null
  success "Created state bucket '${STATE_BUCKET}' with versioning"
fi

# ── Create GitHub OIDC provider ───────────────────────────────────────────────

info "Checking GitHub OIDC provider..."
EXISTING_OIDC=$(aws iam list-open-id-connect-providers \
  --query "OpenIDConnectProviderList[?ends_with(Arn, '/token.actions.githubusercontent.com')].Arn" \
  --output text 2>/dev/null || true)

if [[ -n "${EXISTING_OIDC}" ]]; then
  OIDC_ARN="${EXISTING_OIDC}"
  warn "OIDC provider already exists -- skipping (${OIDC_ARN})"
else
  OIDC_ARN=$(aws iam create-open-id-connect-provider \
    --url "${OIDC_URL}" \
    --client-id-list "sts.amazonaws.com" \
    --thumbprint-list "6938fd4d98bab03faadb97b34396831e3780aea1" \
    --query "OpenIDConnectProviderArn" --output text)
  success "Created OIDC provider (${OIDC_ARN})"
fi

# ── Create Terraform IAM role ─────────────────────────────────────────────────

TRUST_POLICY=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "${OIDC_ARN}"
    },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringLike": {
        "token.actions.githubusercontent.com:sub": "repo:${GITHUB_REPO}:*"
      },
      "StringEquals": {
        "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
      }
    }
  }]
}
EOF
)

info "Creating IAM role '${ROLE_NAME}'..."
if aws iam get-role --role-name "${ROLE_NAME}" &>/dev/null; then
  warn "Role '${ROLE_NAME}' already exists -- updating trust policy"
  aws iam update-assume-role-policy --role-name "${ROLE_NAME}" \
    --policy-document "${TRUST_POLICY}" >/dev/null
else
  aws iam create-role --role-name "${ROLE_NAME}" \
    --assume-role-policy-document "${TRUST_POLICY}" >/dev/null
  success "Created IAM role '${ROLE_NAME}'"
fi

ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/${ROLE_NAME}"

# ── Attach infrastructure policy ─────────────────────────────────────────────

INFRA_POLICY=$(cat <<'POLICY'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "S3",
      "Effect": "Allow",
      "Action": "s3:*",
      "Resource": "*"
    },
    {
      "Sid": "CloudFront",
      "Effect": "Allow",
      "Action": "cloudfront:*",
      "Resource": "*"
    },
    {
      "Sid": "ACM",
      "Effect": "Allow",
      "Action": "acm:*",
      "Resource": "*"
    },
    {
      "Sid": "IAM",
      "Effect": "Allow",
      "Action": [
        "iam:CreateRole",
        "iam:GetRole",
        "iam:DeleteRole",
        "iam:UpdateRole",
        "iam:TagRole",
        "iam:UntagRole",
        "iam:ListRoleTags",
        "iam:PutRolePolicy",
        "iam:GetRolePolicy",
        "iam:DeleteRolePolicy",
        "iam:ListRolePolicies",
        "iam:ListAttachedRolePolicies",
        "iam:PassRole",
        "iam:GetOpenIDConnectProvider",
        "iam:ListOpenIDConnectProviders"
      ],
      "Resource": "*"
    },
    {
      "Sid": "SNS",
      "Effect": "Allow",
      "Action": "sns:*",
      "Resource": "*"
    },
    {
      "Sid": "CloudWatch",
      "Effect": "Allow",
      "Action": [
        "cloudwatch:PutMetricAlarm",
        "cloudwatch:DescribeAlarms",
        "cloudwatch:DeleteAlarms",
        "cloudwatch:ListTagsForResource"
      ],
      "Resource": "*"
    },
    {
      "Sid": "STS",
      "Effect": "Allow",
      "Action": "sts:GetCallerIdentity",
      "Resource": "*"
    }
  ]
}
POLICY
)

info "Attaching infrastructure policy..."
POLICY_ARN="arn:aws:iam::${ACCOUNT_ID}:policy/${POLICY_NAME}"

if aws iam get-policy --policy-arn "${POLICY_ARN}" &>/dev/null; then
  warn "Policy '${POLICY_NAME}' already exists -- updating..."
  VERSION_ID=$(aws iam list-policy-versions --policy-arn "${POLICY_ARN}" \
    --query 'Versions[?!IsDefaultVersion].VersionId' --output text | head -1)
  if [[ -n "${VERSION_ID}" ]]; then
    aws iam delete-policy-version --policy-arn "${POLICY_ARN}" --version-id "${VERSION_ID}" >/dev/null
  fi
  aws iam create-policy-version --policy-arn "${POLICY_ARN}" \
    --policy-document "${INFRA_POLICY}" --set-as-default >/dev/null
else
  POLICY_ARN=$(aws iam create-policy \
    --policy-name "${POLICY_NAME}" \
    --policy-document "${INFRA_POLICY}" \
    --query 'Policy.Arn' --output text)
fi
success "Policy ready: ${POLICY_ARN}"

aws iam attach-role-policy --role-name "${ROLE_NAME}" --policy-arn "${POLICY_ARN}" 2>/dev/null || true
success "Policy attached to role"

# ── Summary ────────────────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}==> Done. Next steps:${NC}"
echo ""
echo "  1. Set the GitHub repo secret:"
echo -e "     ${YELLOW}AWS_TERRAFORM_ROLE_ARN${NC} = ${ROLE_ARN}"
echo ""
echo "     Or run:"
echo "     gh secret set AWS_TERRAFORM_ROLE_ARN --body '${ROLE_ARN}'"
echo ""
echo "  2. Remove old static key secrets if they exist:"
echo "     gh secret delete TF_AWS_ACCESS_KEY_ID"
echo "     gh secret delete TF_AWS_SECRET_ACCESS_KEY"
echo ""
echo "  3. Remove Terraform state reference to the OIDC provider:"
echo "     cd infra && terraform state rm aws_iam_openid_connect_provider.github"
echo ""
echo "  4. Delete the old terraform-cli IAM user (if it exists):"
echo "     aws iam detach-user-policy --user-name terraform-cli --policy-arn arn:aws:iam::${ACCOUNT_ID}:policy/bctl-docs-terraform"
echo "     aws iam list-access-keys --user-name terraform-cli --query 'AccessKeyMetadata[].AccessKeyId' --output text | xargs -I{} aws iam delete-access-key --user-name terraform-cli --access-key-id {}"
echo "     aws iam delete-user --user-name terraform-cli"
echo "     aws iam delete-policy --policy-arn arn:aws:iam::${ACCOUNT_ID}:policy/bctl-docs-terraform"
echo ""
echo "  5. Push the code changes -- CI will use OIDC from now on."
echo ""
