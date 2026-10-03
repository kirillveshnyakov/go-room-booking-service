package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/tests/integration/testapp"
	"github.com/stretchr/testify/require"
)

func TestBookingHappyPath(t *testing.T) {
	app := testapp.New(t)
	roomID, slotID, date := seedFutureSlot(t, app)
	registerUser(t, app, "booker@example.com", "123456")
	userToken, _ := loginUser(t, app, app.Client, "booker@example.com", "123456")

	freeSlots := listFreeSlots(t, app, userToken, roomID, date)
	require.Len(t, freeSlots, 1)
	require.Equal(t, slotID.String(), freeSlots[0].ID)

	createResp := app.Do(t, testapp.Request{
		Method:      http.MethodPost,
		Path:        "/bookings/create",
		AccessToken: userToken,
		Body: dto.CreateBookingRequest{
			SlotID: freeSlots[0].ID,
		},
	})
	booking := decodeResponse[dto.CreateBookingResponse](t, createResp, http.StatusCreated).Booking
	bookingID, err := uuid.Parse(booking.ID)
	require.NoError(t, err)
	require.Equal(t, slotID.String(), booking.SlotID)
	require.Equal(t, "active", booking.Status)
	require.Empty(t, listFreeSlots(t, app, userToken, roomID, date))

	myResp := app.Do(t, testapp.Request{
		Method: http.MethodGet, Path: "/bookings/my", AccessToken: userToken,
	})
	myBookings := decodeResponse[dto.ListMyBookingsResponse](t, myResp, http.StatusOK).Bookings
	require.Len(t, myBookings, 1)
	require.Equal(t, booking.ID, myBookings[0].ID)
	require.Equal(t, booking.UserID, myBookings[0].UserID)

	cancelResp := app.Do(t, testapp.Request{
		Method:      http.MethodPost,
		Path:        fmt.Sprintf("/bookings/%s/cancel", bookingID),
		AccessToken: userToken,
	})
	cancelled := decodeResponse[dto.CancelBookingResponse](t, cancelResp, http.StatusOK).Booking
	require.Equal(t, booking.ID, cancelled.ID)
	require.Equal(t, "cancelled", cancelled.Status)

	myResp = app.Do(t, testapp.Request{
		Method: http.MethodGet, Path: "/bookings/my", AccessToken: userToken,
	})
	myBookings = decodeResponse[dto.ListMyBookingsResponse](t, myResp, http.StatusOK).Bookings
	require.Len(t, myBookings, 1)
	require.Equal(t, booking.ID, myBookings[0].ID)
	require.Equal(t, "cancelled", myBookings[0].Status)

	freeSlots = listFreeSlots(t, app, userToken, roomID, date)
	require.Len(t, freeSlots, 1)
	require.Equal(t, slotID.String(), freeSlots[0].ID)
}

func TestBookingConflict(t *testing.T) {
	app := testapp.New(t)
	_, slotID, _ := seedFutureSlot(t, app)
	registerUser(t, app, "first@example.com", "123456")
	firstToken, _ := loginUser(t, app, app.Client, "first@example.com", "123456")
	registerUser(t, app, "second@example.com", "123456")
	secondToken, _ := loginUser(t, app, app.Client, "second@example.com", "123456")

	firstBooking := createBooking(t, app, firstToken, slotID)
	require.Equal(t, slotID.String(), firstBooking.SlotID)
	require.Equal(t, "active", firstBooking.Status)

	secondResp := app.Do(t, testapp.Request{
		Method:      http.MethodPost,
		Path:        "/bookings/create",
		AccessToken: secondToken,
		Body: dto.CreateBookingRequest{
			SlotID: slotID.String(),
		},
	})
	requireHTTPError(t, secondResp, http.StatusConflict, httperror.CodeSlotAlreadyBooked)
}

func TestBookingListMyReturnsAllBookings(t *testing.T) {
	app := testapp.New(t)
	const email = "history@example.com"
	registerUser(t, app, email, "123456")
	userToken, _ := loginUser(t, app, app.Client, email, "123456")
	roomID, futureActiveSlotID, date := seedFutureSlot(t, app)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var userID uuid.UUID
	require.NoError(t, app.DB.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&userID))

	pastSlotID := uuid.New()
	futureCancelledSlotID := uuid.New()
	for _, slot := range []struct {
		id    uuid.UUID
		start time.Time
	}{
		{id: pastSlotID, start: date.Add(-7*24*time.Hour + 9*time.Hour)},
		{id: futureCancelledSlotID, start: date.Add(10 * time.Hour)},
	} {
		_, err := app.DB.Exec(ctx,
			"INSERT INTO slots (id, room_id, start_at, end_at) VALUES ($1, $2, $3, $4)",
			slot.id, roomID, slot.start, slot.start.Add(30*time.Minute),
		)
		require.NoError(t, err)
	}

	expected := make(map[string]string, 3)
	for _, booking := range []struct {
		slotID uuid.UUID
		status string
	}{
		{slotID: pastSlotID, status: "active"},
		{slotID: futureActiveSlotID, status: "active"},
		{slotID: futureCancelledSlotID, status: "cancelled"},
	} {
		bookingID := uuid.New()
		_, err := app.DB.Exec(ctx,
			"INSERT INTO bookings (id, slot_id, user_id, status) VALUES ($1, $2, $3, $4::booking_status)",
			bookingID, booking.slotID, userID, booking.status,
		)
		require.NoError(t, err)
		expected[bookingID.String()] = booking.status
	}

	resp := app.Do(t, testapp.Request{
		Method: http.MethodGet, Path: "/bookings/my", AccessToken: userToken,
	})
	bookings := decodeResponse[dto.ListMyBookingsResponse](t, resp, http.StatusOK).Bookings
	require.Len(t, bookings, 3)
	for _, booking := range bookings {
		status, ok := expected[booking.ID]
		require.True(t, ok)
		require.Equal(t, status, booking.Status)
		require.Equal(t, userID.String(), booking.UserID)
		delete(expected, booking.ID)
	}
	require.Empty(t, expected)
}
