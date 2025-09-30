package main

import (
	"fmt"
	"learning_hub/handlers"
	"learning_hub/middleware"
	"learning_hub/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=postgres password=1289 dbname=learning_hub port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect with database: " + err.Error())
	}

	// Auto migrate ALL models
	if err := db.AutoMigrate(
		&models.User{},
		&models.Course{},
		&models.Module{},
		&models.Lesson{},
		&models.Enrollment{},
		&models.LessonProgress{},
		&models.Review{},
	); err != nil {
		panic("migration failed: " + err.Error())
	}

	// Initialize handlers
	userHandler := handlers.NewUserHandler(db)
	courseHandler := handlers.NewCourseHandler(db)

	r := gin.Default()

	// API routes group
	api := r.Group("/api")
	{
		// Public routes
		api.GET("/courses", courseHandler.GetCourses)
		api.GET("/courses/:id", courseHandler.GetCourseByID)

		// Auth routes
		api.POST("/register", userHandler.RegisterUser)
		api.POST("/login", userHandler.LoginUser)

		// Protected user routes (all authenticated users)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/profile", userHandler.GetProfile)
			protected.PUT("/profile", userHandler.UpdateProfile)
			protected.GET("/dashboard", courseHandler.GetStudentDashboard)
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

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	fmt.Println("🚀 LearnHub API running on port 8080...")
	if err := r.Run(":8080"); err != nil {
		panic("failed to run the server: " + err.Error())
	}
}
