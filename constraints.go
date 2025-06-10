package routix

import (
	"fmt"
	"regexp"
	"time"
)

// EmailConstraint validates email addresses
type EmailConstraint struct {
	*BaseSchema
}

func NewEmailConstraint() *EmailConstraint {
	return &EmailConstraint{BaseSchema: &BaseSchema{}}
}

func (e *EmailConstraint) Validate(value any) error {
	if value == nil {
		if e.IsRequired() {
			return fmt.Errorf("value is required")
		}
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// URLConstraint validates URLs
type URLConstraint struct {
	*BaseSchema
}

func NewURLConstraint() *URLConstraint {
	return &URLConstraint{BaseSchema: &BaseSchema{}}
}

func (u *URLConstraint) Validate(value any) error {
	if value == nil {
		if u.IsRequired() {
			return fmt.Errorf("value is required")
		}
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	urlRegex := regexp.MustCompile(`^(https?|ftp)://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(str) {
		return fmt.Errorf("invalid URL format")
	}

	return nil
}

// DateConstraint validates dates
type DateConstraint struct {
	*BaseSchema
	format string
}

func NewDateConstraint(format string) *DateConstraint {
	return &DateConstraint{
		BaseSchema: &BaseSchema{},
		format:     format,
	}
}

func (d *DateConstraint) Validate(value any) error {
	if value == nil {
		if d.IsRequired() {
			return fmt.Errorf("value is required")
		}
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	_, err := time.Parse(d.format, str)
	if err != nil {
		return fmt.Errorf("invalid date format: %v", err)
	}

	return nil
}

// RegexConstraint validates strings against a regex pattern
type RegexConstraint struct {
	*BaseSchema
	pattern *regexp.Regexp
}

func NewRegexConstraint(pattern string) (*RegexConstraint, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %v", err)
	}

	return &RegexConstraint{
		BaseSchema: &BaseSchema{},
		pattern:    re,
	}, nil
}

func (r *RegexConstraint) Validate(value any) error {
	if value == nil {
		if r.IsRequired() {
			return fmt.Errorf("value is required")
		}
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if !r.pattern.MatchString(str) {
		return fmt.Errorf("value does not match pattern")
	}

	return nil
}
