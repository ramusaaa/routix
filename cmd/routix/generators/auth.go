package generators

import (
	"path/filepath"
	"strings"
)

func GenerateAuthFiles(projectName string, config ProjectConfig) {
	generateAuthController(projectName)
	generateAuthMiddleware(projectName)
	generateAuthService(projectName)
	generateUserModel(projectName)
	generateAuthMigration(projectName)
}

func generateAuthController(projectName string) {
	content := `package controllers

import (
 "errors"
 "github.com/ramusaaa/routix"
 "gorm.io/gorm"
 "{{module}}/app/models"
 "{{module}}/app/services"
)

type AuthController struct {
 BaseController
 authService *services.AuthService
}

func NewAuthController(db *gorm.DB) *AuthController {
 return &AuthController{authService:services.NewAuthService(db)}
}

func (ctrl *AuthController) Register(c *routix.Context) error {
 var input struct {
  Name string ` + "`" + `json:"name"` + "`" + `
  Email string ` + "`" + `json:"email"` + "`" + `
  Password string ` + "`" + `json:"password"` + "`" + `
 }
 if err:=c.ParseJSON(&input);err!=nil {return ctrl.Error(c,400,"Invalid JSON body")}
 if err:=ctrl.authService.Register(input.Email,input.Password,input.Name);err!=nil {
  if errors.Is(err,services.ErrInvalidInput) {return ctrl.Error(c,400,err.Error())}
  if errors.Is(err,services.ErrEmailExists) {return ctrl.Error(c,409,err.Error())}
  return err
 }
 return ctrl.Created(c,map[string]any{"message":"User registered successfully"})
}

func (ctrl *AuthController) Login(c *routix.Context) error {
 var input struct {Email string ` + "`" + `json:"email"` + "`" + `;Password string ` + "`" + `json:"password"` + "`" + `}
 if err:=c.ParseJSON(&input);err!=nil {return ctrl.Error(c,400,"Invalid JSON body")}
 token,err:=ctrl.authService.Login(input.Email,input.Password)
 if errors.Is(err,services.ErrInvalidCredentials) {return ctrl.Error(c,401,"Invalid credentials")}
 if err!=nil {return err}
 return ctrl.Success(c,map[string]any{"token":token,"type":"Bearer","expires_in":86400})
}

func (ctrl *AuthController) Logout(c *routix.Context) error {
 token,err:=services.BearerToken(c.GetHeader("Authorization"))
 if err!=nil {return ctrl.Error(c,401,"Invalid authentication token")}
 if err:=ctrl.authService.Logout(token);err!=nil {
  if errors.Is(err,services.ErrUnauthenticated) {return ctrl.Error(c,401,"Invalid authentication token")}
  return err
 }
 return ctrl.Success(c,map[string]any{"message":"Logged out successfully"})
}

func (ctrl *AuthController) Me(c *routix.Context) error {
 value,ok:=c.Get("user")
 user,valid:=value.(*models.User)
 if !ok || !valid {return ctrl.Error(c,401,"Authentication required")}
 return ctrl.Success(c,user)
}

func (ctrl *AuthController) RefreshToken(c *routix.Context) error {
 current,err:=services.BearerToken(c.GetHeader("Authorization"))
 if err!=nil {return ctrl.Error(c,401,"Invalid authentication token")}
 token,err:=ctrl.authService.RefreshToken(current)
 if errors.Is(err,services.ErrUnauthenticated) {return ctrl.Error(c,401,"Invalid authentication token")}
 if err!=nil {return err}
 return ctrl.Success(c,map[string]any{"token":token,"type":"Bearer","expires_in":86400})
}
`
	writeFile(filepath.Join(projectName, "app", "controllers", "auth_controller.go"), strings.ReplaceAll(content, "{{module}}", projectName))
}

func generateAuthMiddleware(projectName string) {
	content := `package middleware

import (
 "errors"
 "github.com/ramusaaa/routix"
 "gorm.io/gorm"
 "{{module}}/app/services"
)

func auth(db *gorm.DB,optional bool) routix.Middleware {
 service:=services.NewAuthService(db)
 return func(next routix.Handler) routix.Handler {
  return func(c *routix.Context) error {
   if optional && c.GetHeader("Authorization")=="" {return next(c)}
   token,err:=services.BearerToken(c.GetHeader("Authorization"))
   if err!=nil {c.SetHeader("WWW-Authenticate","Bearer");return c.Unauthorized("Invalid authentication token")}
   user,err:=service.Authenticate(token)
   if errors.Is(err,services.ErrUnauthenticated) {c.SetHeader("WWW-Authenticate","Bearer");return c.Unauthorized("Invalid authentication token")}
   if err!=nil {return err}
   c.Set("user",user)
   c.Set("user_id",user.ID)
   return next(c)
  }
 }
}

func Auth(db *gorm.DB) routix.Middleware {return auth(db,false)}
func OptionalAuth(db *gorm.DB) routix.Middleware {return auth(db,true)}
`
	writeFile(filepath.Join(projectName, "app", "middleware", "auth.go"), strings.ReplaceAll(content, "{{module}}", projectName))
}

func generateAuthService(projectName string) {
	content := `package services

import (
 "crypto/rand"
 "encoding/hex"
 "errors"
 "fmt"
 "net/mail"
 "strconv"
 "strings"
 "time"
 "unicode/utf8"

 "github.com/golang-jwt/jwt/v5"
 "golang.org/x/crypto/bcrypt"
 "gorm.io/gorm"
 "{{module}}/app/models"
 "{{module}}/config"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidInput = errors.New("invalid registration details")
var ErrEmailExists = errors.New("email already registered")
var ErrUnauthenticated = errors.New("invalid or expired token")

type AuthClaims struct {
 UserID uint ` + "`" + `json:"user_id"` + "`" + `
 jwt.RegisteredClaims
}

type AuthService struct {
 db *gorm.DB
 cfg *config.Config
}

func NewAuthService(db *gorm.DB) *AuthService {
 if db == nil { panic("authentication requires a database") }
 return &AuthService{db:db,cfg:config.Load()}
}

func (s *AuthService) HashPassword(password string) (string,error) {
 hash,err:=bcrypt.GenerateFromPassword([]byte(password),bcrypt.DefaultCost)
 return string(hash),err
}

func (s *AuthService) CheckPassword(password,hash string) bool {
 return bcrypt.CompareHashAndPassword([]byte(hash),[]byte(password))==nil
}

func (s *AuthService) Register(email,password,name string) error {
 email=strings.ToLower(strings.TrimSpace(email))
 name=strings.TrimSpace(name)
 address,err:=mail.ParseAddress(email)
 if err!=nil || address.Address!=email || len(email)>254 || len(password)<8 || len(password)>72 || utf8.RuneCountInString(name)<1 || utf8.RuneCountInString(name)>100 {return ErrInvalidInput}
 hash,err:=s.HashPassword(password)
 if err!=nil {return err}
 user:=models.User{Name:name,Email:email,Password:hash,IsActive:true}
 if err:=s.db.Create(&user).Error;err!=nil {
  if errors.Is(err,gorm.ErrDuplicatedKey) {return ErrEmailExists}
  var count int64
  if lookupErr:=s.db.Unscoped().Model(&models.User{}).Where("email = ?",email).Count(&count).Error;lookupErr==nil && count>0 {return ErrEmailExists}
  return err
 }
 return nil
}

func (s *AuthService) Login(email,password string) (string,error) {
 var user models.User
 err:=s.db.Where("email = ?",strings.ToLower(strings.TrimSpace(email))).First(&user).Error
 if errors.Is(err,gorm.ErrRecordNotFound) {
  _=bcrypt.CompareHashAndPassword([]byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"),[]byte(password))
  return "",ErrInvalidCredentials
 }
 if err!=nil {return "",err}
 if !s.CheckPassword(password,user.Password) || !user.IsActive {return "",ErrInvalidCredentials}
 return s.GenerateToken(user.ID)
}

func (s *AuthService) issueToken(db *gorm.DB,userID uint) (string,error) {
 var random [32]byte
 if _,err:=rand.Read(random[:]);err!=nil {return "",err}
 now:=time.Now().UTC().Truncate(time.Second)
 expires:=now.Add(24*time.Hour)
 id:=hex.EncodeToString(random[:])
 claims:=AuthClaims{UserID:userID,RegisteredClaims:jwt.RegisteredClaims{
  Subject:strconv.FormatUint(uint64(userID),10),ID:id,Issuer:s.cfg.AppName,
  Audience:jwt.ClaimStrings{s.cfg.AppName},IssuedAt:jwt.NewNumericDate(now),ExpiresAt:jwt.NewNumericDate(expires),
 }}
 token,err:=jwt.NewWithClaims(jwt.SigningMethodHS256,claims).SignedString([]byte(s.cfg.JWTSecret))
 if err!=nil {return "",err}
 session:=models.AuthSession{ID:id,UserID:userID,ExpiresAt:expires}
 if err:=db.Create(&session).Error;err!=nil {return "",err}
 return token,nil
}

func (s *AuthService) GenerateToken(userID uint) (string,error) {
 var user models.User
 if err:=s.db.Where("id = ? AND is_active = ?",userID,true).First(&user).Error;err!=nil {
  if errors.Is(err,gorm.ErrRecordNotFound) {return "",ErrUnauthenticated}
  return "",err
 }
 return s.issueToken(s.db,userID)
}

func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token,error) {
 claims:=&AuthClaims{}
 token,err:=jwt.ParseWithClaims(tokenString,claims,func(token *jwt.Token)(any,error){return []byte(s.cfg.JWTSecret),nil},
  jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),jwt.WithExpirationRequired(),jwt.WithIssuedAt(),jwt.WithIssuer(s.cfg.AppName),jwt.WithAudience(s.cfg.AppName))
 if err!=nil || token==nil || !token.Valid || claims.ID=="" || claims.IssuedAt==nil || claims.UserID==0 || claims.Subject!=strconv.FormatUint(uint64(claims.UserID),10) {return nil,ErrUnauthenticated}
 var session models.AuthSession
 if err:=s.db.Where("id = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?",claims.ID,claims.UserID,time.Now().UTC()).First(&session).Error;err!=nil {
  if errors.Is(err,gorm.ErrRecordNotFound) {return nil,ErrUnauthenticated}
  return nil,err
 }
 if !session.ExpiresAt.Equal(claims.ExpiresAt.Time) {return nil,ErrUnauthenticated}
 return token,nil
}

func (s *AuthService) Authenticate(tokenString string) (*models.User,error) {
 token,err:=s.ValidateToken(tokenString)
 if err!=nil {return nil,err}
 claims:=token.Claims.(*AuthClaims)
 var user models.User
 if err:=s.db.Where("id = ? AND is_active = ?",claims.UserID,true).First(&user).Error;err!=nil {
  if errors.Is(err,gorm.ErrRecordNotFound) {return nil,ErrUnauthenticated}
  return nil,err
 }
 return &user,nil
}

func (s *AuthService) Logout(tokenString string) error {
 token,err:=s.ValidateToken(tokenString)
 if err!=nil {return err}
 result:=s.db.Model(&models.AuthSession{}).Where("id = ? AND revoked_at IS NULL",token.Claims.(*AuthClaims).ID).Update("revoked_at",time.Now().UTC())
 if result.Error!=nil {return result.Error}
 if result.RowsAffected!=1 {return ErrUnauthenticated}
 return nil
}

func (s *AuthService) RefreshToken(tokenString string) (string,error) {
 token,err:=s.ValidateToken(tokenString)
 if err!=nil {return "",err}
 claims:=token.Claims.(*AuthClaims)
 var refreshed string
 err=s.db.Transaction(func(tx *gorm.DB) error {
  var user models.User
  if err:=tx.Where("id = ? AND is_active = ?",claims.UserID,true).First(&user).Error;err!=nil {
   if errors.Is(err,gorm.ErrRecordNotFound) {return ErrUnauthenticated}
   return err
  }
  result:=tx.Model(&models.AuthSession{}).Where("id = ? AND revoked_at IS NULL AND expires_at > ?",claims.ID,time.Now().UTC()).Update("revoked_at",time.Now().UTC())
  if result.Error!=nil {return result.Error}
  if result.RowsAffected!=1 {return ErrUnauthenticated}
  refreshed,err=s.issueToken(tx,claims.UserID)
  return err
 })
 return refreshed,err
}

func BearerToken(header string) (string,error) {
 parts:=strings.Fields(header)
 if len(parts)!=2 || !strings.EqualFold(parts[0],"Bearer") {return "",fmt.Errorf("%w: Bearer token required",ErrUnauthenticated)}
 return parts[1],nil
}
`
	writeFile(filepath.Join(projectName, "app", "services", "auth_service.go"), strings.ReplaceAll(content, "{{module}}", projectName))
}

func generateUserModel(projectName string) {
	content := `package models

import "time"

type User struct {
 BaseModel
 Name string ` + "`" + `gorm:"not null;size:100" json:"name"` + "`" + `
 Email string ` + "`" + `gorm:"uniqueIndex;not null;size:254" json:"email"` + "`" + `
 Password string ` + "`" + `gorm:"not null" json:"-"` + "`" + `
 IsActive bool ` + "`" + `gorm:"default:true" json:"is_active"` + "`" + `
}

func (u *User) TableName() string {return "users"}

type AuthSession struct {
 ID string ` + "`" + `gorm:"primaryKey;size:64" json:"-"` + "`" + `
 UserID uint ` + "`" + `gorm:"not null;index" json:"-"` + "`" + `
 ExpiresAt time.Time ` + "`" + `gorm:"not null;index" json:"-"` + "`" + `
 RevokedAt *time.Time ` + "`" + `gorm:"index" json:"-"` + "`" + `
 User User ` + "`" + `gorm:"constraint:OnDelete:CASCADE" json:"-"` + "`" + `
}
`
	writeFile(filepath.Join(projectName, "app", "models", "user.go"), strings.ReplaceAll(content, "{{module}}", projectName))
}

func generateAuthMigration(projectName string) {
	content := `package migrations

import (
 "gorm.io/gorm"
 "{{module}}/app/models"
)

func init() {
 RegisterMigration(&Migration{
  ID:"00000000000000000000000_create_auth_tables",
  Up:func(db *gorm.DB) error {return db.AutoMigrate(&models.User{},&models.AuthSession{})},
  Down:func(db *gorm.DB) error {return db.Migrator().DropTable(&models.AuthSession{},&models.User{})},
 })
}
`
	writeFile(filepath.Join(projectName, "database", "migrations", "00000000000000000000000_create_auth_tables.go"), strings.ReplaceAll(content, "{{module}}", projectName))
}
