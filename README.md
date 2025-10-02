name: 🎓 LearningHub Backend
description: >
  LearningHub is a complete backend system for an e-learning platform built 
  with Golang (Gin Framework), PostgreSQL, and GORM. It supports secure 
  authentication, course management, content delivery, assessment system, 
  payments (Chapa), student progress tracking, certificates, email notifications, 
  and admin dashboards.

major_updates:
  - Advanced Content Delivery System:
      - Lesson Management (CRUD for modules/lessons)
      - Rich Content Support (Videos, PDFs, Docs, Presentations, Text)
      - Progress Tracking (time-based & completion status)
      - Curriculum Hierarchy with prerequisites
      - File Storage: Local & S3-compatible
  - Comprehensive Assessment System:
      - Quiz Engine (MCQ, T/F, Short Answer, Coding)
      - Assignment System (File upload & text submissions with grading)
      - Automatic Grading & Real-time scoring
      - Attempt Management (time limits, retries)
      - Instructor Analytics (performance insights)

project_structure:
  handlers:
    - user_handler.go: User authentication & management
    - course_handler.go: Course CRUD & enrollment
    - lesson.go: Lesson management & progress tracking
    - assessment.go: Quizzes & assignments
    - payment.go: Chapa payment integration
    - progress.go: Tracking & certificates
    - admin.go: Admin dashboard & user management
    - upload.go: File upload handling
  middleware:
    - auth.go: JWT validation
    - instructor.go: Instructor role middleware
    - student.go: Student role middleware
    - admin.go: Admin role middleware
  models:
    - user.go: User accounts & profiles
    - course.go: Courses, modules, lessons
    - assessment.go: Quizzes, assignments, attempts
    - payment.go: Payment transactions
    - progress.go: Progress tracking
    - certificate.go: Certificate generation
  pkg:
    - config: Configuration management
    - jwt: Token handling
    - email: SMTP email service
    - chapa: Payment gateway integration
    - fileupload: Secure file uploads
    - validation: Input validation
    - storage: File storage (local/S3)
  uploads:
    - images/: Course thumbnails
    - videos/: Lesson videos
    - documents/: PDFs & materials
    - assignments/: Student submissions
  files:
    - main.go
    - .env
    - go.mod / go.sum
    - README.md
    - test_*.sh

features:
  authentication:
    - JWT-based authentication
    - Role-based access (Student, Instructor, Admin)
    - Email verification
    - Password reset
  course_management:
    - Course creation & categorization
    - Module-based curriculum
    - Enrollment with payment
    - Reviews & ratings
    - Instructor analytics
  content_delivery:
    - Lesson CRUD
    - Multi-format content (Video, PDF, Docs)
    - Real-time progress & completion tracking
  assessment:
    - Quizzes: MCQ, T/F, Short answer, coding
    - Automatic grading & scoring
    - Assignments: File & text submissions
    - Instructor grading + feedback
    - Due date management
  payments:
    - Chapa integration
    - Secure initiation & verification
    - Webhook handling
    - Payment history & receipts
  progress_certificates:
    - Progress tracking
    - Auto certificate generation
    - Verification system
  admin_management:
    - User management & roles
    - Platform statistics
    - System configuration

api_endpoints:
  authentication:
    - POST /api/register
    - POST /api/login
    - GET /api/profile
    - PUT /api/profile
  courses:
    - GET /api/courses
    - POST /api/courses
    - PUT /api/courses/:id
    - POST /api/courses/:id/enroll
    - GET /api/courses/:id
  lessons:
    - POST /api/lessons
    - GET /api/lessons/:id
    - PUT /api/lessons/:id
    - PUT /api/lessons/:id/progress
    - GET /api/lessons/module/:moduleId
    - GET /api/lessons/:id/analytics
  assessments:
    quizzes:
      - POST /api/assessments/quizzes
      - POST /api/assessments/quizzes/:quizId/attempt
      - POST /api/assessments/attempts/:attemptId/answer
      - POST /api/assessments/attempts/:attemptId/complete
    assignments:
      - POST /api/assessments/assignments
      - POST /api/assessments/assignments/:assignmentId/submit
      - POST /api/assessments/submissions/:submissionId/grade
  payments:
    - POST /api/payments/initiate
    - GET /api/payments/status/:id
    - POST /api/webhooks/chapa
  progress_certificates:
    - PUT /api/progress/lesson
    - GET /api/courses/:id/progress
    - POST /api/courses/:id/certificate
    - GET /api/certificates/:id
  admin:
    - GET /api/admin/stats
    - GET /api/admin/users
    - PUT /api/admin/users/:id/role

workflow:
  students: [Browse & Enroll, Learn, Assess, Track, Review]
  instructors: [Create, Deliver, Assess, Monitor, Engage]
  admins: [Manage, Monitor, Analyze, Support]

quick_start:
  prerequisites:
    - Go 1.19+
    - PostgreSQL 12+
    - SMTP server
    - Chapa account
  setup:
    - Clone repo & install deps
    - Configure .env
    - Run migrations
    - Start server: go run main.go
  testing:
    - ./test_all_apis_real.sh
    - ./test_payment_flow.sh
    - ./test_content_delivery.sh
    - ./test_assessment_system.sh

architecture:
  flow: >
    Frontend Clients → API Gateway → LearningHub Backend → PostgreSQL
                      → File Storage (Local/S3)
                      → Email Service (SMTP)
                      → Payment Gateway (Chapa)
                      → Analytics & Monitoring

stack:
  backend: Golang (Gin, GORM)
  database: PostgreSQL
  auth: JWT with roles
  storage: Local + S3
  payments: Chapa
  email: SMTP
  testing: Bash test scripts

production_ready:
  - Security: JWT auth, validation, secure uploads
  - Performance: Indexed DB, efficient queries
  - Scalability: Modular design, storage flexibility
  - Monitoring: Health checks, error logging
  - Docs: Full API documentation
  - Testing: End-to-end coverage

contact:
  name: Ermias Abebe
  email: ermiasabebezewdie@gmail.com
  portfolio: "https://ermias-abebe-portfolio.vercel.app/"
  github: "https://github.com/ermi21ad"
  linkedin: "https://linkedin.com/in/ermias-abebe"
