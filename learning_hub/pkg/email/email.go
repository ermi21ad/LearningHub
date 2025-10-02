package email

import (
	"crypto/tls"
	"fmt"
	"learning_hub/pkg/config"
	"log"
	"strings"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	config *config.Config
}

var (
	emailService *EmailService
)

type EmailData struct {
	To      string
	Subject string
	Body    string
	Name    string
}

func Init(cfg *config.Config) {
	emailService = &EmailService{config: cfg}

	// Test email configuration
	if cfg.SMTPHost != "" && cfg.SMTPUsername != "" {
		log.Println("✅ Email service initialized with SMTP")
	} else {
		log.Println("📧 Email service initialized in simulation mode")
	}
}

// SendEmail sends an email using SMTP or simulates in development
func SendEmail(data EmailData) error {
	if emailService == nil {
		return fmt.Errorf("email service not initialized")
	}

	cfg := emailService.config

	// Log email attempt
	log.Printf("📧 Attempting to send email to: %s", data.To)
	log.Printf("📧 Subject: %s", data.Subject)

	// Check if SMTP is configured
	if cfg.SMTPHost == "" || cfg.SMTPUsername == "" || cfg.SMTPPassword == "" {
		log.Printf("❌ SMTP not configured properly. Host: %s, Username: %s", cfg.SMTPHost, cfg.SMTPUsername)
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	// Create message
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("LearnHub <%s>", cfg.SMTPUsername))
	m.SetHeader("To", data.To)
	m.SetHeader("Subject", data.Subject)
	m.SetBody("text/html", data.Body)

	// Create dialer with proper configuration
	d := gomail.NewDialer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword)

	// For Gmail, you might need these settings:
	d.TLSConfig = &tls.Config{
		ServerName: cfg.SMTPHost,
	}

	// Only skip TLS verification in development
	if cfg.ServerEnv == "development" {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	log.Printf("🔧 Connecting to SMTP server: %s:%d", cfg.SMTPHost, cfg.SMTPPort)

	// Send email
	if err := d.DialAndSend(m); err != nil {
		log.Printf("❌ Failed to send email to %s: %v", data.To, err)
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("✅ Email sent successfully to: %s", data.To)
	return nil
}

// Helper function to strip HTML for logging
func stripHTML(html string) string {
	// Simple HTML tag removal for clean logging
	clean := strings.ReplaceAll(html, "<br>", "\n")
	clean = strings.ReplaceAll(clean, "</p>", "\n")
	clean = strings.ReplaceAll(clean, "<li>", "\n• ")

	// Remove all HTML tags
	var result strings.Builder
	var inTag bool

	for _, ch := range clean {
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(ch)
		}
	}

	return strings.TrimSpace(result.String())
}

// SendWelcomeEmail sends welcome email to new users
func SendWelcomeEmail(to, name string) error {
	subject := "🎉 Welcome to LearnHub!"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; }
				.container { max-width: 600px; margin: 0 auto; background: #ffffff; }
				.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 40px 20px; text-align: center; }
				.content { padding: 30px; background: #f8fafc; }
				.footer { padding: 20px; text-align: center; color: #64748b; font-size: 14px; background: #1e293b; color: white; }
				.button { display: inline-block; padding: 12px 30px; background: #10b981; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Welcome to LearnHub! 🎓</h1>
					<p>Your learning journey begins now</p>
				</div>
				<div class="content">
					<h2>Hello %s,</h2>
					<p>We're thrilled to welcome you to LearnHub - your gateway to knowledge and skill development!</p>
					
					<p><strong>What you can do now:</strong></p>
					<ul>
						<li>📚 Browse our extensive course catalog</li>
						<li>🎯 Enroll in courses that match your interests</li>
						<li>📈 Track your learning progress</li>
						<li>🏆 Earn certificates upon completion</li>
					</ul>
					
					<p>Ready to start learning?</p>
					<center>
						<a href="http://localhost:8080/api/courses" class="button">Explore Courses</a>
					</center>
					
					<p>If you have any questions, feel free to reply to this email.</p>
					
					<p>Happy learning!<br><strong>The LearnHub Team</strong></p>
				</div>
				<div class="footer">
					<p>&copy; 2024 LearnHub. All rights reserved.</p>
					<p>This is an automated message, please do not reply directly to this email.</p>
				</div>
			</div>
		</body>
		</html>
	`, name)

	return SendEmail(EmailData{
		To:      to,
		Subject: subject,
		Body:    body,
		Name:    name,
	})
}

// SendPaymentSuccessEmail sends payment confirmation email
func SendPaymentSuccessEmail(to, name, courseTitle string, amount float64, transactionRef string) error {
	subject := "✅ Payment Successful - Course Enrollment Confirmed"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; }
				.container { max-width: 600px; margin: 0 auto; background: #ffffff; }
				.header { background: linear-gradient(135deg, #10b981 0%%, #059669 100%%); color: white; padding: 40px 20px; text-align: center; }
				.content { padding: 30px; background: #f8fafc; }
				.receipt { background: white; padding: 20px; border-radius: 10px; border-left: 4px solid #10b981; margin: 20px 0; }
				.footer { padding: 20px; text-align: center; color: #64748b; font-size: 14px; background: #1e293b; color: white; }
				.button { display: inline-block; padding: 12px 30px; background: #3b82f6; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Payment Successful! 🎉</h1>
					<p>You're now enrolled in your course</p>
				</div>
				<div class="content">
					<h2>Hello %s,</h2>
					<p>Your payment has been processed successfully and you now have full access to your course.</p>
					
					<div class="receipt">
						<h3>📋 Payment Receipt</h3>
						<p><strong>Course:</strong> %s</p>
						<p><strong>Amount Paid:</strong> ETB %.2f</p>
						<p><strong>Transaction ID:</strong> %s</p>
						<p><strong>Status:</strong> <span style="color: #10b981;">Confirmed ✅</span></p>
						<p><strong>Access:</strong> Immediate</p>
					</div>
					
					<p>You can start learning right away! All course materials are now available to you.</p>
					
					<center>
						<a href="http://localhost:8080/api/my-courses" class="button">Start Learning Now</a>
					</center>
					
					<p>If you encounter any issues accessing your course, please contact our support team.</p>
					
					<p>Happy learning!<br><strong>The LearnHub Team</strong></p>
				</div>
				<div class="footer">
					<p>&copy; 2024 LearnHub. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, name, courseTitle, amount, transactionRef)

	return SendEmail(EmailData{
		To:      to,
		Subject: subject,
		Body:    body,
		Name:    name,
	})
}

// SendEnrollmentNotification sends notification to instructor about new enrollment
func SendEnrollmentNotification(to, instructorName, studentName, courseTitle string) error {
	subject := "🎓 New Student Enrollment - " + courseTitle
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; }
				.container { max-width: 600px; margin: 0 auto; background: #ffffff; }
				.header { background: linear-gradient(135deg, #f59e0b 0%%, #d97706 100%%); color: white; padding: 40px 20px; text-align: center; }
				.content { padding: 30px; background: #f8fafc; }
				.enrollment-info { background: white; padding: 20px; border-radius: 10px; border-left: 4px solid #f59e0b; margin: 20px 0; }
				.footer { padding: 20px; text-align: center; color: #64748b; font-size: 14px; background: #1e293b; color: white; }
				.button { display: inline-block; padding: 12px 30px; background: #3b82f6; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>New Student Enrollment! 🎉</h1>
					<p>Your course is making an impact</p>
				</div>
				<div class="content">
					<h2>Hello %s,</h2>
					<p>Great news! Another student has enrolled in your course and is excited to learn from you.</p>
					
					<div class="enrollment-info">
						<h3>📈 Enrollment Details</h3>
						<p><strong>Student:</strong> %s</p>
						<p><strong>Course:</strong> %s</p>
						<p><strong>Enrollment Date:</strong> %s</p>
						<p><strong>Total Students:</strong> Growing! 📊</p>
					</div>
					
					<p>Your expertise is helping shape the future of education. Keep up the amazing work!</p>
					
					<center>
						<a href="http://localhost:8080/api/dashboard" class="button">View Course Dashboard</a>
					</center>
					
					<p>Thank you for being an invaluable part of the LearnHub community.</p>
					
					<p>Best regards,<br><strong>The LearnHub Team</strong></p>
				</div>
				<div class="footer">
					<p>&copy; 2024 LearnHub. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, instructorName, studentName, courseTitle, getCurrentDate())

	return SendEmail(EmailData{
		To:      to,
		Subject: subject,
		Body:    body,
		Name:    instructorName,
	})
}

// SendAdminNotification sends notification to admin for important events
func SendAdminNotification(event, details string) error {
	// This would send to admin email - you can configure this
	adminEmail := "admin@learnhub.com" // You can make this configurable

	subject := "🔔 Admin Notification: " + event
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background: #ef4444; color: white; padding: 20px; text-align: center; }
				.content { padding: 20px; background: #fef2f2; }
				.info-box { background: white; padding: 15px; border-radius: 5px; margin: 15px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Admin Notification</h1>
				</div>
				<div class="content">
					<h2>Event: %s</h2>
					<div class="info-box">
						<p><strong>Details:</strong> %s</p>
						<p><strong>Time:</strong> %s</p>
					</div>
					<p>This is an automated notification from the LearnHub system.</p>
				</div>
			</div>
		</body>
		</html>
	`, event, details, getCurrentDate())

	return SendEmail(EmailData{
		To:      adminEmail,
		Subject: subject,
		Body:    body,
		Name:    "Admin",
	})
}

// Helper function to get current date
func getCurrentDate() string {
	// This would be implemented to return current date
	return "Just now"
}

// SendVerificationEmail sends email verification link
func SendVerificationEmail(to, name, verificationToken string) error {
	subject := "🔐 Verify Your LearnHub Account"
	verificationURL := fmt.Sprintf("http://localhost:8080/api/verify-email?token=%s", verificationToken)

	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; }
				.container { max-width: 600px; margin: 0 auto; background: #ffffff; }
				.header { background: linear-gradient(135deg, #6366f1 0%%, #4f46e5 100%%); color: white; padding: 40px 20px; text-align: center; }
				.content { padding: 30px; background: #f8fafc; }
				.verification-box { background: white; padding: 25px; border-radius: 10px; border: 2px dashed #e2e8f0; margin: 20px 0; text-align: center; }
				.verification-button { display: inline-block; padding: 15px 30px; background: #10b981; color: white; text-decoration: none; border-radius: 8px; font-size: 16px; font-weight: bold; margin: 15px 0; }
				.verification-code { background: #f1f5f9; padding: 15px; border-radius: 8px; font-family: monospace; font-size: 18px; color: #1e293b; margin: 15px 0; }
				.footer { padding: 20px; text-align: center; color: #64748b; font-size: 14px; background: #1e293b; color: white; }
				.note { background: #fffbeb; border-left: 4px solid #f59e0b; padding: 15px; margin: 15px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Verify Your Email Address</h1>
					<p>One last step to activate your LearnHub account</p>
				</div>
				<div class="content">
					<h2>Hello %s,</h2>
					<p>Thank you for registering with LearnHub! To complete your registration and activate your account, please verify your email address.</p>
					
					<div class="verification-box">
						<h3>📧 Email Verification Required</h3>
						<p>Click the button below to verify your email address:</p>
						
						<center>
							<a href="%s" class="verification-button">Verify Email Address</a>
						</center>
						
						<p style="margin-top: 20px; color: #64748b; font-size: 14px;">
							Or copy and paste this link in your browser:<br>
							<span class="verification-code">%s</span>
						</p>
					</div>
					
					<div class="note">
						<p><strong>⚠️ Important:</strong> This verification link will expire in 24 hours.</p>
						<p>If you didn't create an account with LearnHub, please ignore this email.</p>
					</div>
					
					<p>Once verified, you'll have full access to:</p>
					<ul>
						<li>📚 Browse and enroll in courses</li>
						<li>🎯 Track your learning progress</li>
						<li>🏆 Earn completion certificates</li>
						<li>👥 Join our learning community</li>
					</ul>
					
					<p>Happy learning!<br><strong>The LearnHub Team</strong></p>
				</div>
				<div class="footer">
					<p>&copy; 2024 LearnHub. All rights reserved.</p>
					<p>This is an automated message, please do not reply directly to this email.</p>
				</div>
			</div>
		</body>
		</html>
	`, name, verificationURL, verificationURL)

	return SendEmail(EmailData{
		To:      to,
		Subject: subject,
		Body:    body,
		Name:    name,
	})
}

// SendVerificationSuccessEmail sends confirmation after successful verification
func SendVerificationSuccessEmail(to, name string) error {
	subject := "✅ Email Verified Successfully!"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; }
				.container { max-width: 600px; margin: 0 auto; background: #ffffff; }
				.header { background: linear-gradient(135deg, #10b981 0%%, #059669 100%%); color: white; padding: 40px 20px; text-align: center; }
				.content { padding: 30px; background: #f8fafc; }
				.success-box { background: white; padding: 25px; border-radius: 10px; border: 2px solid #10b981; margin: 20px 0; text-align: center; }
				.button { display: inline-block; padding: 12px 30px; background: #3b82f6; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
				.footer { padding: 20px; text-align: center; color: #64748b; font-size: 14px; background: #1e293b; color: white; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Email Verified Successfully! 🎉</h1>
					<p>Your LearnHub account is now active</p>
				</div>
				<div class="content">
					<h2>Hello %s,</h2>
					
					<div class="success-box">
						<h3 style="color: #10b981;">✅ Verification Complete</h3>
						<p>Your email address has been successfully verified and your LearnHub account is now fully activated!</p>
					</div>
					
					<p>You now have full access to all LearnHub features:</p>
					<ul>
						<li>🔐 Secure account access</li>
						<li>📚 Full course catalog</li>
						<li>💳 Course enrollment and payments</li>
						<li>📊 Progress tracking</li>
						<li>🏆 Achievement system</li>
					</ul>
					
					<center>
						<a href="http://localhost:8080/api/courses" class="button">Start Exploring Courses</a>
					</center>
					
					<p style="margin-top: 30px;">If you have any questions or need assistance, don't hesitate to contact our support team.</p>
					
					<p>Welcome aboard!<br><strong>The LearnHub Team</strong></p>
				</div>
				<div class="footer">
					<p>&copy; 2024 LearnHub. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, name)

	return SendEmail(EmailData{
		To:      to,
		Subject: subject,
		Body:    body,
		Name:    name,
	})
}
