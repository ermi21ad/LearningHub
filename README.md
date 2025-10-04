# 🎓 LearningHub Backend

LearningHub is a **backend system for an e-learning platform** built with **Golang (Gin Framework)**, **PostgreSQL**, and **GORM**.
It supports **secure authentication, course management, payments (Chapa), student progress tracking, certificates, email notifications, and admin dashboards**.

---

## 📂 Project Structure

```
learning_hub/
│── handlers/          # API route handlers (controllers)
│── middleware/        # JWT auth, role-based access, logging
│── models/            # Database models (GORM)
│── pkg/               # Utility packages (email, JWT, file upload, payments)
│── uploads/           # Uploaded files (images, videos, PDFs)
│── .env               # Environment variables
│── .gitignore
│── go.mod / go.sum    # Dependencies
│── main.go            # Application entry point
│── README.md
│── test_all_apis_real.sh      # End-to-end testing
│── test_payment_flow.sh       # Payment test script
│── test_webhook.sh            # Webhook test script
```

---

## 🚀 Features

* **🔐 Authentication & Authorization** (JWT, role-based access, password reset)
* **📧 Email Notifications & Verification** (SMTP integration, email verification, password reset)
* **📚 Course Management** (create, update, enroll, reviews, categories)
* **💳 Payment Integration** with Chapa (initiate, verify, webhooks)
* **📊 Progress Tracking** (lesson completion, certificates, dashboards)
* **👨‍💼 Admin Tools** (user management, course analytics, payment reports)
* **🛠️ Utilities** (file upload, health check, allowed email domains)

---
## 📧 Email Notifications & Verification

The platform integrates **SMTP-based email notifications** for user communication and security flows.

* **Email Verification:**

  * New users receive an email with a verification link.
  * Only verified accounts can access full platform features.
* **Password Reset:**

  * Forgot password flow sends a secure reset token via email.
  * Token must be validated before setting a new password.
* **System Notifications:**

  * Confirmation emails for enrollments and payments.
  * Admin and instructors receive alerts for new activities.

### Email APIs

* `GET /api/verify-email` → Verify email via token
* `POST /api/resend-verification` → Resend verification email
* `POST /api/forgot-password` → Request reset link
* `GET /api/validate-reset-token` → Validate reset token
* `POST /api/reset-password` → Reset password
---
## 🔐 Authentication & Authorization

The platform uses **JWT-based authentication** with role-based access control to secure user operations.

* **JWT Authentication:**

  * Each login generates a signed JWT token.
  * Tokens are required for all protected routes.
* **Role-Based Access:**

  * Users have roles (Admin, Instructor, Student).
  * Role-based middleware restricts actions (e.g., only instructors can create courses).
* **Password Reset:**

  * Users can request password reset links via email.
  * Secure tokens ensure safe password updates.

### Auth APIs

* `POST /api/register` → Register new user
* `POST /api/login` → Login & issue JWT
* `GET /api/profile` → Get user profile
* `PUT /api/profile` → Update profile

---

## 📚 Course Management

Comprehensive tools for creating, managing, and engaging with courses.

* **Course Creation & Update:**

  * Instructors can create, edit, and categorize courses.
* **Enrollment:**

  * Students enroll in courses with or without payment.
* **Reviews & Ratings:**

  * Students can leave feedback for quality assurance.
* **Categories & Search:**

  * Courses can be grouped and filtered for easier discovery.

### Course APIs

* `GET /api/courses` → List all courses
* `POST /api/courses` → Create course *(Instructor only)*
* `PUT /api/courses/:id` → Update course
* `POST /api/courses/:id/enroll` → Enroll student

---

## 💳 Payment Integration with Chapa

The system integrates with **Chapa Payments** for seamless transactions.

* **Payment Initiation:**

  * Students initiate payments for paid courses.
* **Payment Verification:**

  * Server validates payment status with Chapa API.
* **Webhooks:**

  * Real-time notifications ensure secure transaction updates.

### Payment APIs

* `POST /api/payments/initiate` → Start a payment
* `GET /api/payments/status/:id` → Verify payment status
* `POST /api/webhooks/chapa` → Handle Chapa webhook

---

## 📊 Progress Tracking

Students and instructors can monitor progress and achievements.

* **Lesson Completion:**

  * Each lesson marked as completed is stored in DB.
* **Dashboards:**

  * Students see course progress visually.
  * Instructors view student performance.
* **Certificates:**

  * Auto-generated on course completion.
  * Certificates can be downloaded or verified.

### Progress APIs

* `PUT /api/progress/lesson` → Update lesson progress
* `POST /api/courses/:id/certificate` → Generate certificate
* `GET /api/certificates/:id` → Fetch certificate

---

## 👨‍💼 Admin Tools

Powerful tools for admins to manage the platform.

* **User Management:**

  * Create, update, or deactivate users.
  * Assign roles (Admin, Instructor, Student).
* **Course Analytics:**

  * Insights into most popular courses, enrollments, revenue.
* **Payment Reports:**

  * Track successful/failed transactions.

### Admin APIs

* `GET /api/admin/stats` → Get platform stats
* `GET /api/admin/users` → List all users
* `PUT /api/admin/users/:id/role` → Update user role

---

## 🛠️ Utilities

Helper features to improve system usability and security.

* **File Uploads:**

  * Supports video, PDFs, and images.
  * Stored in `uploads/` with unique naming.
* **Health Check:**

  * Endpoint to confirm API is running.
* **Allowed Email Domains:**

  * Restricts registration to trusted domains.

### Utility APIs

* `POST /api/upload` → Upload file
* `GET /api/health` → Check API health
* (Config) Restrict user registration domain



---

## 🌐 API Endpoints (Highlights)

### 🔐 Authentication

* `POST /api/register` → Register new user
* `POST /api/login` → Login & get token
* `GET /api/profile` → Get logged-in user profile
* `PUT /api/profile` → Update profile

### 📚 Courses

* `GET /api/courses` → Public list of courses
* `POST /api/courses` → Create course *(Instructor only)*
* `POST /api/courses/:id/enroll` → Enroll in a course *(Student only)*

### 💳 Payments

* `POST /api/payments/initiate` → Start payment
* `GET /api/payments/status/:id` → Check payment status
* `POST /api/webhooks/chapa` → Chapa webhook

### 📊 Progress & Certificates

* `PUT /api/progress/lesson` → Update lesson progress
* `POST /api/courses/:id/certificate` → Generate certificate
* `GET /api/certificates/:id` → Get student certificate

### 👨‍💼 Admin

* `GET /api/admin/stats` → Platform stats
* `GET /api/admin/users` → Manage users
* `PUT /api/admin/users/:id/role` → Assign roles

---

## 🎯 Sample Email Flow

**Resend Verification Email**

```http
POST http://localhost:8080/api/resend-verification
Content-Type: application/json

{
  "email": "student@example.com"
}
```

**Forgot Password**

```http
POST http://localhost:8080/api/forgot-password
Content-Type: application/json

{
  "email": "student@example.com"
}
```

**Reset Password**

```http
POST http://localhost:8080/api/reset-password
Content-Type: application/json

{
  "token": "VALID_RESET_TOKEN",
  "new_password": "newSecurePass123"
}
```

---

## 📋 Testing

Run provided scripts for full coverage:

```bash
./test_all_apis_real.sh       # Test all major APIs
./test_payment_flow.sh        # Test Chapa payment integration
./test_webhook.sh             # Test webhook handling
```

Or test step by step in **Postman**, following the structured API sequence.

---

## 📬 Contact

👤 **Ermias Abebe**
🔗 LinkedIn: [ermias-abebe-zewdie](https://linkedin.com/in/ermias-abebe-zewdie)
💻 GitHub: [@ermi21ad](https://github.com/ermi21ad)

---
