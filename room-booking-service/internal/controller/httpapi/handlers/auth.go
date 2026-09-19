package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/mapper"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
)

type (
	authUsecase interface {
		Register(ctx context.Context, email string, password string) (entity.User, error)
		Login(ctx context.Context, email string, password string) (port.AuthTokens, error)
		Refresh(ctx context.Context, rawRefreshToken string) (port.AuthTokens, error)
		Logout(ctx context.Context, sessionID uuid.UUID) error
		LogoutAll(ctx context.Context, userID uuid.UUID) error
		DummyLogin(ctx context.Context, role entity.UserRole) (string, error)
	}
)

type AuthHandler struct {
	authUsecase         authUsecase
	refreshCookieConfig RefreshCookieConfig
}

func NewAuthHandler(
	authUsecase authUsecase,
	refreshCookieConfig RefreshCookieConfig,
) *AuthHandler {
	return &AuthHandler{
		authUsecase:         authUsecase,
		refreshCookieConfig: refreshCookieConfig,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request dto.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	user, err := h.authUsecase.Register(
		c.Request.Context(),
		request.Email,
		request.Password,
	)
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.RegisterResponse{
		User: mapper.UserToResponse(user),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request dto.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	tokens, err := h.authUsecase.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	c.JSON(http.StatusOK, mapper.AuthTokensToAccessTokenResponse(tokens))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(h.refreshCookieConfig.Name)
	if err != nil {
		h.clearRefreshCookie(c)
		writeUnauthorized(c)
		return
	}

	tokens, err := h.authUsecase.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidRefreshToken) {
			h.clearRefreshCookie(c)
		}

		httperror.HandleError(c, err)
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	c.JSON(http.StatusOK, mapper.AuthTokensToAccessTokenResponse(tokens))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	actor, ok := requestctx.Identity(c.Request.Context())
	if !ok {
		writeUnauthorized(c)
		return
	}

	if err := h.authUsecase.Logout(c.Request.Context(), actor.SessionID); err != nil {
		httperror.HandleError(c, err)
		return
	}

	h.clearRefreshCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) LogoutAll(c *gin.Context) {
	actor, ok := requestctx.Identity(c.Request.Context())
	if !ok {
		writeUnauthorized(c)
		return
	}

	if err := h.authUsecase.LogoutAll(c.Request.Context(), actor.UserID); err != nil {
		httperror.HandleError(c, err)
		return
	}

	h.clearRefreshCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) DummyLogin(c *gin.Context) {
	var request dto.DummyLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	token, err := h.authUsecase.DummyLogin(c.Request.Context(), entity.UserRole(request.Role))
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.TokenResponse{Token: token})
}

func writeUnauthorized(c *gin.Context) {
	httperror.WriteError(
		c,
		http.StatusUnauthorized,
		httperror.CodeUnauthorized,
		"unauthorized",
	)
}
