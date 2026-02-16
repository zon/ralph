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
gh auth token | podman login ghcr.io -u zon --password-stdin
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
