#!/bin/bash
# =============================================================================
# Generate RSA Key Pair for JWT Signing
# =============================================================================
#
# This script generates an RSA key pair (2048-bit) for JWT token signing
# and verification in the NisFix Backend.
#
# Usage:
#   ./scripts/generate-keys.sh          # Generate in ./keys directory
#   ./scripts/generate-keys.sh /path    # Generate in custom directory
#
# Output:
#   - private.pem: RSA private key (PKCS#1 format)
#   - public.pem:  RSA public key (PKIX/SPKI format)
#
# Security Notes:
#   - Keep private.pem secure and never commit to version control
#   - The keys directory is gitignored by default
#   - Use different keys for each environment (dev, staging, prod)

set -euo pipefail

# Configuration
KEY_SIZE=2048
KEY_DIR="${1:-./keys}"
PRIVATE_KEY="$KEY_DIR/private.pem"
PUBLIC_KEY="$KEY_DIR/public.pem"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if openssl is available
if ! command -v openssl &> /dev/null; then
    log_error "openssl is not installed. Please install it first."
    exit 1
fi

# Create key directory if it doesn't exist
if [ ! -d "$KEY_DIR" ]; then
    log_info "Creating directory: $KEY_DIR"
    mkdir -p "$KEY_DIR"
fi

# Check if keys already exist
if [ -f "$PRIVATE_KEY" ] || [ -f "$PUBLIC_KEY" ]; then
    log_warn "Existing keys found in $KEY_DIR"
    read -p "Do you want to overwrite them? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Operation cancelled. Existing keys preserved."
        exit 0
    fi
    log_info "Overwriting existing keys..."
fi

# Generate private key
log_info "Generating $KEY_SIZE-bit RSA private key..."
openssl genrsa -out "$PRIVATE_KEY" $KEY_SIZE 2>/dev/null

if [ ! -f "$PRIVATE_KEY" ]; then
    log_error "Failed to generate private key"
    exit 1
fi

# Set restrictive permissions on private key
chmod 600 "$PRIVATE_KEY"

# Extract public key
log_info "Extracting public key..."
openssl rsa -in "$PRIVATE_KEY" -pubout -out "$PUBLIC_KEY" 2>/dev/null

if [ ! -f "$PUBLIC_KEY" ]; then
    log_error "Failed to extract public key"
    exit 1
fi

# Set permissions on public key
chmod 644 "$PUBLIC_KEY"

# Verify the keys
log_info "Verifying key pair..."
VERIFY_RESULT=$(openssl rsa -in "$PRIVATE_KEY" -check 2>&1)
if [[ "$VERIFY_RESULT" != *"RSA key ok"* ]]; then
    log_error "Key verification failed"
    exit 1
fi

# Display success message
echo ""
log_info "RSA key pair generated successfully!"
echo ""
echo "  Private Key: $PRIVATE_KEY (mode: 600)"
echo "  Public Key:  $PUBLIC_KEY (mode: 644)"
echo ""
echo "  Key Size:    $KEY_SIZE bits"
echo "  Algorithm:   RS512 (RSA-SHA512)"
echo ""
log_warn "Keep private.pem secure and never commit it to version control!"
echo ""

# Show environment variable hints
echo "Add to your .env file:"
echo ""
echo "  NISFIX_JWT_PRIVATE_KEY_PATH=$PRIVATE_KEY"
echo "  NISFIX_JWT_PUBLIC_KEY_PATH=$PUBLIC_KEY"
echo ""
