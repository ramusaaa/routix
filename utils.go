package routix

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func GenerateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func ParseDuration(s string) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	
	switch s {
	case "1s":
		return time.Second
	case "1m":
		return time.Minute
	case "5m":
		return 5 * time.Minute
	case "10m":
		return 10 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "24h":
		return 24 * time.Hour
	default:
		return time.Minute
	}
}

func ToJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func FromJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func ToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func FromMap(m map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, v)
}

func GetContentType(req *http.Request) string {
	return req.Header.Get("Content-Type")
}

func IsJSONRequest(req *http.Request) bool {
	contentType := GetContentType(req)
	return strings.Contains(contentType, "application/json")
}

func IsFormRequest(req *http.Request) bool {
	contentType := GetContentType(req)
	return strings.Contains(contentType, "application/x-www-form-urlencoded") ||
		strings.Contains(contentType, "multipart/form-data")
}

func GetRealIP(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	if cf := req.Header.Get("CF-Connecting-IP"); cf != "" {
		return cf
	}
	
	return req.RemoteAddr
}

func ParseInt(s string, defaultValue int) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return defaultValue
}

func ParseInt64(s string, defaultValue int64) int64 {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	return defaultValue
}

func ParseFloat(s string, defaultValue float64) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return defaultValue
}

func ParseBool(s string, defaultValue bool) bool {
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return defaultValue
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ContainsInt(slice []int, item int) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}

func Unique(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

func UniqueInt(slice []int) []int {
	keys := make(map[int]bool)
	var result []int
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

func Merge(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	
	return result
}

func Clone(src interface{}) interface{} {
	srcValue := reflect.ValueOf(src)
	if !srcValue.IsValid() {
		return nil
	}
	
	return reflect.New(srcValue.Type()).Elem().Interface()
}

func IsEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	}
	
	return false
}

func Coalesce(values ...interface{}) interface{} {
	for _, value := range values {
		if !IsEmpty(value) {
			return value
		}
	}
	return nil
}

func TernaryString(condition bool, trueValue, falseValue string) string {
	if condition {
		return trueValue
	}
	return falseValue
}

func TernaryInt(condition bool, trueValue, falseValue int) int {
	if condition {
		return trueValue
	}
	return falseValue
}

func Paginate(items interface{}, page, limit int) (interface{}, int, error) {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil, 0, fmt.Errorf("items must be a slice")
	}
	
	total := v.Len()
	totalPages := (total + limit - 1) / limit
	
	if page < 1 {
		page = 1
	}
	
	start := (page - 1) * limit
	end := start + limit
	
	if start >= total {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface(), totalPages, nil
	}
	
	if end > total {
		end = total
	}
	
	return v.Slice(start, end).Interface(), totalPages, nil
}

func Chunk(items interface{}, size int) (interface{}, error) {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("items must be a slice")
	}
	
	length := v.Len()
	chunks := reflect.MakeSlice(reflect.SliceOf(v.Type()), 0, (length+size-1)/size)
	
	for i := 0; i < length; i += size {
		end := i + size
		if end > length {
			end = length
		}
		
		chunk := v.Slice(i, end)
		chunks = reflect.Append(chunks, chunk)
	}
	
	return chunks.Interface(), nil
}

func Filter(items interface{}, predicate func(interface{}) bool) (interface{}, error) {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("items must be a slice")
	}
	
	result := reflect.MakeSlice(v.Type(), 0, v.Len())
	
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		if predicate(item) {
			result = reflect.Append(result, v.Index(i))
		}
	}
	
	return result.Interface(), nil
}

func Map(items interface{}, mapper func(interface{}) interface{}) ([]interface{}, error) {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("items must be a slice")
	}
	
	result := make([]interface{}, v.Len())
	
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		result[i] = mapper(item)
	}
	
	return result, nil
}

func Reduce(items interface{}, reducer func(interface{}, interface{}) interface{}, initial interface{}) (interface{}, error) {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("items must be a slice")
	}
	
	accumulator := initial
	
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		accumulator = reducer(accumulator, item)
	}
	
	return accumulator, nil
}