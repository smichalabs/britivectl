#!/usr/bin/env bash
# bootstrap-aws.sh — creates a least-privilege IAM user for running the bctl docs Terraform
# Usage: AWS_ACCESS_KEY_ID=xxx AWS_SECRET_ACCESS_KEY=xxx ./scripts/bootstrap-aws.sh
# Or run interactively — it will prompt for credentials if not set.
set -euo pipefail

POLICY_NAME="bctl-docs-terraform"
USER_NAME="terraform-cli"
STATE_BUCKET="smichalabs-terraform-state"

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
echo -e "${BOLD}bctl docs — AWS bootstrap${NC}"
echo "  Creates a least-privilege IAM user for running Terraform."
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
  || fatal "Could not authenticate — check your credentials"
success "Authenticated (account: ${ACCOUNT_ID})"
echo ""

# ── Least-privilege policy document ───────────────────────────────────────────

POLICY_DOC=$(cat <<'POLICY'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "S3Docs",
      "Effect": "Allow",
      "Action": [
        "s3:Get*",
        "s3:List*",
        "s3:CreateBucket",
        "s3:DeleteBucket",
        "s3:PutBucketPolicy",
        "s3:DeleteBucketPolicy",
        "s3:PutBucketPublicAccessBlock",
        "s3:PutObject",
        "s3:DeleteObject"
      ],
      "Resource": "*"
    },
    {
      "Sid": "CloudFront",
      "Effect": "Allow",
      "Action": [
        "cloudfront:CreateDistribution",
        "cloudfront:GetDistribution",
        "cloudfront:UpdateDistribution",
        "cloudfront:DeleteDistribution",
        "cloudfront:TagResource",
        "cloudfront:ListTagsForResource",
        "cloudfront:CreateOriginAccessControl",
        "cloudfront:GetOriginAccessControl",
        "cloudfront:UpdateOriginAccessControl",
        "cloudfront:DeleteOriginAccessControl",
        "cloudfront:ListOriginAccessControls",
        "cloudfront:CreateInvalidation",
        "cloudfront:ListDistributions",
        "cloudfront:CreateFunction",
        "cloudfront:UpdateFunction",
        "cloudfront:DeleteFunction",
        "cloudfront:DescribeFunction",
        "cloudfront:PublishFunction",
        "cloudfront:GetFunction"
      ],
      "Resource": "*"
    },
    {
      "Sid": "ACM",
      "Effect": "Allow",
      "Action": [
        "acm:RequestCertificate",
        "acm:DescribeCertificate",
        "acm:DeleteCertificate",
        "acm:ListTagsForCertificate",
        "acm:AddTagsToCertificate"
      ],
      "Resource": "*"
    },
    {
      "Sid": "IAMOIDCAndRole",
      "Effect": "Allow",
      "Action": [
        "iam:CreateOpenIDConnectProvider",
        "iam:GetOpenIDConnectProvider",
        "iam:DeleteOpenIDConnectProvider",
        "iam:TagOpenIDConnectProvider",
        "iam:CreateRole",
        "iam:GetRole",
        "iam:DeleteRole",
        "iam:TagRole",
        "iam:UntagRole",
        "iam:ListRoleTags",
        "iam:PutRolePolicy",
        "iam:GetRolePolicy",
        "iam:DeleteRolePolicy",
        "iam:ListRolePolicies",
        "iam:ListAttachedRolePolicies",
        "iam:PassRole"
      ],
      "Resource": "*"
    },
    {
      "Sid": "TerraformState",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket",
        "s3:CreateBucket",
        "s3:GetBucketVersioning",
        "s3:PutBucketVersioning"
      ],
      "Resource": [
        "arn:aws:s3:::smichalabs-terraform-state",
        "arn:aws:s3:::smichalabs-terraform-state/*"
      ]
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

# ── Create Terraform state bucket ─────────────────────────────────────────────

info "Creating Terraform state bucket '${STATE_BUCKET}'..."
if aws s3api head-bucket --bucket "${STATE_BUCKET}" 2>/dev/null; then
  warn "State bucket '${STATE_BUCKET}' already exists — skipping"
else
  aws s3api create-bucket --bucket "${STATE_BUCKET}" --region us-east-1 >/dev/null
  aws s3api put-bucket-versioning --bucket "${STATE_BUCKET}" \
    --versioning-configuration Status=Enabled >/dev/null
  success "Created state bucket '${STATE_BUCKET}' with versioning"
fi

# ── Create IAM user ────────────────────────────────────────────────────────────

info "Creating IAM user '${USER_NAME}'..."
if aws iam get-user --user-name "${USER_NAME}" &>/dev/null; then
  warn "User '${USER_NAME}' already exists — skipping creation"
else
  aws iam create-user --user-name "${USER_NAME}" >/dev/null
  success "Created IAM user '${USER_NAME}'"
fi

# ── Create and attach policy ───────────────────────────────────────────────────

info "Creating least-privilege policy '${POLICY_NAME}'..."
POLICY_ARN="arn:aws:iam::${ACCOUNT_ID}:policy/${POLICY_NAME}"

if aws iam get-policy --policy-arn "${POLICY_ARN}" &>/dev/null; then
  warn "Policy '${POLICY_NAME}' already exists — updating..."
  VERSION_ID=$(aws iam list-policy-versions --policy-arn "${POLICY_ARN}" \
    --query 'Versions[?!IsDefaultVersion].VersionId' --output text | head -1)
  if [[ -n "${VERSION_ID}" ]]; then
    aws iam delete-policy-version --policy-arn "${POLICY_ARN}" --version-id "${VERSION_ID}" >/dev/null
  fi
  aws iam create-policy-version --policy-arn "${POLICY_ARN}" \
    --policy-document "${POLICY_DOC}" --set-as-default >/dev/null
else
  POLICY_ARN=$(aws iam create-policy \
    --policy-name "${POLICY_NAME}" \
    --policy-document "${POLICY_DOC}" \
    --query 'Policy.Arn' --output text)
fi
success "Policy ready: ${POLICY_ARN}"

info "Attaching policy to user..."
aws iam attach-user-policy --user-name "${USER_NAME}" --policy-arn "${POLICY_ARN}"
success "Policy attached"

# ── Create access key (skip if one already exists) ────────────────────────────

info "Checking access keys for '${USER_NAME}'..."
KEY_COUNT=$(aws iam list-access-keys --user-name "${USER_NAME}" \
  --query 'length(AccessKeyMetadata)' --output text)

if [[ "${KEY_COUNT}" -ge 1 ]]; then
  warn "Access key already exists — skipping creation (delete old keys manually if needed)"
  KEY_ID="(existing — check aws configure --profile terraform)"
  KEY_SECRET="(existing)"
else
  info "Creating access key for '${USER_NAME}'..."
  KEY_OUTPUT=$(aws iam create-access-key --user-name "${USER_NAME}")
  KEY_ID=$(echo "${KEY_OUTPUT}" | python3 -c "import sys,json; d=json.load(sys.stdin)['AccessKey']; print(d['AccessKeyId'])")
  KEY_SECRET=$(echo "${KEY_OUTPUT}" | python3 -c "import sys,json; d=json.load(sys.stdin)['AccessKey']; print(d['SecretAccessKey'])")
  success "Access key created"
fi

# ── Summary ────────────────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}==> Done. Next steps:${NC}"
echo ""
echo "  1. Configure local CLI profile:"
echo "     aws configure --profile terraform"
echo -e "     ${YELLOW}AWS Access Key ID:${NC}     ${KEY_ID}"
echo -e "     ${YELLOW}AWS Secret Access Key:${NC} ${KEY_SECRET}"
echo -e "     ${YELLOW}Default region:${NC}        us-east-1"
echo ""
echo "  2. Add GitHub repo secrets (Settings → Secrets → Actions):"
echo -e "     ${YELLOW}TF_AWS_ACCESS_KEY_ID${NC}     = ${KEY_ID}"
echo -e "     ${YELLOW}TF_AWS_SECRET_ACCESS_KEY${NC} = ${KEY_SECRET}"
echo ""
echo "  3. Push to main — CI will run terraform apply automatically"
echo ""
echo -e "${BOLD}==> After terraform apply:${NC}"
echo "  1. Add DNS records from 'terraform output acm_validation_records' to Namecheap"
echo "  2. Add GitHub repo secrets from 'terraform output'"
echo "  3. Delete this user's access key (you won't need it again):"
echo ""
echo "  aws iam delete-access-key --user-name ${USER_NAME} --access-key-id ${KEY_ID} --profile terraform"
echo ""
warn "Store the key and secret above securely — this is the only time they are shown."
echo ""
