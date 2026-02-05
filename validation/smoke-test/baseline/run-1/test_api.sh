#!/bin/bash

BASE_URL="http://localhost:5000"
COOKIE_FILE="cookies.txt"

echo "=== Testing Flask Multi-Tenant API ==="
echo ""

echo "1. Login as admin user..."
curl -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}' \
  -c "$COOKIE_FILE" \
  -s | jq .

echo ""
echo "2. Get current user info..."
curl -X GET "$BASE_URL/auth/me" \
  -b "$COOKIE_FILE" \
  -s | jq .

echo ""
echo "3. Upload sample file..."
FILE_RESPONSE=$(curl -X POST "$BASE_URL/files" \
  -F "file=@test_data/sample.txt" \
  -b "$COOKIE_FILE" \
  -s)

echo "$FILE_RESPONSE" | jq .
FILE_ID=$(echo "$FILE_RESPONSE" | jq -r '.id')

echo ""
echo "4. Wait for background processing (3 seconds)..."
sleep 3

echo ""
echo "5. Get file details..."
curl -X GET "$BASE_URL/files/$FILE_ID" \
  -b "$COOKIE_FILE" \
  -s | jq .

echo ""
echo "6. List all files..."
curl -X GET "$BASE_URL/files?limit=5" \
  -b "$COOKIE_FILE" \
  -s | jq .

echo ""
echo "7. Logout..."
curl -X POST "$BASE_URL/auth/logout" \
  -b "$COOKIE_FILE" \
  -s | jq .

echo ""
echo "8. Test unauthorized access (should fail)..."
curl -X GET "$BASE_URL/files" \
  -s | jq .

echo ""
echo "=== Tests Complete ==="
