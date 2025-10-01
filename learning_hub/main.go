package main

import (
	"fmt"
	"learning_hub/handlers"
	"learning_hub/middleware"
	"learning_hub/models"
	"learning_hub/pkg/chapa"
	"learning_hub/pkg/config"
	"learning_hub/pkg/fileupload"
	"log"

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
	if err := db.AutoMigrate(
		&models.User{},
		&models.Course{},
		&models.Module{},
		&models.Lesson{},
		&models.Enrollment{},
		&models.Payment{},
		&models.LessonProgress{},
		&models.Review{},
	); err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Initialize handlers
	userHandler := handlers.NewUserHandler(db)
	courseHandler := handlers.NewCourseHandler(db)
	uploadHandler := handlers.NewUploadHandler(db)
	paymentHandler := handlers.NewPaymentHandler(db)

	r := gin.Default()

	// API routes group
	// In your main.go, update the routes section:

	// API routes group
	api := r.Group("/api")
	{
		// Public routes
		api.GET("/courses", courseHandler.GetCourses)
		api.GET("/courses/:id", courseHandler.GetCourseByID)
		api.POST("/register", userHandler.RegisterUser)
		api.POST("/login", userHandler.LoginUser)
		api.POST("/upload", uploadHandler.UploadFile)

		// Webhook and success page (public - no auth needed)
		api.POST("/webhooks/chapa", paymentHandler.HandlePaymentCallback)
		api.GET("/payment/success", paymentHandler.PaymentSuccess)

		// Protected routes (require authentication)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/profile", userHandler.GetProfile)
			protected.PUT("/profile", userHandler.UpdateProfile)
			protected.GET("/dashboard", courseHandler.GetStudentDashboard)
			protected.GET("/my-payments", paymentHandler.GetUserPayments)
			protected.GET("/my-enrollments", userHandler.GetUserEnrollments)

			// MOVE THESE PAYMENT ROUTES UNDER PROTECTED:
			protected.POST("/payments/initiate", paymentHandler.InitiatePayment)
			protected.GET("/payments/status/:id", paymentHandler.GetPaymentStatus)
		}

		// Student-only routes
		student := api.Group("/")
		student.Use(middleware.AuthMiddleware(), middleware.StudentOnly())
		{
			student.POST("/courses/:id/enroll", courseHandler.EnrollCourse)
			student.GET("/my-courses", courseHandler.GetStudentCourses)
			student.PUT("/progress/lesson", courseHandler.UpdateLessonProgress)
			student.GET("/courses/:id/progress", courseHandler.GetCourseProgress)
			student.POST("/courses/:id/review", courseHandler.SubmitCourseReview)
		}

		// Instructor-only routes
		instructor := api.Group("/")
		instructor.Use(middleware.AuthMiddleware(), middleware.InstructorOnly())
		{
			instructor.POST("/courses", courseHandler.CreateCourse)
			instructor.PUT("/courses/:id", courseHandler.UpdateCourse)
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
	if err := r.Run(serverAddr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
