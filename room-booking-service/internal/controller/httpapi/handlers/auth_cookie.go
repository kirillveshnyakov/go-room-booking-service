package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	refreshCookieName = "refresh_token"
	refreshCookiePath = "/refresh"
)

type RefreshCookieConfig struct {
	Secure bool
}

func (h *AuthHandler) setRefreshCookie(
	c *gin.Context,
	refreshToken string,
	expiresAt time.Time,
) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     refreshCookiePath,
		Expires:  expiresAt.UTC(),
		MaxAge:   maxAge,
		Secure:   h.refreshCookieConfig.Secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		Secure:   h.refreshCookieConfig.Secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
