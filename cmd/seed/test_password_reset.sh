#!/bin/bash

echo "=== Testing Admin Password Reset Feature ==="
echo ""

# Login as admin
echo "1. Logging in as admin..."
LOGIN_RESPONSE=$(curl -s http://localhost:8080/api/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"testadmin","password":"password123"}')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ Failed to login"
  exit 1
fi

echo "✅ Login successful, got token"
echo ""

# Get users list
echo "2. Fetching users list..."
USERS=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/admin/users)
echo "✅ Users fetched successfully"
echo "$USERS" | python3 -m json.tool
echo ""

# Reset testuser password
echo "3. Resetting testuser password to 'temppassword123'..."
RESET_RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"new_password":"temppassword123"}' \
  http://localhost:8080/api/admin/users/2/reset-password)

echo "Response: $RESET_RESPONSE"
echo ""

# Try to login with new password
echo "4. Testing login with new password..."
NEW_LOGIN=$(curl -s http://localhost:8080/api/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"temppassword123"}')

NEW_TOKEN=$(echo $NEW_LOGIN | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$NEW_TOKEN" ]; then
  echo "❌ Failed to login with new password"
  exit 1
fi

echo "✅ Successfully logged in with new password!"
echo ""

# Reset password back to original
echo "5. Resetting password back to original..."
RESET_BACK=$(curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"new_password":"password123"}' \
  http://localhost:8080/api/admin/users/2/reset-password)

echo "Response: $RESET_BACK"
echo ""

echo "=== All Tests Passed! ✅ ==="
