package fileupload

import (
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

// File type categories and their allowed extensions
const (
	// Size limits in bytes
	ImageMaxSize    = 10 * 1024 * 1024  // 10MB
	VideoMaxSize    = 100 * 1024 * 1024 // 100MB
	DocumentMaxSize = 5 * 1024 * 1024   // 5MB

	// File type categories
	FileTypeImage    = "image"
	FileTypeVideo    = "video"
	FileTypeDocument = "document"
)

// Allowed file extensions
var (
	AllowedImageExtensions = map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".svg":  true,
	}

	AllowedVideoExtensions = map[string]bool{
		".mp4":  true,
		".mov":  true,
		".avi":  true,
		".mkv":  true,
		".webm": true,
		".flv":  true,
		".wmv":  true,
	}

	AllowedDocumentExtensions = map[string]bool{
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".ppt":  true,
		".pptx": true,
		".xls":  true,
		".xlsx": true,
		".txt":  true,
		".csv":  true,
		".rtf":  true,
		".md":   true,
		".epub": true,
		".mobi": true,
	}
)

// FileValidationResult holds the result of file validation
type FileValidationResult struct {
	IsValid bool
	Type    string
	Error   error
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
		return DocumentMaxSize // default to document size
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
