package handlers

import (
	"fmt"
	"learning_hub/models"
	"learning_hub/pkg/chapa"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	db *gorm.DB
}

func NewPaymentHandler(db *gorm.DB) *PaymentHandler {
	return &PaymentHandler{db: db}
}

// InitiatePayment handles course purchase and payment initiation
// In the InitiatePayment function, add test mode handling:
func (h *PaymentHandler) InitiatePayment(c *gin.Context) {
	var request struct {
		CourseID uint `json:"course_id" binding:"required"`
	}

	// Bind and validate request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Get user from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get course details
	var course models.Course
	if err := h.db.First(&course, request.CourseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch course"})
		return
	}

	// Check if user is already enrolled
	var existingEnrollment models.Enrollment
	err := h.db.Where("user_id = ? AND course_id = ?", userID, request.CourseID).First(&existingEnrollment).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "You are already enrolled in this course"})
		return
	}

	// Get user details for payment
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user details"})
		return
	}

	// Generate unique transaction reference
	txRef := fmt.Sprintf("learnhub-%d-%s", time.Now().Unix(), generateRandomString(8))

	// TEST MODE: If using test keys, simulate payment
	if strings.Contains(chapa.GetSecretKey(), "test") {
		fmt.Println("🔧 TEST MODE: Simulating payment flow")

		// Create payment record in database
		payment := models.Payment{
			UserID:     user.ID,
			CourseID:   course.ID,
			Amount:     course.Price,
			Currency:   "ETB",
			ChapaTxRef: txRef,
			Status:     models.PaymentStatusSuccess, // Simulate success in test mode
		}

		if err := h.db.Create(&payment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment record"})
			return
		}

		// Create enrollment
		enrollment := models.Enrollment{
			UserID:     user.ID,
			CourseID:   course.ID,
			PaymentID:  &payment.ID,
			IsActive:   true,
			Progress:   0,
			EnrolledAt: time.Now(),
		}

		if err := h.db.Create(&enrollment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create enrollment"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":         "TEST MODE: Payment completed successfully",
			"checkout_url":    "https://chapa.co/test-mode",
			"transaction_ref": txRef,
			"payment_id":      payment.ID,
			"test_mode":       true,
		})
		return
	}

	// REAL MODE: Use actual Chapa API
	paymentReq := &chapa.PaymentRequest{
		Amount:      fmt.Sprintf("%.2f", course.Price),
		Currency:    "ETB",
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		PhoneNumber: user.Phone,
		TxRef:       txRef,
		CallbackURL: "https://webhook.site/f661bb23-ccc5-478f-8c9e-c835551834c6",
		ReturnURL:   "http://localhost:8080/api/payment/success",
		Customization: chapa.Customization{
			Title:       "LearnHub", // Shortened to meet 16 char limit
			Description: fmt.Sprintf("Pay for %s", course.Title),
		},
		Meta: map[string]interface{}{
			"user_id":   userID,
			"course_id": course.ID,
		},
	}

	// Initialize payment with Chapa
	paymentResp, err := chapa.InitializePayment(paymentReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to initialize payment",
			"details": err.Error(),
		})
		return
	}

	// Create payment record in database
	payment := models.Payment{
		UserID:     user.ID,
		CourseID:   course.ID,
		Amount:     course.Price,
		Currency:   "ETB",
		ChapaTxRef: txRef,
		Status:     models.PaymentStatusPending,
	}

	if err := h.db.Create(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment record"})
		return
	}

	// Return payment URL to frontend
	c.JSON(http.StatusOK, gin.H{
		"message":         "Payment initialized successfully",
		"checkout_url":    paymentResp.Data.CheckoutURL,
		"transaction_ref": txRef,
		"payment_id":      payment.ID,
	})
}

// HandlePaymentCallback handles Chapa webhook callbacks
// HandlePaymentCallback handles Chapa webhook callbacks
func (h *PaymentHandler) HandlePaymentCallback(c *gin.Context) {
	var webhookPayload chapa.WebhookPayload

	// Bind webhook payload
	if err := c.ShouldBindJSON(&webhookPayload); err != nil {
		fmt.Printf("❌ Webhook bind error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	fmt.Printf("🔔 Webhook received: %+v\n", webhookPayload)

	// Find payment by transaction reference
	var payment models.Payment
	if err := h.db.Where("chapa_tx_ref = ?", webhookPayload.TxRef).First(&payment).Error; err != nil {
		fmt.Printf("❌ Payment not found for tx_ref: %s\n", webhookPayload.TxRef)
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	fmt.Printf("🔍 Found payment: ID=%d, Status=%s\n", payment.ID, payment.Status)

	// If webhook status is success, update payment and create enrollment
	if webhookPayload.Status == "success" {
		// Update payment status
		payment.Status = models.PaymentStatusSuccess
		payment.ChapaRefID = webhookPayload.RefID

		if err := h.db.Save(&payment).Error; err != nil {
			fmt.Printf("❌ Failed to update payment: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment"})
			return
		}

		fmt.Printf("✅ Payment updated to success: ID=%d\n", payment.ID)

		// Check if enrollment already exists
		var existingEnrollment models.Enrollment
		err := h.db.Where("user_id = ? AND course_id = ?", payment.UserID, payment.CourseID).First(&existingEnrollment).Error

		if err != nil {
			// Create enrollment if it doesn't exist
			enrollment := models.Enrollment{
				UserID:     payment.UserID,
				CourseID:   payment.CourseID,
				PaymentID:  &payment.ID,
				IsActive:   true,
				Progress:   0,
				EnrolledAt: time.Now(),
			}

			if err := h.db.Create(&enrollment).Error; err != nil {
				fmt.Printf("❌ Failed to create enrollment: %v\n", err)
				// Don't return error - we still want to acknowledge the webhook
			} else {
				fmt.Printf("✅ Enrollment created: UserID=%d, CourseID=%d\n", payment.UserID, payment.CourseID)
			}
		} else {
			fmt.Printf("ℹ️ Enrollment already exists: ID=%d\n", existingEnrollment.ID)
		}
	} else {
		// Payment failed
		payment.Status = models.PaymentStatusFailed
		h.db.Save(&payment)
		fmt.Printf("❌ Payment failed: %s\n", webhookPayload.TxRef)
	}

	// Always return success to Chapa
	c.JSON(http.StatusOK, gin.H{"status": "webhook processed successfully"})
}

// GetPaymentStatus checks the status of a payment
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	paymentID := c.Param("id")

	var payment models.Payment
	if err := h.db.Preload("User").Preload("Course").First(&payment, paymentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payment"})
		return
	}

	// Verify with Chapa for latest status (optional)
	verifyResp, err := chapa.VerifyPayment(payment.ChapaTxRef)
	if err == nil {
		// Update local status if different
		if verifyResp.Data.Status == "success" && payment.Status != models.PaymentStatusSuccess {
			payment.Status = models.PaymentStatusSuccess
			h.db.Save(&payment)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"payment": payment,
		"status":  payment.Status,
	})
}

// GetUserPayments returns all payments made by the authenticated user
func (h *PaymentHandler) GetUserPayments(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var payments []models.Payment
	if err := h.db.Preload("User").Preload("Course").Preload("Course.Instructor").
		Where("user_id = ?", userID).Order("created_at DESC").Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments": payments})
}

// generateRandomString generates a random string for transaction references
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// PaymentSuccess handles the return URL from Chapa
func (h *PaymentHandler) PaymentSuccess(c *gin.Context) {
	txRef := c.Query("tx_ref")
	status := c.Query("status")

	if status == "success" && txRef != "" {
		// Find the payment
		var payment models.Payment
		if err := h.db.Preload("Course").Where("chapa_tx_ref = ?", txRef).First(&payment).Error; err == nil {
			c.HTML(http.StatusOK, "payment_success.html", gin.H{
				"course_title": payment.Course.Title,
				"amount":       payment.Amount,
				"currency":     payment.Currency,
			})
			return
		}
	}

	// Generic success message
	c.JSON(http.StatusOK, gin.H{
		"message": "Payment completed successfully! You can now access your course.",
		"status":  "success",
	})
}
