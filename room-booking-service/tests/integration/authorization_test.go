package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/tests/integration/testapp"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationBoundaries(t *testing.T) {
	t.Run("user cannot create room", func(t *testing.T) {
		app := testapp.New(t)
		registerUser(t, app, "regular@example.com", "123456")
		userToken, _ := loginUser(t, app, app.Client, "regular@example.com", "123456")

		resp := app.Do(t, testapp.Request{
			Method:      http.MethodPost,
			Path:        "/rooms/create",
			AccessToken: userToken,
			Body: dto.CreateRoomRequest{
				Name: "Restricted room", Capacity: 4,
			},
		})
		requireHTTPError(t, resp, http.StatusForbidden, httperror.CodeForbidden)
	})

	t.Run("other user cannot cancel booking, admin can", func(t *testing.T) {
		app := testapp.New(t)
		_, slotID, _ := seedFutureSlot(t, app)
		registerUser(t, app, "owner@example.com", "123456")
		ownerToken, _ := loginUser(t, app, app.Client, "owner@example.com", "123456")
		booking := createBooking(t, app, ownerToken, slotID)

		registerUser(t, app, "other@example.com", "123456")
		otherToken, _ := loginUser(t, app, app.Client, "other@example.com", "123456")
		cancelPath := fmt.Sprintf("/bookings/%s/cancel", booking.ID)
		otherResp := app.Do(t, testapp.Request{
			Method: http.MethodPost, Path: cancelPath, AccessToken: otherToken,
		})
		requireHTTPError(t, otherResp, http.StatusForbidden, httperror.CodeForbidden)

		ownerResp := app.Do(t, testapp.Request{
			Method: http.MethodGet, Path: "/bookings/my", AccessToken: ownerToken,
		})
		ownerBookings := decodeResponse[dto.ListMyBookingsResponse](t, ownerResp, http.StatusOK).Bookings
		require.Len(t, ownerBookings, 1)
		require.Equal(t, booking.ID, ownerBookings[0].ID)
		require.Equal(t, "active", ownerBookings[0].Status)

		adminToken := adminAccessToken(t, app)
		adminResp := app.Do(t, testapp.Request{
			Method: http.MethodPost, Path: cancelPath, AccessToken: adminToken,
		})
		cancelled := decodeResponse[dto.CancelBookingResponse](t, adminResp, http.StatusOK).Booking
		require.Equal(t, booking.ID, cancelled.ID)
		require.Equal(t, "cancelled", cancelled.Status)
	})
}
