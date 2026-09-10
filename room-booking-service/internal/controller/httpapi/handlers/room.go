package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/mapper"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
)

type roomUsecase interface {
	Create(ctx context.Context, params port.CreateRoomParams) (entity.Room, error)
	List(ctx context.Context) ([]entity.Room, error)
}

type RoomHandler struct {
	roomUsecase roomUsecase
}

func NewRoomHandler(roomUsecase roomUsecase) *RoomHandler {
	return &RoomHandler{
		roomUsecase: roomUsecase,
	}
}

func (h *RoomHandler) Create(c *gin.Context) {
	var request dto.CreateRoomRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	room, err := h.roomUsecase.Create(c.Request.Context(), mapper.CreateRoomRequestToParams(request))
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.CreateRoomResponse{
		Room: mapper.RoomToResponse(room),
	})
}

func (h *RoomHandler) List(c *gin.Context) {
	rooms, err := h.roomUsecase.List(c.Request.Context())
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ListRoomsResponse{
		Rooms: mapper.RoomsToResponse(rooms),
	})
}
