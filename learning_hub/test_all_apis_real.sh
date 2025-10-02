#!/bin/bash

echo "🧪 COMPREHENSIVE LEARNHUB API TESTING WITH REAL EMAILS"
echo "======================================================"

# CORRECTED BASE URL - using port 3000 and API version
BASE_URL="http://localhost:8080/api/v1"

# REAL EMAIL ADDRESSES FOR TESTING
ADMIN_EMAIL="ermiasabebe1808@gmail.com"
INSTRUCTOR_EMAIL="nobodyknowme6533@gmail.com"  
STUDENT_EMAIL="jerbawjerbex@gmail.com"
PASSWORD="TestPassword123"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables to store tokens and IDs
ADMIN_TOKEN=""
INSTRUCTOR_TOKEN=""
STUDENT_TOKEN=""
COURSE_ID=""
PAYMENT_ID=""
ENROLLMENT_ID=""

# IMPROVED Function to make API requests
make_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local token=$4
    
    local curl_cmd="curl -s -w \"%{http_code}\" -X $method \"$BASE_URL$endpoint\" -H \"Content-Type: application/json\""
    
    if [ -n "$token" ]; then
        curl_cmd="$curl_cmd -H \"Authorization: Bearer $token\""
    fi
    
    if [ -n "$data" ]; then
        curl_cmd="$curl_cmd -d '$data'"
    fi
    
    # Execute and capture response
    local response=$(eval $curl_cmd)
    
    # Extract HTTP status (last 3 characters)
    local http_status="${response: -3}"
    # Extract JSON response (all but last 3 characters)
    local json_response="${response%???}"
    
    echo "$json_response"
    return $http_status
}

# Function to print test result
print_result() {
    local test_name=$1
    local status=$2
    
    if [ $status -ge 200 ] && [ $status -lt 300 ]; then
        echo -e "${GREEN}✅ $test_name - SUCCESS (HTTP $status)${NC}"
    else
        echo -e "${RED}❌ $test_name - FAILED (HTTP $status)${NC}"
    fi
}

# Function to extract ID from response
extract_id() {
    echo "$1" | grep -o '"id":[^,}]*' | cut -d: -f2 | tr -d '" ' | head -1
}

# Function to extract token from response
extract_token() {
    echo "$1" | grep -o '"token":"[^"]*' | cut -d'"' -f4
}

# Function to wait for verification with manual option
wait_for_verification() {
    echo ""
    echo -e "${YELLOW}⚠️  VERIFICATION REQUIRED${NC}"
    echo "=============================================="
    echo "Emails sent to:"
    echo "- Admin: $ADMIN_EMAIL"
    echo "- Instructor: $INSTRUCTOR_EMAIL" 
    echo "- Student: $STUDENT_EMAIL"
    echo ""
    echo -e "${YELLOW}Options:${NC}"
    echo "1. Check emails and verify normally"
    echo "2. Use manual verification workaround (if emails not received)"
    echo ""
    read -p "Choose option (1 or 2): " option
    
    if [ "$option" == "2" ]; then
        echo ""
        echo -e "${BLUE}🔧 USING MANUAL VERIFICATION WORKAROUND${NC}"
        echo "This requires admin privileges to verify users..."
        return 1
    else
        echo ""
        echo -e "${YELLOW}📧 Please check your emails for verification links${NC}"
        echo "After verifying ALL emails, press Enter to continue..."
        read -p ""
    fi
}

echo ""
echo -e "${BLUE}📋 PHASE 1: USER REGISTRATION & VERIFICATION${NC}"
echo "=============================================="

# Test 1: Health Check
echo "1. Testing Health Check..."
health_response=$(make_request "GET" "/health")
health_status=$?
print_result "Health Check" $health_status

# Test 2: Register Admin User
echo ""
echo "2. Registering Admin User..."
admin_data=$(cat <<EOF
{
    "first_name": "Ermias",
    "last_name": "Abebe",
    "email": "$ADMIN_EMAIL",
    "password": "$PASSWORD",
    "phone": "0965335366",
    "role": "admin"
}
EOF
)
admin_register_response=$(make_request "POST" "/auth/register" "$admin_data")
admin_register_status=$?
print_result "Admin Registration" $admin_register_status

# Test 3: Register Instructor User
echo ""
echo "3. Registering Instructor User..."
instructor_data=$(cat <<EOF
{
    "first_name": "Yordanos",
    "last_name": "Mulugeta",
    "email": "$INSTRUCTOR_EMAIL",
    "password": "$PASSWORD",
    "phone": "0965335366",
    "role": "instructor"
}
EOF
)
instructor_register_response=$(make_request "POST" "/auth/register" "$instructor_data")
instructor_register_status=$?
print_result "Instructor Registration" $instructor_register_status

# Test 4: Register Student User
echo ""
echo "4. Registering Student User..."
student_data=$(cat <<EOF
{
    "first_name": "Eyob",
    "last_name": "Hailu",
    "email": "$STUDENT_EMAIL",
    "password": "$PASSWORD",
    "phone": "0965335366",
    "role": "student"
}
EOF
)
student_register_response=$(make_request "POST" "/auth/register" "$student_data")
student_register_status=$?
print_result "Student Registration" $student_register_status

# Wait for verification
wait_for_verification

echo ""
echo -e "${BLUE}📋 PHASE 2: USER AUTHENTICATION${NC}"
echo "=================================="

# Test 5: Admin Login
echo "5. Admin Login..."
admin_login_data=$(cat <<EOF
{
    "email": "$ADMIN_EMAIL",
    "password": "$PASSWORD"
}
EOF
)
admin_login_response=$(make_request "POST" "/auth/login" "$admin_login_data")
admin_login_status=$?
print_result "Admin Login" $admin_login_status
ADMIN_TOKEN=$(extract_token "$admin_login_response")
if [ -n "$ADMIN_TOKEN" ]; then
    echo "   Admin Token: ${ADMIN_TOKEN:0:30}..."
else
    echo "   Admin Token: NOT RECEIVED"
fi

# Test 6: Instructor Login
echo ""
echo "6. Instructor Login..."
instructor_login_data=$(cat <<EOF
{
    "email": "$INSTRUCTOR_EMAIL",
    "password": "$PASSWORD"
}
EOF
)
instructor_login_response=$(make_request "POST" "/auth/login" "$instructor_login_data")
instructor_login_status=$?
print_result "Instructor Login" $instructor_login_status
INSTRUCTOR_TOKEN=$(extract_token "$instructor_login_response")
if [ -n "$INSTRUCTOR_TOKEN" ]; then
    echo "   Instructor Token: ${INSTRUCTOR_TOKEN:0:30}..."
else
    echo "   Instructor Token: NOT RECEIVED - User likely not verified"
fi

# Test 7: Student Login
echo ""
echo "7. Student Login..."
student_login_data=$(cat <<EOF
{
    "email": "$STUDENT_EMAIL",
    "password": "$PASSWORD"
}
EOF
)
student_login_response=$(make_request "POST" "/auth/login" "$student_login_data")
student_login_status=$?
print_result "Student Login" $student_login_status
STUDENT_TOKEN=$(extract_token "$student_login_response")
if [ -n "$STUDENT_TOKEN" ]; then
    echo "   Student Token: ${STUDENT_TOKEN:0:30}..."
else
    echo "   Student Token: NOT RECEIVED - User likely not verified"
fi

# Check if we can continue with testing
if [ -z "$INSTRUCTOR_TOKEN" ] || [ -z "$STUDENT_TOKEN" ]; then
    echo ""
    echo -e "${RED}🚫 CRITICAL: Instructor and/or Student authentication failed${NC}"
    echo -e "${YELLOW}This is likely because emails were not verified.${NC}"
    echo ""
    echo -e "${BLUE}🛠️  TROUBLESHOOTING OPTIONS:${NC}"
    echo "1. Check email spam folders for verification links"
    echo "2. Use the manual verification workaround below"
    echo "3. Check backend logs for email sending errors"
    echo ""
    
    # Manual verification workaround
    echo -e "${GREEN}🔧 MANUAL VERIFICATION WORKAROUND:${NC}"
    echo "Run these commands to manually verify users (requires admin access):"
    echo ""
    echo "# Verify Instructor"
    echo "curl -X PATCH \\"
    echo "  -H \"Authorization: Bearer $ADMIN_TOKEN\" \\"
    echo "  -H \"Content-Type: application/json\" \\"
    echo "  \"$BASE_URL/admin/users/$INSTRUCTOR_EMAIL/verify\""
    echo ""
    echo "# Verify Student"  
    echo "curl -X PATCH \\"
    echo "  -H \"Authorization: Bearer $ADMIN_TOKEN\" \\"
    echo "  -H \"Content-Type: application/json\" \\"
    echo "  \"$BASE_URL/admin/users/$STUDENT_EMAIL/verify\""
    echo ""
    
    read -p "Press Enter to attempt manual verification and continue testing, or Ctrl+C to stop..."
    
    # Attempt manual verification
    if [ -n "$ADMIN_TOKEN" ]; then
        echo ""
        echo -e "${BLUE}🛠️  Attempting manual verification...${NC}"
        
        # Verify Instructor
        echo "Verifying instructor..."
        verify_instructor_response=$(make_request "PATCH" "/admin/users/$INSTRUCTOR_EMAIL/verify" "" "$ADMIN_TOKEN")
        verify_instructor_status=$?
        
        # Verify Student
        echo "Verifying student..."
        verify_student_response=$(make_request "PATCH" "/admin/users/$STUDENT_EMAIL/verify" "" "$ADMIN_TOKEN")
        verify_student_status=$?
        
        # Try logins again
        echo ""
        echo "Retrying Instructor Login..."
        instructor_login_response=$(make_request "POST" "/auth/login" "$instructor_login_data")
        instructor_login_status=$?
        INSTRUCTOR_TOKEN=$(extract_token "$instructor_login_response")
        
        echo "Retrying Student Login..."
        student_login_response=$(make_request "POST" "/auth/login" "$student_login_data")
        student_login_status=$?
        STUDENT_TOKEN=$(extract_token "$student_login_response")
    fi
fi

# Only continue if we have the necessary tokens
if [ -n "$INSTRUCTOR_TOKEN" ] && [ -n "$STUDENT_TOKEN" ]; then
    echo ""
    echo -e "${BLUE}📋 PHASE 3: COURSE MANAGEMENT${NC}"
    echo "================================="

    # Test 8: Create Course as Instructor
    echo "8. Creating Course as Instructor..."
    course_data=$(cat <<EOF
    {
        "title": "Advanced Web Development",
        "description": "Learn modern web development with React, Node.js, and MongoDB",
        "price": 299.99,
        "category": "Web Development",
        "level": "intermediate",
        "published": true
    }
EOF
    )
    course_response=$(make_request "POST" "/courses" "$course_data" "$INSTRUCTOR_TOKEN")
    course_status=$?
    print_result "Course Creation" $course_status
    COURSE_ID=$(extract_id "$course_response")
    echo "   Course ID: $COURSE_ID"

    # Test 9: Get All Courses (Public)
    echo ""
    echo "9. Getting Public Courses List..."
    courses_response=$(make_request "GET" "/courses")
    courses_status=$?
    print_result "Public Courses" $courses_status

    echo ""
    echo -e "${BLUE}📋 PHASE 4: PAYMENT & ENROLLMENT${NC}"
    echo "=================================="

    # Test 10: Initiate Payment as Student
    echo "10. Initiating Payment..."
    payment_data=$(cat <<EOF
    {
        "course_id": $COURSE_ID
    }
EOF
    )
    payment_response=$(make_request "POST" "/payments/initiate" "$payment_data" "$STUDENT_TOKEN")
    payment_status=$?
    print_result "Payment Initiation" $payment_status
    PAYMENT_ID=$(echo "$payment_response" | grep -o '"payment_id":[^,}]*' | cut -d: -f2 | tr -d ' ')
    echo "   Payment ID: $PAYMENT_ID"

    # Continue with other tests...
else
    echo ""
    echo -e "${RED}🚫 CANNOT CONTINUE TESTING - Missing required authentication tokens${NC}"
    echo "Please ensure all users are verified before running the test script."
fi

# Final Summary
echo ""
echo -e "${GREEN}🎯 TESTING COMPLETE${NC}"
echo "=================="
echo ""
echo -e "${BLUE}📊 TEST SUMMARY:${NC}"
echo "------------------"
echo "Base URL: $BASE_URL"
echo "Admin: $(if [ -n "$ADMIN_TOKEN" ]; then echo "✅ Authenticated"; else echo "❌ Failed"; fi)"
echo "Instructor: $(if [ -n "$INSTRUCTOR_TOKEN" ]; then echo "✅ Authenticated"; else echo "❌ Failed"; fi)"
echo "Student: $(if [ -n "$STUDENT_TOKEN" ]; then echo "✅ Authenticated"; else echo "❌ Failed"; fi)"
echo "Course Created: $(if [ -n "$COURSE_ID" ]; then echo "✅ ID: $COURSE_ID"; else echo "❌ Failed"; fi)"
echo ""
echo -e "${YELLOW}📧 Emails Used:${NC}"
echo "Admin: $ADMIN_EMAIL"
echo "Instructor: $INSTRUCTOR_EMAIL"
echo "Student: $STUDENT_EMAIL"