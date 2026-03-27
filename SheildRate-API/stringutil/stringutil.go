// Package stringutil provides advanced string manipulation utilities
// for input validation, secure logging, and text processing.
// Demonstrates Go-specific string handling: rune iteration, strings.Builder,
// regexp compilation, and Unicode-aware transformations.
package stringutil

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// clientIDPattern enforces that client IDs are alphanumeric with underscores,
// between 3 and 32 characters long. Compiled once for performance.
var clientIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

// slugUnsafe matches any character that is not alphanumeric or a hyphen.
var slugUnsafe = regexp.MustCompile(`[^a-z0-9-]+`)

// multiHyphen collapses consecutive hyphens into a single one.
var multiHyphen = regexp.MustCompile(`-{2,}`)

// ValidateClientID checks that a client ID conforms to the allowed pattern.
// Returns an error message string if invalid, or empty string if valid.
// Uses regexp for pattern matching and strings functions for normalization.
func ValidateClientID(id string) (string, bool) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "client ID cannot be empty", false
	}
	if len(trimmed) < 3 {
		return fmt.Sprintf("client ID '%s' is too short (min 3 chars)", trimmed), false
	}
	if len(trimmed) > 32 {
		return fmt.Sprintf("client ID exceeds maximum length of 32 characters (got %d)", len(trimmed)), false
	}
	if !clientIDPattern.MatchString(trimmed) {
		return fmt.Sprintf("client ID '%s' contains invalid characters (only alphanumeric and underscores allowed)", trimmed), false
	}
	return "", true
}

// MaskToken obscures the middle portion of a token string for safe logging.
// Shows the first `reveal` and last `reveal` characters, replacing the middle
// with dots. Handles edge cases for very short tokens gracefully.
//
// Example: MaskToken("eyJhbGciOiJIUzI1NiJ9.payload.signature", 6)
//          → "eyJhbG...nature"
func MaskToken(token string, reveal int) string {
	runes := []rune(token)
	n := len(runes)

	if n <= reveal*2 {
		// Token is too short to mask meaningfully
		return strings.Repeat("*", n)
	}

	var b strings.Builder
	b.Grow(reveal*2 + 3)
	for i := 0; i < reveal; i++ {
		b.WriteRune(runes[i])
	}
	b.WriteString("...")
	for i := n - reveal; i < n; i++ {
		b.WriteRune(runes[i])
	}
	return b.String()
}

// FormatLogEntry builds a structured, padded log line using strings.Builder
// for efficient string concatenation. Demonstrates advanced formatting with
// fixed-width columns and ANSI-style structure.
//
// Format: [TIMESTAMP] | CLIENT_ID (padded) | STATUS | MESSAGE
func FormatLogEntry(clientID, status, message string) string {
	const clientPadWidth = 16

	var b strings.Builder
	b.Grow(128)

	// Timestamp block
	b.WriteString("[")
	b.WriteString(time.Now().Format("2006-01-02 15:04:05.000"))
	b.WriteString("]")

	// Separator
	b.WriteString(" │ ")

	// Client ID — right-padded to fixed width using strings.Builder + Repeat
	b.WriteString(padRight(clientID, clientPadWidth))

	// Status indicator
	b.WriteString(" │ ")
	b.WriteString(padRight(strings.ToUpper(status), 8))

	// Message
	b.WriteString(" │ ")
	b.WriteString(message)

	return b.String()
}

// SanitizeInput removes potentially dangerous characters from user input
// using strings.Map with a custom rune-filtering function. Allows only
// printable, non-control Unicode characters, explicitly blocking common
// injection vectors.
func SanitizeInput(s string) string {
	return strings.Map(func(r rune) rune {
		// Block control characters, null bytes, and common injection chars
		if r == '<' || r == '>' || r == '&' || r == '"' || r == '\'' {
			return -1 // Drop the rune
		}
		if !unicode.IsPrint(r) || unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// Slugify converts an arbitrary string into a URL-safe, lowercase slug.
// Uses Unicode-aware lowercasing, regexp replacement, and multi-pass
// string transformation.
//
// Example: Slugify("Hello World! (2024)") → "hello-world-2024"
func Slugify(s string) string {
	// Step 1: Lowercase the entire string (Unicode-aware)
	slug := strings.ToLower(strings.TrimSpace(s))

	// Step 2: Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Step 3: Remove unsafe characters via compiled regexp
	slug = slugUnsafe.ReplaceAllString(slug, "")

	// Step 4: Collapse multiple hyphens
	slug = multiHyphen.ReplaceAllString(slug, "-")

	// Step 5: Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

// ReverseString reverses a Unicode-safe string by iterating over runes.
// This handles multi-byte characters correctly, unlike naive byte reversal.
func ReverseString(s string) string {
	runes := []rune(s)
	n := len(runes)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// CamelToSnake converts a CamelCase or PascalCase string to snake_case.
// Uses rune iteration and Unicode category detection.
//
// Example: CamelToSnake("RateLimiterService") → "rate_limiter_service"
func CamelToSnake(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)

	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				// Insert underscore before uppercase if previous char was lowercase
				// or next char is lowercase (handles acronyms like "HTTPServer")
				prev := runes[i-1]
				if unicode.IsLower(prev) || (i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
					b.WriteRune('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// padRight pads a string to the specified width with spaces on the right.
// If the string is longer than width, it is truncated.
func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}
