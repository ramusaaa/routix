package routix

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Constraint interface {
	Validate(value interface{}) error
	Name() string
}

type RequiredConstraint struct{}

func (c RequiredConstraint) Name() string {
	return "required"
}

func (c RequiredConstraint) Validate(value interface{}) error {
	if value == nil {
		return fmt.Errorf("value is required")
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		if v.String() == "" {
			return fmt.Errorf("value is required")
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if v.Len() == 0 {
			return fmt.Errorf("value is required")
		}
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return fmt.Errorf("value is required")
		}
	}
	
	return nil
}

type MinLengthConstraint struct {
	Min int
}

func (c MinLengthConstraint) Name() string {
	return fmt.Sprintf("min=%d", c.Min)
}

func (c MinLengthConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	v := reflect.ValueOf(value)
	var length int
	
	switch v.Kind() {
	case reflect.String:
		length = len(v.String())
	case reflect.Slice, reflect.Map, reflect.Array:
		length = v.Len()
	default:
		return fmt.Errorf("min length constraint not applicable to type %T", value)
	}
	
	if length < c.Min {
		return fmt.Errorf("value must be at least %d characters/items", c.Min)
	}
	
	return nil
}

type MaxLengthConstraint struct {
	Max int
}

func (c MaxLengthConstraint) Name() string {
	return fmt.Sprintf("max=%d", c.Max)
}

func (c MaxLengthConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	v := reflect.ValueOf(value)
	var length int
	
	switch v.Kind() {
	case reflect.String:
		length = len(v.String())
	case reflect.Slice, reflect.Map, reflect.Array:
		length = v.Len()
	default:
		return fmt.Errorf("max length constraint not applicable to type %T", value)
	}
	
	if length > c.Max {
		return fmt.Errorf("value must be at most %d characters/items", c.Max)
	}
	
	return nil
}

type MinValueConstraint struct {
	Min float64
}

func (c MinValueConstraint) Name() string {
	return fmt.Sprintf("min=%g", c.Min)
}

func (c MinValueConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	var num float64
	var err error
	
	switch v := value.(type) {
	case int:
		num = float64(v)
	case int32:
		num = float64(v)
	case int64:
		num = float64(v)
	case float32:
		num = float64(v)
	case float64:
		num = v
	case string:
		num, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("value must be a number")
		}
	default:
		return fmt.Errorf("min value constraint not applicable to type %T", value)
	}
	
	if num < c.Min {
		return fmt.Errorf("value must be at least %g", c.Min)
	}
	
	return nil
}

type MaxValueConstraint struct {
	Max float64
}

func (c MaxValueConstraint) Name() string {
	return fmt.Sprintf("max=%g", c.Max)
}

func (c MaxValueConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	var num float64
	var err error
	
	switch v := value.(type) {
	case int:
		num = float64(v)
	case int32:
		num = float64(v)
	case int64:
		num = float64(v)
	case float32:
		num = float64(v)
	case float64:
		num = v
	case string:
		num, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("value must be a number")
		}
	default:
		return fmt.Errorf("max value constraint not applicable to type %T", value)
	}
	
	if num > c.Max {
		return fmt.Errorf("value must be at most %g", c.Max)
	}
	
	return nil
}

type EmailConstraint struct{}

func (c EmailConstraint) Name() string {
	return "email"
}

func (c EmailConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("email constraint only applicable to strings")
	}
	
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("value must be a valid email address")
	}
	
	return nil
}

type RegexConstraint struct {
	Pattern *regexp.Regexp
	Raw     string
}

func (c RegexConstraint) Name() string {
	return fmt.Sprintf("regex=%s", c.Raw)
}

func (c RegexConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("regex constraint only applicable to strings")
	}
	
	if !c.Pattern.MatchString(str) {
		return fmt.Errorf("value must match pattern %s", c.Raw)
	}
	
	return nil
}

type EnumConstraint struct {
	Values []interface{}
}

func (c EnumConstraint) Name() string {
	var strs []string
	for _, v := range c.Values {
		strs = append(strs, fmt.Sprintf("%v", v))
	}
	return fmt.Sprintf("enum=%s", strings.Join(strs, "|"))
}

func (c EnumConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	for _, allowed := range c.Values {
		if reflect.DeepEqual(value, allowed) {
			return nil
		}
	}
	
	return fmt.Errorf("value must be one of: %v", c.Values)
}

type DateConstraint struct {
	Format string
}

func (c DateConstraint) Name() string {
	return fmt.Sprintf("date=%s", c.Format)
}

func (c DateConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("date constraint only applicable to strings")
	}
	
	_, err := time.Parse(c.Format, str)
	if err != nil {
		return fmt.Errorf("value must be a valid date in format %s", c.Format)
	}
	
	return nil
}

type URLConstraint struct{}

func (c URLConstraint) Name() string {
	return "url"
}

func (c URLConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("url constraint only applicable to strings")
	}
	
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(str) {
		return fmt.Errorf("value must be a valid URL")
	}
	
	return nil
}

type AlphaConstraint struct{}

func (c AlphaConstraint) Name() string {
	return "alpha"
}

func (c AlphaConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("alpha constraint only applicable to strings")
	}
	
	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	if !alphaRegex.MatchString(str) {
		return fmt.Errorf("value must contain only alphabetic characters")
	}
	
	return nil
}

type AlphaNumConstraint struct{}

func (c AlphaNumConstraint) Name() string {
	return "alphanum"
}

func (c AlphaNumConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("alphanum constraint only applicable to strings")
	}
	
	alphaNumRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphaNumRegex.MatchString(str) {
		return fmt.Errorf("value must contain only alphanumeric characters")
	}
	
	return nil
}

type NumericConstraint struct{}

func (c NumericConstraint) Name() string {
	return "numeric"
}

func (c NumericConstraint) Validate(value interface{}) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("numeric constraint only applicable to strings")
	}
	
	_, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return fmt.Errorf("value must be numeric")
	}
	
	return nil
}

func ParseConstraints(tag string) ([]Constraint, error) {
	var constraints []Constraint
	
	if tag == "" {
		return constraints, nil
	}
	
	rules := strings.Split(tag, ",")
	
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		
		switch {
		case rule == "required":
			constraints = append(constraints, RequiredConstraint{})
			
		case strings.HasPrefix(rule, "min="):
			val, err := strconv.Atoi(rule[4:])
			if err != nil {
				return nil, fmt.Errorf("invalid min constraint: %s", rule)
			}
			constraints = append(constraints, MinLengthConstraint{Min: val})
			
		case strings.HasPrefix(rule, "max="):
			val, err := strconv.Atoi(rule[4:])
			if err != nil {
				return nil, fmt.Errorf("invalid max constraint: %s", rule)
			}
			constraints = append(constraints, MaxLengthConstraint{Max: val})
			
		case rule == "email":
			constraints = append(constraints, EmailConstraint{})
			
		case strings.HasPrefix(rule, "regex="):
			pattern := rule[6:]
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid regex constraint: %s", rule)
			}
			constraints = append(constraints, RegexConstraint{Pattern: regex, Raw: pattern})
			
		case strings.HasPrefix(rule, "enum="):
			values := strings.Split(rule[5:], "|")
			var enumValues []interface{}
			for _, v := range values {
				enumValues = append(enumValues, v)
			}
			constraints = append(constraints, EnumConstraint{Values: enumValues})
			
		case strings.HasPrefix(rule, "date="):
			format := rule[5:]
			constraints = append(constraints, DateConstraint{Format: format})
			
		case rule == "url":
			constraints = append(constraints, URLConstraint{})
			
		case rule == "alpha":
			constraints = append(constraints, AlphaConstraint{})
			
		case rule == "alphanum":
			constraints = append(constraints, AlphaNumConstraint{})
			
		case rule == "numeric":
			constraints = append(constraints, NumericConstraint{})
			
		default:
			return nil, fmt.Errorf("unknown constraint: %s", rule)
		}
	}
	
	return constraints, nil
}