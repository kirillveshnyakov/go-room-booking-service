package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/mapper"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
)

type scheduleUsecase interface {
	Create(ctx context.Context, actor entity.Identity, roomID uuid.UUID, rules []entity.ScheduleRule) (entity.Schedule, error)
}

type ScheduleHandler struct {
	scheduleUsecase scheduleUsecase
}

func NewScheduleHandler(scheduleUsecase scheduleUsecase) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleUsecase: scheduleUsecase,
	}
}

func (h *ScheduleHandler) Create(c *gin.Context) {
	actor, ok := requestctx.Identity(c.Request.Context())
	if !ok {
		writeUnauthorized(c)
		return
	}

	var uri dto.CreateScheduleURI
	if err := c.ShouldBindUri(&uri); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	var request dto.CreateScheduleRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}
	if err := binding.Validator.ValidateStruct(&request); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	roomID, err := uuid.Parse(uri.RoomID)
	if err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	rules, err := mapper.ScheduleRuleRequestsToEntity(request.Rules)
	if err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	schedule, err := h.scheduleUsecase.Create(c.Request.Context(), actor, roomID, rules)
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.CreateScheduleResponse{
		Schedule: mapper.ScheduleToResponse(schedule),
	})
}
