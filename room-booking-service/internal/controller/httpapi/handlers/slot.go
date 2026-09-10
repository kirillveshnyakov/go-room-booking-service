package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/mapper"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

type slotUsecase interface {
	ListFree(ctx context.Context, roomID uuid.UUID, targetDate time.Time) ([]entity.Slot, error)
}

type SlotHandler struct {
	slotUsecase slotUsecase
}

func NewSlotHandler(slotUsecase slotUsecase) *SlotHandler {
	return &SlotHandler{
		slotUsecase: slotUsecase,
	}
}

func (h *SlotHandler) List(c *gin.Context) {
	var uri dto.ListFreeSlotsURI
	if err := c.ShouldBindUri(&uri); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	var query dto.ListFreeSlotsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	roomID, err := uuid.Parse(uri.RoomID)
	if err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	targetDate, err := mapper.ListFreeSlotsQueryToDate(query)
	if err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	slots, err := h.slotUsecase.ListFree(c.Request.Context(), roomID, targetDate)
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ListFreeSlotsResponse{
		Slots: mapper.SlotsToResponse(slots),
	})
}
