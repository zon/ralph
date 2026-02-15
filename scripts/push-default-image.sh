#!/bin/bash
set -e

# Configuration variables
REPOSITORY="${RALPH_IMAGE_REPOSITORY:-ghcr.io/zon/ralph}"
TAG="${RALPH_IMAGE_TAG:-latest}"
IMAGE="${REPOSITORY}:${TAG}"

echo "Building Ralph default image..."
echo "Repository: ${REPOSITORY}"
echo "Tag: ${TAG}"
echo "Full image: ${IMAGE}"
echo ""

# Build the image
echo "Building Docker image..."
docker build -t "${IMAGE}" -f Dockerfile .

echo ""
echo "Image built successfully!"
echo ""

# Push the image
echo "Pushing image to registry..."
docker push "${IMAGE}"

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
