#!/bin/bash
set -e

# Configuration variables
REPOSITORY="${RALPH_IMAGE_REPOSITORY:-ghcr.io/zon/ralph}"
TAG=$(cat CONTAINER_VERSION)
IMAGE="${REPOSITORY}:${TAG}"

echo "Building Ralph default image..."
echo "Repository: ${REPOSITORY}"
echo "Tag: ${TAG}"
echo "Full image: ${IMAGE}"
echo ""

# Build the image
echo "Building image with Podman..."
podman build -t "${IMAGE}" -f Containerfile .

echo ""
echo "Image built successfully!"
echo ""

# Push the image
echo "Pushing image to registry..."
podman push "${IMAGE}"

echo ""
echo "Image pushed successfully!"
echo "Image: ${IMAGE}"
echo ""
echo "You can now use this image in your workflow configuration:"
echo ""
echo "workflow:"
echo "  image:"
echo "    repository: ${REPOSITORY}"
echo "    tag: ${TAG}"
