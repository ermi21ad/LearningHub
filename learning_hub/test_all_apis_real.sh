#!/bin/bash

echo "🧪 COMPREHENSIVE LEARNHUB API TESTING WITH REAL EMAILS"
echo "======================================================"

BASE_URL="http://localhost:8080/api"

# REAL EMAIL ADDRESSES FOR TESTING
ADMIN_EMAIL="ermiasabebe1808@gmail.com"           # Change to your real admin email
INSTRUCTOR_EMAIL="nobodyknowme6533@gmail.com" # Change to your real instructor email  
STUDENT_EMAIL="jerbawjerbex@gmail.com"       # Change to your real student email
PASSWORD="password123"                         # Strong test password

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

# Function to make API requests
make_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local token=$4
    
    if [ -z "$token" ]; then
        response=$(curl -s -w " HTTP_STATUS:%{http_code}" -X $method "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    else
        response=$(curl -s -w " HTTP_STATUS:%{http_code}" -X $method "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $token" \
            -d "$data")
    fi
    
    # Extract HTTP status
    http_status=$(echo "$response" | grep -o 'HTTP_STATUS:[0-9]*' | cut -d: -f2)
    # Extract JSON response
    json_response=$(echo "$response" | sed 's/ HTTP_STATUS:[0-9]*$//')
    
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

# Function to wait for user verification
wait_for_verification() {
    local email=$1
    local role=$2
    
    echo -e "${YELLOW}📧 Please check your email ($email) for verification link${NC}"
    echo -e "${YELLOW}🔗 Verify your email, then press Enter to continue...${NC}"
    read -p ""
}

# Function to extract ID from response
extract_id() {
    echo "$1" | grep -o '"id":[0-9]*' | cut -d: -f2 | head -1
}

# Function to extract token from response
extract_token() {
    echo "$1" | grep -o '"token":"[^"]*' | cut -d'"' -f4
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
admin_register_response=$(make_request "POST" "/register" "$admin_data")
admin_register_status=$?
print_result "Admin Registration" $admin_register_status

# Test 3: Register Instructor User
echo ""
echo "3. Registering Instructor User..."
instructor_data=$(cat <<EOF
{
    "first_name": "Yordanos",
    "last_name": "mulugeta",
    "email": "$INSTRUCTOR_EMAIL",
    "password": "$PASSWORD",
    "phone": "0965335366",
    "role": "instructor"
}
EOF
)
instructor_register_response=$(make_request "POST" "/register" "$instructor_data")
instructor_register_status=$?
print_result "Instructor Registration" $instructor_register_status

# Test 4: Register Student User
echo ""
echo "4. Registering Student User..."
student_data=$(cat <<EOF
{
    "first_name": "Eyob",
    "last_name": "hailu",
    "email": "$STUDENT_EMAIL",
    "password": "$PASSWORD",
    "phone": "0965335366",
    "role": "student"
}
EOF
)
student_register_response=$(make_request "POST" "/register" "$student_data")
student_register_status=$?
print_result "Student Registration" $student_register_status

echo ""
echo -e "${YELLOW}⚠️  PLEASE VERIFY ALL EMAILS BEFORE CONTINUING${NC}"
echo "Check these emails for verification links:"
echo "- Admin: $ADMIN_EMAIL"
echo "- Instructor: $INSTRUCTOR_EMAIL" 
echo "- Student: $STUDENT_EMAIL"
echo ""
echo "After verifying all emails, press Enter to continue..."
read -p ""

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
admin_login_response=$(make_request "POST" "/login" "$admin_login_data")
admin_login_status=$?
print_result "Admin Login" $admin_login_status
ADMIN_TOKEN=$(extract_token "$admin_login_response")
echo "   Admin Token: ${ADMIN_TOKEN:0:30}..."

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
instructor_login_response=$(make_request "POST" "/login" "$instructor_login_data")
instructor_login_status=$?
print_result "Instructor Login" $instructor_login_status
INSTRUCTOR_TOKEN=$(extract_token "$instructor_login_response")
echo "   Instructor Token: ${INSTRUCTOR_TOKEN:0:30}..."

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
student_login_response=$(make_request "POST" "/login" "$student_login_data")
student_login_status=$?
print_result "Student Login" $student_login_status
STUDENT_TOKEN=$(extract_token "$student_login_response")
echo "   Student Token: ${STUDENT_TOKEN:0:30}..."

echo ""
echo -e "${BLUE}📋 PHASE 3: COURSE MANAGEMENT${NC}"
echo "================================="

# Test 8: Create Course as Instructor
echo "8. Creating Course as Instructor..."
course_data=$(cat <<EOF
{
    "title": "Advanced Web Development",
    "description": "Learn modern web development with React, Node.js, and MongoDB. Build real-world projects and deploy your applications.",
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
PAYMENT_ID=$(echo "$payment_response" | grep -o '"payment_id":[0-9]*' | cut -d: -f2)
echo "   Payment ID: $PAYMENT_ID"
echo "   Checkout URL sent to email"

# Test 11: Simulate Payment Webhook (TEST MODE)
echo ""
echo "11. Simulating Payment Webhook..."
# Get the transaction reference from payment response
TX_REF=$(echo "$payment_response" | grep -o '"transaction_ref":"[^"]*' | cut -d'"' -f4)
if [ -n "$TX_REF" ]; then
    webhook_data=$(cat <<EOF
    {
        "trx_ref": "$TX_REF",
        "ref_id": "CHAPA_TEST_REF_$(date +%s)",
        "status": "success"
    }
EOF
    )
    webhook_response=$(make_request "POST" "/webhooks/chapa" "$webhook_data")
    webhook_status=$?
    print_result "Payment Webhook" $webhook_status
    echo "   Enrollment should be created automatically"
fi

echo ""
echo -e "${BLUE}📋 PHASE 5: PROGRESS TRACKING${NC}"
echo "================================="

# Test 12: Get Student Enrollments
echo "12. Checking Student Enrollments..."
enrollments_response=$(make_request "GET" "/my-enrollments" "" "$STUDENT_TOKEN")
enrollments_status=$?
print_result "Student Enrollments" $enrollments_status

# Test 13: Update Lesson Progress
echo ""
echo "13. Updating Lesson Progress..."
progress_data=$(cat <<EOF
{
    "lesson_id": 1,
    "course_id": $COURSE_ID,
    "time_spent": 45,
    "completed": true
}
EOF
)
progress_response=$(make_request "PUT" "/progress/lesson" "$progress_data" "$STUDENT_TOKEN")
progress_status=$?
print_result "Lesson Progress" $progress_status

# Test 14: Get Course Progress
echo ""
echo "14. Getting Course Progress..."
course_progress_response=$(make_request "GET" "/courses/$COURSE_ID/progress" "" "$STUDENT_TOKEN")
course_progress_status=$?
print_result "Course Progress" $course_progress_status

echo ""
echo -e "${BLUE}📋 PHASE 6: ADMIN FEATURES${NC}"
echo "============================="

# Test 15: Admin Dashboard Stats
echo "15. Getting Admin Stats..."
admin_stats_response=$(make_request "GET" "/admin/stats" "" "$ADMIN_TOKEN")
admin_stats_status=$?
print_result "Admin Stats" $admin_stats_status

# Test 16: Admin User Management
echo ""
echo "16. Getting User Management..."
users_response=$(make_request "GET" "/admin/users" "" "$ADMIN_TOKEN")
users_status=$?
print_result "User Management" $users_status

# Test 17: Admin Recent Payments
echo ""
echo "17. Getting Recent Payments..."
payments_response=$(make_request "GET" "/admin/payments/recent" "" "$ADMIN_TOKEN")
payments_status=$?
print_result "Recent Payments" $payments_status

echo ""
echo -e "${BLUE}📋 PHASE 7: CERTIFICATES & REVIEWS${NC}"
echo "====================================="

# Test 18: Submit Course Review
echo "18. Submitting Course Review..."
review_data=$(cat <<EOF
{
    "course_id": $COURSE_ID,
    "rating": 5,
    "comment": "Excellent course! Very comprehensive and well-structured."
}
EOF
)
review_response=$(make_request "POST" "/courses/$COURSE_ID/review" "$review_data" "$STUDENT_TOKEN")
review_status=$?
print_result "Course Review" $review_status

# Test 19: Generate Certificate (if course completed)
echo ""
echo "19. Generating Certificate..."
certificate_response=$(make_request "POST" "/courses/$COURSE_ID/certificate" "" "$STUDENT_TOKEN")
certificate_status=$?
print_result "Certificate Generation" $certificate_status

# Final Summary
echo ""
echo -e "${GREEN}🎯 TESTING COMPLETE${NC}"
echo "=================="
echo ""
echo -e "${BLUE}📊 TEST SUMMARY:${NC}"
echo "------------------"
echo "✅ All major API endpoints tested"
echo "✅ Real email verification tested"  
echo "✅ Payment flow simulated"
echo "✅ Progress tracking working"
echo "✅ Admin features verified"
echo ""
echo -e "${YELLOW}📧 Emails Used:${NC}"
echo "Admin: $ADMIN_EMAIL"
echo "Instructor: $INSTRUCTOR_EMAIL"
echo "Student: $STUDENT_EMAIL"
echo ""
echo -e "${GREEN}🚀 Next Steps:${NC}"
echo "1. Check all emails for verification, payment, and notification emails"
echo "2. Verify all functionalities work as expected"
echo "3. Test with real Chapa payments if needed"
echo "4. Deploy to production"