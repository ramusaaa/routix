package routix

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SecurityHeadersConfig struct {
	ContentSecurityPolicy string
	ReferrerPolicy        string
	HSTSMaxAge            int
}

func SecureHeaders(configs ...SecurityHeadersConfig) Middleware {
	config := SecurityHeadersConfig{ContentSecurityPolicy: "default-src 'self'; frame-ancestors 'none'; base-uri 'self'", ReferrerPolicy: "strict-origin-when-cross-origin", HSTSMaxAge: 31536000}
	if len(configs) > 1 {
		panic("routix: expected one security headers configuration")
	}
	if len(configs) == 1 {
		config = configs[0]
	}
	return func(next Handler) Handler {
		return func(c *Context) error {
			h := c.Response.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			if config.ContentSecurityPolicy != "" {
				h.Set("Content-Security-Policy", config.ContentSecurityPolicy)
			}
			if config.ReferrerPolicy != "" {
				h.Set("Referrer-Policy", config.ReferrerPolicy)
			}
			if c.IsSecure() && config.HSTSMaxAge > 0 {
				h.Set("Strict-Transport-Security", fmt.Sprintf("max-age=%d", config.HSTSMaxAge))
			}
			return next(c)
		}
	}
}

type CSRFConfig struct {
	Key                  []byte
	CookieName           string
	HeaderName           string
	AllowInsecureCookies bool
	SessionID            func(*Context) string
}

func CSRF(configs ...CSRFConfig) Middleware {
	if len(configs) > 1 {
		panic("routix: expected one CSRF configuration")
	}
	var config CSRFConfig
	if len(configs) == 1 {
		config = configs[0]
	}
	if len(config.Key) == 0 {
		config.Key = make([]byte, 32)
		if _, err := rand.Read(config.Key); err != nil {
			panic(err)
		}
	}
	if len(config.Key) < 32 {
		panic("routix: CSRF key must contain at least 32 bytes")
	}
	config.Key = append([]byte(nil), config.Key...)
	if config.CookieName == "" {
		config.CookieName = "__Host-routix_csrf"
		if config.AllowInsecureCookies {
			config.CookieName = "routix_csrf"
		}
	}
	if config.AllowInsecureCookies && strings.HasPrefix(config.CookieName, "__Host-") {
		panic("routix: __Host- cookies require HTTPS")
	}
	if config.HeaderName == "" {
		config.HeaderName = "X-CSRF-Token"
	}
	signature := func(c *Context, nonce string) []byte {
		mac := hmac.New(sha256.New, config.Key)
		session := ""
		if config.SessionID != nil {
			session = config.SessionID(c)
		}
		fmt.Fprintf(mac, "%d:%s:%s", len(session), session, nonce)
		return mac.Sum(nil)
	}
	valid := func(c *Context, token string) bool {
		nonce, encoded, ok := strings.Cut(token, ".")
		if !ok {
			return false
		}
		data, err := base64.RawURLEncoding.DecodeString(nonce)
		if err != nil || len(data) != 32 {
			return false
		}
		signed, err := base64.RawURLEncoding.DecodeString(encoded)
		return err == nil && hmac.Equal(signed, signature(c, nonce))
	}
	return func(next Handler) Handler {
		return func(c *Context) error {
			safe := c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions
			cookie, err := c.Request.Cookie(config.CookieName)
			token := ""
			if err == nil {
				token = cookie.Value
			}
			if !safe {
				if origin := c.GetHeader("Origin"); origin != "" {
					parsed, err := url.Parse(origin)
					scheme := "http"
					if c.IsSecure() {
						scheme = "https"
					}
					if err != nil || parsed.Scheme != scheme || !sameCSRFHost(parsed.Host, c.Request.Host, scheme) || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
						return c.Forbidden("Invalid CSRF origin")
					}
				}
				supplied := c.GetHeader(config.HeaderName)
				if c.GetHeader("Sec-Fetch-Site") == "cross-site" || !valid(c, token) || subtle.ConstantTimeCompare([]byte(token), []byte(supplied)) != 1 {
					return c.Forbidden("Invalid CSRF token")
				}
			} else if !valid(c, token) {
				nonce := make([]byte, 32)
				if _, err := rand.Read(nonce); err != nil {
					return err
				}
				encoded := base64.RawURLEncoding.EncodeToString(nonce)
				token = encoded + "." + base64.RawURLEncoding.EncodeToString(signature(c, encoded))
				http.SetCookie(c.Response, &http.Cookie{Name: config.CookieName, Value: token, Path: "/", Secure: !config.AllowInsecureCookies, HttpOnly: false, SameSite: http.SameSiteLaxMode})
				c.SetHeader("Cache-Control", "no-store")
			}
			c.Set("csrf_token", token)
			return next(c)
		}
	}
}

func sameCSRFHost(origin, host, scheme string) bool {
	normalize := func(value string) string {
		parsed, err := url.Parse(scheme + "://" + value)
		if err != nil {
			return ""
		}
		port := parsed.Port()
		if scheme == "https" && port == "443" || scheme == "http" && port == "80" {
			port = ""
		}
		return strings.ToLower(parsed.Hostname()) + ":" + port
	}
	return normalize(origin) == normalize(host)
}

func (api *APIBuilder) SecurityHeaders(configs ...SecurityHeadersConfig) *APIBuilder {
	api.router.Use(SecureHeaders(configs...))
	return api
}
func (api *APIBuilder) CSRF(configs ...CSRFConfig) *APIBuilder {
	api.router.Use(CSRF(configs...))
	return api
}
