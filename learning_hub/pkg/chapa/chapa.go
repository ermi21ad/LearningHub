package chapa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"learning_hub/pkg/config"
	"log"
	"net/http"
	"time"
)

// Chapa API constants
const (
	BaseURL        = "https://api.chapa.co/v1"
	InitializePath = "/transaction/initialize"
	VerifyPath     = "/transaction/verify/"
	BanksPath      = "/banks"
)

// PaymentRequest represents the request to initialize a payment
type PaymentRequest struct {
	Amount        string                 `json:"amount"`
	Currency      string                 `json:"currency"`
	Email         string                 `json:"email"`
	FirstName     string                 `json:"first_name"`
	LastName      string                 `json:"last_name"`
	PhoneNumber   string                 `json:"phone_number,omitempty"`
	TxRef         string                 `json:"tx_ref"`
	CallbackURL   string                 `json:"callback_url"`
	ReturnURL     string                 `json:"return_url"`
	Customization Customization          `json:"customization,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
}

// Customization allows customizing the payment page
type Customization struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Logo        string `json:"logo,omitempty"`
}

// PaymentResponse represents the response from payment initialization
type PaymentResponse struct {
	Message string      `json:"message"`
	Status  string      `json:"status"`
	Data    PaymentData `json:"data"`
}

// PaymentData contains the payment URL and reference
type PaymentData struct {
	CheckoutURL string `json:"checkout_url"`
	TxRef       string `json:"tx_ref"`
}

// VerifyResponse represents the response from payment verification
type VerifyResponse struct {
	Message string     `json:"message"`
	Status  string     `json:"status"`
	Data    VerifyData `json:"data"`
}

// VerifyData contains verified payment details
type VerifyData struct {
	FirstName     string        `json:"first_name"`
	LastName      string        `json:"last_name"`
	Email         string        `json:"email"`
	Currency      string        `json:"currency"`
	Amount        float64       `json:"amount"`
	Charge        float64       `json:"charge"`
	Mode          string        `json:"mode"`
	Method        string        `json:"method"`
	Type          string        `json:"type"`
	Status        string        `json:"status"`
	Reference     string        `json:"reference"`
	TxRef         string        `json:"tx_ref"`
	Customization Customization `json:"customization"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// WebhookPayload represents the callback from Chapa webhook
type WebhookPayload struct {
	TxRef  string `json:"trx_ref"`
	RefID  string `json:"ref_id"`
	Status string `json:"status"`
}

// Client represents the Chapa API client
type Client struct {
	secretKey string
	baseURL   string
	client    *http.Client
}

var (
	ChapaClient *Client
)

// Init initializes the Chapa client with configuration
func Init(cfg *config.Config) error {
	if cfg.ChapaSecretKey == "" {
		return fmt.Errorf("CHAPA_SECRET_KEY is required")
	}

	ChapaClient = &Client{
		secretKey: cfg.ChapaSecretKey,
		baseURL:   BaseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	log.Println("✅ Chapa client initialized successfully")
	return nil
}

// TestConnection tests the Chapa API connection
func TestConnection() error {
	if ChapaClient == nil {
		return fmt.Errorf("Chapa client not initialized")
	}

	// Make a simple API call to test connection
	req, err := http.NewRequest("GET", ChapaClient.baseURL+BanksPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+ChapaClient.secretKey)

	resp, err := ChapaClient.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Chapa: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Chapa API returned status: %d", resp.StatusCode)
	}

	log.Println("✅ Chapa connection test successful")
	return nil
}

// InitializePayment creates a new payment transaction
// Update the InitializePayment function in pkg/chapa/chapa.go:

// InitializePayment creates a new payment transaction
func InitializePayment(paymentReq *PaymentRequest) (*PaymentResponse, error) {
	if ChapaClient == nil {
		return nil, fmt.Errorf("Chapa client not initialized")
	}

	// Validate required fields
	if err := validatePaymentRequest(paymentReq); err != nil {
		return nil, err
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(paymentReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", ChapaClient.baseURL+InitializePath, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+ChapaClient.secretKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := ChapaClient.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Debug: Print the raw response
	fmt.Printf("🔍 Chapa Raw Response (Status: %d): %s\n", resp.StatusCode, string(body))

	// Parse response
	var paymentResp PaymentResponse
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		// If parsing fails, try to parse as error response
		var errorResp map[string]interface{}
		if parseErr := json.Unmarshal(body, &errorResp); parseErr == nil {
			if message, exists := errorResp["message"]; exists {
				return nil, fmt.Errorf("Chapa API error: %v", message)
			}
		}
		return nil, fmt.Errorf("failed to parse Chapa response: %v - Raw: %s", err, string(body))
	}

	// Check if Chapa returned an error
	if paymentResp.Status != "success" && paymentResp.Status != "Success" {
		return nil, fmt.Errorf("Chapa API error: %s", paymentResp.Message)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Chapa API error: %s (status: %d)", paymentResp.Message, resp.StatusCode)
	}

	return &paymentResp, nil
}

// VerifyPayment verifies a payment transaction
func VerifyPayment(txRef string) (*VerifyResponse, error) {
	if ChapaClient == nil {
		return nil, fmt.Errorf("Chapa client not initialized")
	}

	// Create HTTP request
	req, err := http.NewRequest("GET", ChapaClient.baseURL+VerifyPath+txRef, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+ChapaClient.secretKey)

	// Send request
	resp, err := ChapaClient.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse response
	var verifyResp VerifyResponse
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Chapa API error: %s (status: %d)", verifyResp.Message, resp.StatusCode)
	}

	return &verifyResp, nil
}

// validatePaymentRequest validates the payment request parameters
func validatePaymentRequest(req *PaymentRequest) error {
	if req.Amount == "" {
		return fmt.Errorf("amount is required")
	}
	if req.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if req.FirstName == "" {
		return fmt.Errorf("first name is required")
	}
	if req.LastName == "" {
		return fmt.Errorf("last name is required")
	}
	if req.TxRef == "" {
		return fmt.Errorf("transaction reference is required")
	}
	if req.CallbackURL == "" {
		return fmt.Errorf("callback URL is required")
	}
	if req.ReturnURL == "" {
		return fmt.Errorf("return URL is required")
	}

	// Validate phone number format if provided
	if req.PhoneNumber != "" {
		if len(req.PhoneNumber) != 10 {
			return fmt.Errorf("phone number must be 10 digits")
		}
		// Check if it starts with 09 or 07
		if !(req.PhoneNumber[:2] == "09" || req.PhoneNumber[:2] == "07") {
			return fmt.Errorf("phone number must start with 09 or 07")
		}
	}

	return nil
}

// GetSecretKey returns the secret key (for webhook verification)
func GetSecretKey() string {
	if ChapaClient == nil {
		return ""
	}
	return ChapaClient.secretKey
}
