package fileupload

import (
	"errors"
	"fmt"
	"io"
	"learning_hub/pkg/config"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File type categories and their allowed extensions
const (
	// Size limits in bytes
	ImageMaxSize    = 10 * 1024 * 1024  // 10MB
	VideoMaxSize    = 100 * 1024 * 1024 // 100MB
	DocumentMaxSize = 5 * 1024 * 1024   // 5MB

	// File type categories - EXPORTED
	FileTypeImage    = "image"
	FileTypeVideo    = "video"
	FileTypeDocument = "document"
)

// Allowed file extensions
var (
	AllowedImageExtensions = map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true,
	}

	AllowedVideoExtensions = map[string]bool{
		".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true, ".flv": true, ".wmv": true,
	}

	AllowedDocumentExtensions = map[string]bool{
		".pdf": true, ".doc": true, ".docx": true, ".ppt": true, ".pptx": true,
		".xls": true, ".xlsx": true, ".txt": true, ".csv": true, ".rtf": true, ".md": true, ".epub": true, ".mobi": true,
	}
)

// FileValidationResult holds the result of file validation
type FileValidationResult struct {
	IsValid bool
	Type    string
	Error   error
}

// UploadResult holds the result of a file upload operation
type UploadResult struct {
	Filename string
	FileURL  string
	FilePath string
	FileType string
	Size     int64
	Success  bool
	Error    error
}

type FileUpload struct {
	cfg *config.Config
}

var (
	fileUpload *FileUpload
)

// Init initializes the file upload package with configuration
func Init(cfg *config.Config) {
	fileUpload = &FileUpload{cfg: cfg}

	// Create upload directories if they don't exist
	dirs := []string{
		"uploads/images",
		"uploads/documents",
		"uploads/videos",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Warning: Failed to create directory %s: %v\n", dir, err)
		}
	}
}

// UploadFile handles file upload with comprehensive validation
func UploadFile(file *multipart.FileHeader, fileType string) (*UploadResult, error) {
	if fileUpload == nil {
		return nil, errors.New("file upload not initialized - call fileupload.Init() first")
	}

	// Validate file type parameter
	if fileType == "" {
		detectedType, err := DetectFileType(file.Filename)
		if err != nil {
			return nil, fmt.Errorf("cannot detect file type and no type specified: %v", err)
		}
		fileType = detectedType
	}

	// Perform comprehensive validation
	validation := ValidateFile(file, fileType)
	if !validation.IsValid {
		return nil, validation.Error
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := generateUniqueFilename(ext)
	uploadPath := GetUploadPath(fileType)
	fullPath := filepath.Join("uploads", uploadPath, filename)

	// Save the file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("failed to save file: %v", err)
	}

	// Return upload result
	result := &UploadResult{
		Filename: filename,
		FileURL:  GetFileURL(filename, fileType),
		FilePath: fullPath,
		FileType: fileType,
		Size:     file.Size,
		Success:  true,
	}

	return result, nil
}

// UploadFileWithAutoDetect handles file upload with automatic type detection
func UploadFileWithAutoDetect(file *multipart.FileHeader) (*UploadResult, error) {
	fileType, err := DetectFileType(file.Filename)
	if err != nil {
		return nil, err
	}
	return UploadFile(file, fileType)
}

// GetMaxSize returns the maximum allowed file size for a given file type
func GetMaxSize(fileType string) int64 {
	switch fileType {
	case FileTypeImage:
		return ImageMaxSize
	case FileTypeVideo:
		return VideoMaxSize
	case FileTypeDocument:
		return DocumentMaxSize
	default:
		return DocumentMaxSize
	}
}

// GetAllowedExtensions returns the allowed extensions for a given file type
func GetAllowedExtensions(fileType string) map[string]bool {
	switch fileType {
	case FileTypeImage:
		return AllowedImageExtensions
	case FileTypeVideo:
		return AllowedVideoExtensions
	case FileTypeDocument:
		return AllowedDocumentExtensions
	default:
		return AllowedDocumentExtensions
	}
}

// ValidateFileExtension checks if the file extension is allowed for the given file type
func ValidateFileExtension(filename, fileType string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	allowedExtensions := GetAllowedExtensions(fileType)
	if !allowedExtensions[ext] {
		return fmt.Errorf("file extension %s is not allowed for %s files", ext, fileType)
	}

	return nil
}

// ValidateFileSize checks if the file size is within the allowed limit
func ValidateFileSize(fileHeader *multipart.FileHeader, fileType string) error {
	maxSize := GetMaxSize(fileType)
	if fileHeader.Size > maxSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes for %s files",
			fileHeader.Size, maxSize, fileType)
	}
	return nil
}

// DetectFileType attempts to detect the file type based on extension
func DetectFileType(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	if AllowedImageExtensions[ext] {
		return FileTypeImage, nil
	}
	if AllowedVideoExtensions[ext] {
		return FileTypeVideo, nil
	}
	if AllowedDocumentExtensions[ext] {
		return FileTypeDocument, nil
	}

	return "", errors.New("unsupported file type")
}

// ValidateFile performs comprehensive validation of a file
func ValidateFile(fileHeader *multipart.FileHeader, expectedType string) *FileValidationResult {
	result := &FileValidationResult{IsValid: false}

	// Validate file extension
	if err := ValidateFileExtension(fileHeader.Filename, expectedType); err != nil {
		result.Error = err
		return result
	}

	// Validate file size
	if err := ValidateFileSize(fileHeader, expectedType); err != nil {
		result.Error = err
		return result
	}

	// Additional MIME type validation
	if err := validateMIMEType(fileHeader, expectedType); err != nil {
		result.Error = err
		return result
	}

	result.IsValid = true
	result.Type = expectedType
	return result
}

// GetUploadPath returns the appropriate subdirectory path for a file type
func GetUploadPath(fileType string) string {
	switch fileType {
	case FileTypeImage:
		return "images"
	case FileTypeVideo:
		return "videos"
	case FileTypeDocument:
		return "documents"
	default:
		return "documents"
	}
}

// GetFileURL returns the URL to access the uploaded file
func GetFileURL(filename, fileType string) string {
	if filename == "" {
		return ""
	}
	return fmt.Sprintf("/uploads/%s/%s", GetUploadPath(fileType), filename)
}

// GetFilePath returns the full filesystem path for a file
func GetFilePath(filename, fileType string) string {
	if filename == "" {
		return ""
	}
	return filepath.Join("uploads", GetUploadPath(fileType), filename)
}

// DeleteFile removes an uploaded file
func DeleteFile(filename, fileType string) error {
	if filename == "" {
		return errors.New("filename cannot be empty")
	}

	filePath := GetFilePath(filename, fileType)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.New("file does not exist")
	}

	return os.Remove(filePath)
}

// FileExists checks if a file exists
func FileExists(filename, fileType string) bool {
	if filename == "" {
		return false
	}

	filePath := GetFilePath(filename, fileType)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// validateMIMEType performs basic MIME type validation
func validateMIMEType(fileHeader *multipart.FileHeader, expectedType string) error {
	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to open file for MIME validation: %v", err)
	}
	defer file.Close()

	// Read first 512 bytes to detect MIME type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file for MIME validation: %v", err)
	}

	mimeType := http.DetectContentType(buffer[:n])

	// Basic MIME type validation
	switch expectedType {
	case FileTypeImage:
		if !strings.HasPrefix(mimeType, "image/") {
			return fmt.Errorf("file is not a valid image, detected MIME type: %s", mimeType)
		}
	case FileTypeVideo:
		if !strings.HasPrefix(mimeType, "video/") {
			return fmt.Errorf("file is not a valid video, detected MIME type: %s", mimeType)
		}
	case FileTypeDocument:
		// Document MIME types are more varied, so we rely more on extension
		// but we can still check for obviously wrong types
		if strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/") {
			return fmt.Errorf("file does not appear to be a document, detected MIME type: %s", mimeType)
		}
	}

	return nil
}

// generateUniqueFilename creates a unique filename with timestamp and random string
func generateUniqueFilename(extension string) string {
	timestamp := time.Now().UnixNano()
	randomStr := generateRandomString(8)
	return fmt.Sprintf("%d_%s%s", timestamp, randomStr, extension)
}

// generateRandomString generates a random string for filename
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// GetSupportedFileTypes returns a map of supported file types and their extensions
func GetSupportedFileTypes() map[string][]string {
	return map[string][]string{
		FileTypeImage:    getKeys(AllowedImageExtensions),
		FileTypeVideo:    getKeys(AllowedVideoExtensions),
		FileTypeDocument: getKeys(AllowedDocumentExtensions),
	}
}

// getKeys returns the keys of a map as a slice
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
