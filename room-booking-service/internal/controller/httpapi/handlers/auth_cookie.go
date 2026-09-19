package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type RefreshCookieConfig struct {
	Name     string
	Path     string
	Secure   bool
	SameSite http.SameSite
	TTL      time.Duration
}

func (h *AuthHandler) setRefreshCookie(
	c *gin.Context,
	refreshToken string,
) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.refreshCookieConfig.Name,
		Value:    refreshToken,
		Path:     h.refreshCookieConfig.Path,
		Expires:  time.Now().UTC().Add(h.refreshCookieConfig.TTL),
		MaxAge:   int(h.refreshCookieConfig.TTL.Seconds()),
		Secure:   h.refreshCookieConfig.Secure,
		HttpOnly: true,
		SameSite: h.refreshCookieConfig.SameSite,
	})
}

func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.refreshCookieConfig.Name,
		Value:    "",
		Path:     h.refreshCookieConfig.Path,
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		Secure:   h.refreshCookieConfig.Secure,
		HttpOnly: true,
		SameSite: h.refreshCookieConfig.SameSite,
	})
}
