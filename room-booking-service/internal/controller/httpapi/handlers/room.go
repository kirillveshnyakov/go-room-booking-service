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
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
)

type roomUsecase interface {
	Create(ctx context.Context, actor entity.Identity, params port.CreateRoomParams) (entity.Room, error)
	List(ctx context.Context, actor entity.Identity) ([]entity.Room, error)
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
	actor, ok := requestctx.Identity(c.Request.Context())
	if !ok {
		writeUnauthorized(c)
		return
	}

	var request dto.CreateRoomRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	room, err := h.roomUsecase.Create(c.Request.Context(), actor, mapper.CreateRoomRequestToParams(request))
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.CreateRoomResponse{
		Room: mapper.RoomToResponse(room),
	})
}

func (h *RoomHandler) List(c *gin.Context) {
	actor, ok := requestctx.Identity(c.Request.Context())
	if !ok {
		writeUnauthorized(c)
		return
	}

	rooms, err := h.roomUsecase.List(c.Request.Context(), actor)
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ListRoomsResponse{
		Rooms: mapper.RoomsToResponse(rooms),
	})
}
