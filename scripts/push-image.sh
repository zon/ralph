#!/bin/bash
set -e

# Configuration variables
REPOSITORY="${RALPH_IMAGE_REPOSITORY:-ghcr.io/zon/ralph}"
BASE_VERSION="$(cat internal/version/VERSION)"
VERSION="${BASE_VERSION}"
# A detached HEAD reports either an empty name or "HEAD", so fall back to an
# explicit override or CI metadata instead of treating that as a feature branch.
BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
if [ -z "${BRANCH}" ] || [ "${BRANCH}" = "HEAD" ]; then
  BRANCH="${RALPH_IMAGE_BRANCH:-${GITHUB_HEAD_REF:-${GITHUB_REF_NAME:-unknown}}}"
fi
if [ "${BRANCH}" != "main" ]; then
  VERSION="${VERSION}-dev"
fi
TAG="${RALPH_IMAGE_TAG:-${VERSION}}"
IMAGE="${REPOSITORY}:${TAG}"

echo "Building Ralph default image..."
echo "Repository: ${REPOSITORY}"
echo "Tag: ${TAG}"
echo "Full image: ${IMAGE}"
echo ""

# Authenticate to GitHub Container Registry
echo "Authenticating to GitHub Container Registry..."
if ! TOKEN=$(gh auth token 2>/dev/null) || [ -z "$TOKEN" ]; then
  echo "Error: not authenticated to GitHub. Run: gh auth login"
  exit 1
fi

if ! gh auth status 2>&1 | grep -q "write:packages"; then
  echo "Error: GitHub token is missing the 'write:packages' scope."
  echo "Re-authenticate with: gh auth login --scopes write:packages"
  exit 1
fi

echo "$TOKEN" | podman login ghcr.io -u zon --password-stdin
echo ""

# Build the image
echo "Building image with Podman..."
podman build -t "${IMAGE}" -f containers/ralph/Containerfile .

echo ""
echo "Image built successfully!"
echo ""

# Scan before pushing so a high or critical finding blocks the release
echo "Scanning vulnerabilities..."
"$(dirname "$0")/scan.sh" image "${IMAGE}"

echo ""

# Push the image with version tag
echo "Pushing image to registry..."
podman push "${IMAGE}"

echo ""
echo "Image pushed successfully with tag: ${TAG}"

# Also tag and push under a rolling tag: latest on main, dev otherwise
if [ "${BRANCH}" = "main" ]; then
  ALIAS_TAG="latest"
else
  ALIAS_TAG="dev"
fi
ALIAS_IMAGE="${REPOSITORY}:${ALIAS_TAG}"
if [ "${ALIAS_IMAGE}" = "${IMAGE}" ]; then
  echo "Tag ${ALIAS_TAG} already matches the pushed tag, skipping alias push."
else
  echo "Tagging and pushing as ${ALIAS_TAG}..."
  podman tag "${IMAGE}" "${ALIAS_IMAGE}"
  podman push "${ALIAS_IMAGE}"
fi

echo ""
echo "Images pushed successfully!"
echo "  - ${IMAGE}"
echo "  - ${ALIAS_IMAGE}"
echo ""
echo "You can now use this image in your workflow configuration:"
echo ""
echo "workflow:"
echo "  image:"
echo "    repository: ${REPOSITORY}"
echo "    tag: ${TAG}"
