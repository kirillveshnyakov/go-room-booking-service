package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/mapper"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

type (
	authUsecase interface {
		Register(ctx context.Context, email string, password string) (entity.User, error)
		Login(ctx context.Context, email string, password string) (string, error)
		DummyLogin(ctx context.Context, role entity.UserRole) (string, error)
	}
)

type AuthHandler struct {
	authUsecase authUsecase
}

func NewAuthHandler(authUsecase authUsecase) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
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

	token, err := h.authUsecase.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.TokenResponse{Token: token})
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
