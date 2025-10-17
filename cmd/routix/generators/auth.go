package generators

import (
	"path/filepath"
)

func GenerateAuthFiles(projectName string, config ProjectConfig) {
	generateAuthController(projectName)
	generateAuthMiddleware(projectName)
	generateAuthService(projectName)
	generateUserModel(projectName)
}

func generateAuthController(projectName string) {
	content := `package controllers

import (
	"github.com/ramusaaa/routix"
	"` + projectName + `/app/services"
)

type AuthController struct {
	BaseController
	authService *services.AuthService
}

func NewAuthController() *AuthController {
	return &AuthController{
		authService: services.NewAuthService(),
	}
}

func (ctrl *AuthController) Register(c *routix.Context) error {
	// TODO: Implement user registration
	return ctrl.Created(c, map[string]interface{}{
		"message": "User registered successfully",
	})
}

func (ctrl *AuthController) Login(c *routix.Context) error {
	// TODO: Implement user login
	token, err := ctrl.authService.Login("user@example.com", "password")
	if err != nil {
		return ctrl.Error(c, 401, "Invalid credentials")
	}

	return ctrl.Success(c, map[string]interface{}{
		"token": token,
		"type":  "Bearer",
	})
}

func (ctrl *AuthController) Logout(c *routix.Context) error {
	// TODO: Implement user logout
	return ctrl.Success(c, map[string]interface{}{
		"message": "Logged out successfully",
	})
}

func (ctrl *AuthController) Me(c *routix.Context) error {
	// TODO: Get authenticated user
	return ctrl.Success(c, map[string]interface{}{
		"id":    1,
		"email": "user@example.com",
		"name":  "John Doe",
	})
}

func (ctrl *AuthController) RefreshToken(c *routix.Context) error {
	// TODO: Implement token refresh
	return ctrl.Success(c, map[string]interface{}{
		"token": "new-jwt-token",
		"type":  "Bearer",
	})
}`

	writeFile(filepath.Join(projectName, "app", "controllers", "auth_controller.go"), content)
}

func generateAuthMiddleware(projectName string) {
	content := `package middleware

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ramusaaa/routix"
	"` + projectName + `/config"
)

func Auth() routix.Middleware {
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				return c.JSON(401, map[string]interface{}{
					"status":  "error",
					"message": "Authorization header required",
				})
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				return c.JSON(401, map[string]interface{}{
					"status":  "error",
					"message": "Invalid authorization format",
				})
			}

			cfg := config.Load()
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(cfg.JWTSecret), nil
			})

			if err != nil || !token.Valid {
				return c.JSON(401, map[string]interface{}{
					"status":  "error",
					"message": "Invalid token",
				})
			}

			// Add user info to context
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				c.Request = c.Request.WithContext(
					// Add user ID to context
					c.Request.Context(),
				)
				_ = claims // Use claims to get user info
			}

			return next(c)
		}
	}
}

func OptionalAuth() routix.Middleware {
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				// Try to authenticate but don't fail if invalid
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				cfg := config.Load()
				
				if token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					return []byte(cfg.JWTSecret), nil
				}); err == nil && token.Valid {
					// Add user info to context if valid
					if claims, ok := token.Claims.(jwt.MapClaims); ok {
						_ = claims // Use claims to get user info
					}
				}
			}

			return next(c)
		}
	}
}`

	writeFile(filepath.Join(projectName, "app", "middleware", "auth.go"), content)
}

func generateAuthService(projectName string) {
	content := `package services

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"` + projectName + `/config"
)

type AuthService struct {
	cfg *config.Config
}

func NewAuthService() *AuthService {
	return &AuthService{
		cfg: config.Load(),
	}
}

func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *AuthService) GenerateToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWTSecret), nil
	})
}

func (s *AuthService) Login(email, password string) (string, error) {
	// TODO: Implement actual login logic with database
	// This is a placeholder implementation
	
	// 1. Find user by email
	// 2. Check password
	// 3. Generate token
	
	return s.GenerateToken(1) // Placeholder user ID
}

func (s *AuthService) Register(email, password, name string) error {
	// TODO: Implement user registration
	// 1. Hash password
	// 2. Create user in database
	// 3. Send welcome email (optional)
	
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return err
	}
	
	_ = hashedPassword // Use this to save user
	_ = email
	_ = name
	
	return nil
}`

	writeFile(filepath.Join(projectName, "app", "services", "auth_service.go"), content)
}

func generateUserModel(projectName string) {
	content := `package models

import (
	"gorm.io/gorm"
)

type User struct {
	BaseModel
	Name     string ` + "`" + `gorm:"not null" json:"name"` + "`" + `
	Email    string ` + "`" + `gorm:"uniqueIndex;not null" json:"email"` + "`" + `
	Password string ` + "`" + `gorm:"not null" json:"-"` + "`" + `
	IsActive bool   ` + "`" + `gorm:"default:true" json:"is_active"` + "`" + `
}

func (u *User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	// Hash password before creating user
	// This should be done in the service layer
	return nil
}

func (u *User) AfterCreate(tx *gorm.DB) error {
	// Send welcome email or perform other actions
	return nil
}

// User methods
func (u *User) IsAdmin() bool {
	// TODO: Implement admin check logic
	return false
}

func (u *User) CanAccess(resource string) bool {
	// TODO: Implement permission check logic
	return true
}`

	writeFile(filepath.Join(projectName, "app", "models", "user.go"), content)
}