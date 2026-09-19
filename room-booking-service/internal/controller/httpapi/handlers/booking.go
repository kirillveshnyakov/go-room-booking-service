package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/mapper"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
)

type bookingUsecase interface {
	Create(ctx context.Context, actor entity.Identity, params port.CreateBookingParams) (entity.Booking, error)
	List(ctx context.Context, actor entity.Identity, page int, pageSize int) ([]entity.Booking, int64, error)
	ListMy(ctx context.Context, actor entity.Identity) ([]entity.Booking, error)
	Cancel(ctx context.Context, actor entity.Identity, bookingID uuid.UUID) (entity.Booking, error)
}

type BookingHandler struct {
	bookingUsecase bookingUsecase
}

func NewBookingHandler(bookingUsecase bookingUsecase) *BookingHandler {
	return &BookingHandler{
		bookingUsecase: bookingUsecase,
	}
}

func (h *BookingHandler) Create(c *gin.Context) {
	actor, ok := requestctx.Identity(c.Request.Context())
	if !ok {
		writeUnauthorized(c)
		return
	}
	var request dto.CreateBookingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	params, err := mapper.CreateBookingRequestToParams(request)
	if err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	booking, err := h.bookingUsecase.Create(c.Request.Context(), actor, params)
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.CreateBookingResponse{
		Booking: mapper.BookingToResponse(booking),
	})
}

func (h *BookingHandler) List(c *gin.Context) {
	actor, ok := requestctx.Identity(c.Request.Context())
	if !ok {
		writeUnauthorized(c)
		return
	}

	var query dto.ListBookingsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	bookings, total, err := h.bookingUsecase.List(c.Request.Context(), actor, query.Page, query.PageSize)
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ListBookingsResponse{
		Bookings: mapper.BookingsToResponse(bookings),
		Pagination: dto.PaginationResponse{
			Page:     query.Page,
			PageSize: query.PageSize,
			Total:    total,
		},
	})
}

func (h *BookingHandler) My(c *gin.Context) {
	actor, ok := requestctx.Identity(c.Request.Context())
	if !ok {
		writeUnauthorized(c)
		return
	}

	bookings, err := h.bookingUsecase.ListMy(c.Request.Context(), actor)
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ListMyBookingsResponse{
		Bookings: mapper.BookingsToResponse(bookings),
	})
}

func (h *BookingHandler) Cancel(c *gin.Context) {
	actor, ok := requestctx.Identity(c.Request.Context())
	if !ok {
		writeUnauthorized(c)
		return
	}

	var uri dto.CancelBookingURI
	if err := c.ShouldBindUri(&uri); err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	bookingID, err := mapper.CancelBookingURIToBookingID(uri)
	if err != nil {
		httperror.WriteError(c, http.StatusBadRequest, httperror.CodeInvalidRequest, "invalid request")
		return
	}

	booking, err := h.bookingUsecase.Cancel(c.Request.Context(), actor, bookingID)
	if err != nil {
		httperror.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CancelBookingResponse{
		Booking: mapper.BookingToResponse(booking),
	})
}
