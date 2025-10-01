#!/bin/bash

echo "🧪 Testing LearnHub Payment Integration"
echo "========================================"

BASE_URL="http://localhost:8080/api"

# Function to make API requests
make_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local token=$4
    
    if [ -z "$token" ]; then
        response=$(curl -s -X $method "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    else
        response=$(curl -s -X $method "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $token" \
            -d "$data")
    fi
    
    echo $response
}

# Step 1: Register a test user
echo ""
echo "1. 📝 Registering test user..."
USER_DATA='{
    "first_name": "Test",
    "last_name": "Student",
    "email": "teststudent@example.com",
    "password": "password123",
    "phone": "0912345678",
    "role": "student"
}'

register_response=$(make_request "POST" "/register" "$USER_DATA")
echo "Register Response: $register_response"

# Extract user ID from response (you might need to adjust this based on your response format)
user_id=$(echo $register_response | grep -o '"id":[0-9]*' | cut -d: -f2)
echo "User ID: $user_id"

# Step 2: Login to get token
echo ""
echo "2. 🔐 Logging in..."
LOGIN_DATA='{
    "email": "teststudent@example.com",
    "password": "password123"
}'

login_response=$(make_request "POST" "/login" "$LOGIN_DATA")
echo "Login Response: $login_response"

# Extract token
token=$(echo $login_response | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "Token: $token"

# Step 3: Create a test course (as instructor)
echo ""
echo "3. 📚 Creating test course..."
COURSE_DATA='{
    "title": "Advanced Web Development",
    "description": "Learn modern web development with React, Node.js, and MongoDB",
    "price": 299.99,
    "category": "Web Development",
    "level": "intermediate",
    "published": true
}'

course_response=$(make_request "POST" "/courses" "$COURSE_DATA" "$token")
echo "Course Response: $course_response"

# Extract course ID
course_id=$(echo $course_response | grep -o '"id":[0-9]*' | cut -d: -f2)
echo "Course ID: $course_id"

# Step 4: Initiate payment for the course
echo ""
echo "4. 💳 Initiating payment..."
PAYMENT_DATA="{
    \"course_id\": $course_id
}"

payment_response=$(make_request "POST" "/payments/initiate" "$PAYMENT_DATA" "$token")
echo "Payment Response: $payment_response"

# Extract payment ID and checkout URL
payment_id=$(echo $payment_response | grep -o '"payment_id":[0-9]*' | cut -d: -f2)
checkout_url=$(echo $payment_response | grep -o '"checkout_url":"[^"]*' | cut -d'"' -f4)
tx_ref=$(echo $payment_response | grep -o '"transaction_ref":"[^"]*' | cut -d'"' -f4)

echo "Payment ID: $payment_id"
echo "Transaction Reference: $tx_ref"
echo "Checkout URL: $checkout_url"

# Step 5: Check payment status
echo ""
echo "5. 🔍 Checking payment status..."
if [ ! -z "$payment_id" ]; then
    status_response=$(make_request "GET" "/payments/status/$payment_id" "" "$token")
    echo "Status Response: $status_response"
fi

# Step 6: Check user enrollments
echo ""
echo "6. 📖 Checking user enrollments..."
enrollments_response=$(make_request "GET" "/my-enrollments" "" "$token")
echo "Enrollments: $enrollments_response"

# Step 7: Check user payments
echo ""
echo "7. 💰 Checking user payment history..."
payments_response=$(make_request "GET" "/my-payments" "" "$token")
echo "Payments: $payments_response"

echo ""
echo "✅ Test flow completed!"
echo ""
echo "📋 Next Steps:"
echo "1. Visit the checkout URL manually to test payment: $checkout_url"
echo "2. Use Chapa test credentials to complete payment"
echo "3. Check webhook.site to see webhook data"
echo "4. Verify enrollment is created after payment"