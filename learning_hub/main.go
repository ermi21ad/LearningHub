package main

import (
	"fmt"
	"learning_hub/handlers"
	"learning_hub/middleware"
	"learning_hub/models"
	"learning_hub/pkg/chapa"
	"learning_hub/pkg/config"
	"learning_hub/pkg/email"
	"learning_hub/pkg/fileupload"
	"learning_hub/pkg/validation"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}
	email.Init(cfg)

	fmt.Printf("🚀 Starting LearnHub API in %s mode...\n", cfg.ServerEnv)

	// Initialize file upload with config
	fileupload.Init(cfg)

	// Initialize Chapa
	if err := chapa.Init(cfg); err != nil {
		log.Fatal("Failed to initialize Chapa:", err)
	}

	// Test Chapa connection
	if err := chapa.TestConnection(); err != nil {
		log.Printf("Warning: Chapa connection test failed: %v", err)
	} else {
		fmt.Println("✅ Chapa connected successfully")
	}

	// Database connection using config
	db, err := gorm.Open(postgres.Open(cfg.GetDBDSN()), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate models
	// Add to your existing AutoMigrate
	if err := db.AutoMigrate(
		&models.User{},
		&models.Course{},
		&models.Module{},
		&models.Lesson{},
		&models.Enrollment{},
		&models.Payment{},
		&models.LessonProgress{},
		&models.Certificate{},
		&models.Review{},
		&models.Quiz{},
		&models.QuizQuestion{},
		&models.QuizAttempt{},
		&models.QuizAnswer{},
		&models.Assignment{},
		&models.AssignmentSubmission{},
	); err != nil {
		log.Fatal("Migration failed:", err)
	}
	fmt.Println("✅ Database migrations completed successfully")

	// Initialize handlers
	userHandler := handlers.NewUserHandler(db)
	courseHandler := handlers.NewCourseHandler(db)
	uploadHandler := handlers.NewUploadHandler(db)
	paymentHandler := handlers.NewPaymentHandler(db)
	adminHandler := handlers.NewAdminHandler(db)
	progressHandler := handlers.NewProgressHandler(db)
	lessonHandler := handlers.NewLessonHandler(db)
	assessmentHandler := handlers.NewAssessmentHandler(db)

	r := gin.Default()

	// API routes group
	api := r.Group("/api")
	{
		// Public routes
		api.GET("/courses", courseHandler.GetCourses)
		api.GET("/courses/:id", courseHandler.GetCourseByID)
		api.POST("/register", userHandler.RegisterUser)
		api.POST("/login", userHandler.LoginUser)
		api.POST("/upload", uploadHandler.UploadFile)

		// Verification & Password routes
		api.GET("/verify-email", userHandler.VerifyEmail)
		api.POST("/resend-verification", userHandler.ResendVerificationEmail)
		api.POST("/forgot-password", userHandler.ForgotPassword)
		api.POST("/reset-password", userHandler.ResetPassword)
		api.GET("/validate-reset-token", userHandler.ValidateResetToken)

		// Public certificate verification
		api.GET("/verify-certificate", progressHandler.VerifyCertificate)

		// Public domain list
		api.GET("/allowed-email-domains", func(c *gin.Context) {
			domains := validation.GetAllowedDomains()
			c.JSON(http.StatusOK, gin.H{
				"allowed_domains": domains,
				"count":           len(domains),
				"message":         "These are the supported email providers for registration",
			})
		})

		// Payment webhooks (public)
		api.POST("/webhooks/chapa", paymentHandler.HandlePaymentCallback)
		api.GET("/payment/success", paymentHandler.PaymentSuccess)

		// Protected routes (require authentication)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/profile", userHandler.GetProfile)
			protected.PUT("/profile", userHandler.UpdateProfile)
			protected.GET("/dashboard", progressHandler.GetStudentDashboard)
			protected.GET("/my-payments", paymentHandler.GetUserPayments)
			protected.GET("/my-enrollments", userHandler.GetUserEnrollments)
			protected.POST("/payments/initiate", paymentHandler.InitiatePayment)
			protected.GET("/payments/status/:id", paymentHandler.GetPaymentStatus)
		}

		// Student-only routes
		student := api.Group("/")
		student.Use(middleware.AuthMiddleware(), middleware.StudentOnly())
		{
			student.POST("/courses/:id/enroll", courseHandler.EnrollCourse)
			student.GET("/my-courses", courseHandler.GetStudentCourses)
			student.PUT("/progress/lesson", progressHandler.UpdateLessonProgress)
			student.GET("/courses/:id/progress", progressHandler.GetCourseProgress)
			student.POST("/courses/:id/review", courseHandler.SubmitCourseReview)
			student.GET("/courses/:id/reviews", courseHandler.GetCourseReviews)
			student.POST("/courses/:id/certificate", progressHandler.GenerateCertificate)
			student.GET("/certificates/:id", progressHandler.GetCertificate)
		}

		// Instructor-only routes
		instructor := api.Group("/")
		instructor.Use(middleware.AuthMiddleware(), middleware.InstructorOnly())
		{
			instructor.POST("/courses", courseHandler.CreateCourse)
			instructor.PUT("/courses/:id", courseHandler.UpdateCourse)
			instructor.DELETE("/courses/:id", courseHandler.DeleteCourse)
			instructor.GET("/instructor/courses", courseHandler.GetInstructorCourses)
			instructor.POST("/courses/:id/modules", courseHandler.CreateModule)
		}

		// Admin-only routes
		admin := api.Group("/")
		admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
		{
			admin.GET("/admin/stats", adminHandler.AdminStats)
			admin.GET("/admin/payments/recent", adminHandler.GetRecentPayments)
			admin.GET("/admin/enrollments/recent", adminHandler.GetRecentEnrollments)
			admin.GET("/admin/courses/:id/analytics", adminHandler.GetCourseAnalytics)
			admin.GET("/admin/users", adminHandler.GetUserManagement)
			admin.PUT("/admin/users/:id/role", adminHandler.UpdateUserRole)
			admin.DELETE("/admin/users/:id", adminHandler.DeleteUser)
			admin.GET("/admin/email-domains", adminHandler.GetEmailDomains)
			admin.POST("/admin/email-domains", adminHandler.AddEmailDomain)
			admin.DELETE("/admin/email-domains/:domain", adminHandler.RemoveEmailDomain)
		}

		// Lesson routes
		lessonRoutes := api.Group("/lessons")
		{
			lessonRoutes.POST("", middleware.AuthMiddleware(), middleware.InstructorOnly(), lessonHandler.CreateLesson)
			lessonRoutes.GET("/:id", middleware.AuthMiddleware(), lessonHandler.GetLesson)
			lessonRoutes.PUT("/:id", middleware.AuthMiddleware(), middleware.InstructorOnly(), lessonHandler.UpdateLesson)
			lessonRoutes.DELETE("/:id", middleware.AuthMiddleware(), middleware.InstructorOnly(), lessonHandler.DeleteLesson)
			lessonRoutes.PUT("/:id/progress", middleware.AuthMiddleware(), lessonHandler.UpdateLessonProgress)
			lessonRoutes.GET("/module/:moduleId", middleware.AuthMiddleware(), lessonHandler.GetModuleLessons)
			lessonRoutes.GET("/:id/analytics", middleware.AuthMiddleware(), middleware.InstructorOnly(), lessonHandler.GetLessonAnalytics)
		}
		assessmentRoutes := api.Group("/assessments")
		{
			// Quiz routes
			assessmentRoutes.POST("/quizzes", middleware.AuthMiddleware(), middleware.InstructorOnly(), assessmentHandler.CreateQuiz)
			assessmentRoutes.POST("/quizzes/:quizId/attempt", middleware.AuthMiddleware(), assessmentHandler.StartQuizAttempt)
			assessmentRoutes.POST("/attempts/:attemptId/answer", middleware.AuthMiddleware(), assessmentHandler.SubmitQuizAnswer)
			assessmentRoutes.POST("/attempts/:attemptId/complete", middleware.AuthMiddleware(), assessmentHandler.CompleteQuizAttempt)
			assessmentRoutes.GET("/quizzes/:quizId/attempts", middleware.AuthMiddleware(), assessmentHandler.GetStudentQuizAttempts)

			// Assignment routes
			assessmentRoutes.POST("/assignments", middleware.AuthMiddleware(), middleware.InstructorOnly(), assessmentHandler.CreateAssignment)
			assessmentRoutes.POST("/assignments/:assignmentId/submit", middleware.AuthMiddleware(), assessmentHandler.SubmitAssignment)
			assessmentRoutes.POST("/submissions/:submissionId/grade", middleware.AuthMiddleware(), middleware.InstructorOnly(), assessmentHandler.GradeAssignment)
			assessmentRoutes.GET("/assignments/:assignmentId/submissions", middleware.AuthMiddleware(), assessmentHandler.GetStudentAssignmentSubmissions)

			// Instructor analytics routes
			assessmentRoutes.GET("/quizzes/:quizId/all-attempts", middleware.AuthMiddleware(), middleware.InstructorOnly(), assessmentHandler.GetQuizAttempts)
			assessmentRoutes.GET("/assignments/:assignmentId/all-submissions", middleware.AuthMiddleware(), middleware.InstructorOnly(), assessmentHandler.GetAssignmentSubmissions)
		}
	}

	// File serving route for uploaded files
	r.GET("/uploads/:type/:filename", uploadHandler.ServeFile)

	// Health check route
	r.GET("/health", func(c *gin.Context) {
		chapaStatus := "connected"
		if err := chapa.TestConnection(); err != nil {
			chapaStatus = "disconnected"
		}

		c.JSON(200, gin.H{
			"status":           "healthy",
			"environment":      cfg.ServerEnv,
			"chapa":            chapaStatus,
			"payment_provider": cfg.GetPaymentProvider(),
		})
	})

	// Start server
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	fmt.Printf("📚 LearnHub API running on port %s...\n", cfg.ServerPort)
	fmt.Printf("💳 Chapa payment integration: ENABLED\n")

	// Create some sample data on startup
	createSampleData(db)

	if err := r.Run(serverAddr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// createSampleData creates initial sample data for testing
func createSampleData(db *gorm.DB) {
	fmt.Println("📝 Creating sample data...")

	fmt.Println("✅ Sample data ready for testing")
}
