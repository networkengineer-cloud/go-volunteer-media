#!/bin/bash

# Test image upload endpoint
# Usage: ./test_upload.sh <token> <image_file>

TOKEN=${1:-"your-jwt-token"}
IMAGE_FILE=${2:-"/tmp/test.png"}

# Create a test image if it doesn't exist
if [ ! -f "$IMAGE_FILE" ]; then
    echo "Creating test PNG image..."
    # Create a simple 1x1 PNG (base64 encoded minimal PNG)
    echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==" | base64 -d > "$IMAGE_FILE"
fi

echo "Testing upload to http://localhost:8080/api/animals/upload-image"
echo "Token: $TOKEN"
echo "File: $IMAGE_FILE"

curl -X POST http://localhost:8080/api/animals/upload-image \
  -H "Authorization: Bearer $TOKEN" \
  -F "image=@$IMAGE_FILE" \
  -v

echo ""
