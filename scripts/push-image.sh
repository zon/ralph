#!/bin/bash
set -e

# Configuration variables
REPOSITORY="${RALPH_IMAGE_REPOSITORY:-ghcr.io/zon/ralph}"
TAG=$(cat VERSION)
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
podman build -t "${IMAGE}" -f Containerfile .

echo ""
echo "Image built successfully!"
echo ""

# Push the image with version tag
echo "Pushing image to registry..."
podman push "${IMAGE}"

echo ""
echo "Image pushed successfully with tag: ${TAG}"

# Also tag and push as latest
LATEST_IMAGE="${REPOSITORY}:latest"
echo "Tagging and pushing as latest..."
podman tag "${IMAGE}" "${LATEST_IMAGE}"
podman push "${LATEST_IMAGE}"

echo ""
echo "Images pushed successfully!"
echo "  - ${IMAGE}"
echo "  - ${LATEST_IMAGE}"
echo ""
echo "You can now use this image in your workflow configuration:"
echo ""
echo "workflow:"
echo "  image:"
echo "    repository: ${REPOSITORY}"
echo "    tag: ${TAG}"
