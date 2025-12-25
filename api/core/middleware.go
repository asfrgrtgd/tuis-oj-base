package core

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

const sessionName = "oj_session"
const sessionMaxAge = 18000 // 5h

// SessionMiddleware ensures a session exists and applies consistent cookie options.
func SessionMiddleware(cfg Config, store *sessions.CookieStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := store.Get(c.Request, sessionName)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "session error")
			c.Abort()
			return
		}

		applySessionOptions(cfg, session)
		// Save to ensure options are persisted even for anonymous users.
		if err := session.Save(c.Request, c.Writer); err != nil {
			respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to persist session")
			c.Abort()
			return
		}

		c.Set("session", session)
		c.Next()
	}
}

// OriginRefererMiddleware validates Origin/Referer against allowed list and sets CORS headers.
func OriginRefererMiddleware(cfg Config) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, o := range cfg.AllowedOrigins {
		allowed[strings.ToLower(o)] = struct{}{}
	}

	isAllowed := func(origin string) bool {
		if origin == "" {
			// Same-origin navigation (no Origin header) is allowed.
			return true
		}
		if len(allowed) == 0 {
			return false
		}
		origin = strings.ToLower(origin)
		_, ok := allowed[origin]
		return ok
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		referer := c.GetHeader("Referer")
		if origin == "" && referer != "" {
			if u, err := url.Parse(referer); err == nil {
				origin = u.Scheme + "://" + u.Host
			}
		}

		// Preflight handling
		if c.Request.Method == http.MethodOptions && origin != "" {
			if !isAllowed(origin) {
				respondError(c, http.StatusForbidden, "FORBIDDEN", "origin not allowed")
				c.Abort()
				return
			}
			setCORSHeaders(c, origin)
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}

		if !isAllowed(origin) {
			respondError(c, http.StatusForbidden, "FORBIDDEN", "origin not allowed")
			c.Abort()
			return
		}
		if origin != "" {
			setCORSHeaders(c, origin)
		}
		c.Next()
	}
}

func setCORSHeaders(c *gin.Context, origin string) {
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Vary", "Origin")
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
	c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
}

// CSRFMiddleware issues and validates a per-session CSRF token.
func CSRFMiddleware(cfg Config, store *sessions.CookieStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionAny, ok := c.Get("session")
		var session *sessions.Session
		var err error
		if ok {
			session, _ = sessionAny.(*sessions.Session)
		}
		if session == nil {
			session, err = store.Get(c.Request, sessionName)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "session error")
				c.Abort()
				return
			}
		}

		token, _ := session.Values["csrf_token"].(string)
		if token == "" {
			token, err = generateCSRFToken()
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to issue csrf token")
				c.Abort()
				return
			}
			session.Values["csrf_token"] = token
			applySessionOptions(cfg, session)
			if err := session.Save(c.Request, c.Writer); err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to persist session")
				c.Abort()
				return
			}
		}

		if !isSafeMethod(c.Request.Method) && !csrfExemptPath(c.Request.URL.Path) {
			header := c.GetHeader("X-CSRF-Token")
			if header == "" || header != token {
				respondError(c, http.StatusForbidden, "FORBIDDEN", "invalid csrf token")
				c.Abort()
				return
			}
		}

		// Expose token so frontend can read and reuse.
		c.Writer.Header().Set("X-CSRF-Token", token)
		c.Next()
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// Paths that intentionally skip CSRF validation (e.g., login).
func csrfExemptPath(path string) bool {
	switch path {
	case "/api/v1/auth/login":
		return true
	default:
		return false
	}
}

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func applySessionOptions(cfg Config, session *sessions.Session) {
	if session.Options == nil {
		session.Options = &sessions.Options{}
	}
	session.Options.Path = "/"
	session.Options.MaxAge = sessionMaxAge
	session.Options.HttpOnly = true
	session.Options.Secure = cfg.CookieSecure
	session.Options.SameSite = sameSiteFromString(cfg.CookieSameSite)
}

func sameSiteFromString(v string) http.SameSite {
	switch strings.ToLower(v) {
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}
