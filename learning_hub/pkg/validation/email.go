package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Valid email domains for registration
var allowedDomains = map[string]bool{
	// Google
	"gmail.com":      true,
	"googlemail.com": true,
	"google.com":     true,

	// Microsoft
	"outlook.com": true,
	"hotmail.com": true,
	"live.com":    true,
	"msn.com":     true,

	// Yahoo
	"yahoo.com":   true,
	"yahoo.co.uk": true,
	"yahoo.ca":    true,
	"ymail.com":   true,

	// Apple
	"icloud.com": true,
	"me.com":     true,
	"mac.com":    true,

	// Proton
	"protonmail.com": true,
	"proton.me":      true,

	// Educational (common ones)
	"edu.com":         true,
	"edu.et":          true, // Ethiopian universities
	"aau.edu.et":      true, // Addis Ababa University
	"haramaya.edu.et": true,
	"hu.edu.et":       true, // Hawassa University
	"ju.edu.et":       true, // Jimma University
	"mu.edu.et":       true, // Mekelle University
	"astu.edu.et":     true, // Adama Science & Technology
	"bdu.edu.et":      true, // Bahir Dar University

	// Ethiopian providers
	"ethionet.et":    true,
	"telecom.net.et": true,

	// Common international providers
	"aol.com":    true,
	"zoho.com":   true,
	"yandex.com": true,
	"mail.com":   true,
	"gmx.com":    true,
}

// Email validation regex
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail validates email format and domain
func IsValidEmail(email string) (bool, string) {
	// Basic format validation
	if !emailRegex.MatchString(email) {
		return false, "Invalid email format"
	}

	// Extract domain
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false, "Invalid email format"
	}

	domain := strings.ToLower(parts[1])

	// Check if domain is in allowed list
	if !allowedDomains[domain] {
		return false, fmt.Sprintf("Email domain '%s' is not allowed for registration. Please use Gmail, Outlook, Yahoo, or other supported email providers.", domain)
	}

	// Additional validation: Check domain has MX records (optional but thorough)
	if hasMX, err := checkMXRecords(domain); err == nil && !hasMX {
		return false, "Email domain does not appear to be valid (no MX records found)"
	}

	return true, ""
}

// checkMXRecords verifies if the domain has valid MX records
func checkMXRecords(domain string) (bool, error) {
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return false, err
	}
	return len(mxRecords) > 0, nil
}

// GetAllowedDomains returns the list of allowed email domains
func GetAllowedDomains() []string {
	domains := make([]string, 0, len(allowedDomains))
	for domain := range allowedDomains {
		domains = append(domains, domain)
	}
	return domains
}

// AddCustomDomain allows adding custom domains during runtime (for admin use)
func AddCustomDomain(domain string) {
	allowedDomains[strings.ToLower(domain)] = true
}

// RemoveCustomDomain allows removing domains during runtime
func RemoveCustomDomain(domain string) {
	delete(allowedDomains, strings.ToLower(domain))
}
