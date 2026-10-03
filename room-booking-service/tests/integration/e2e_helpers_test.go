package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/password"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/tests/integration/testapp"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func decodeResponse[T any](t *testing.T, resp *http.Response, status int) T {
	t.Helper()
	defer resp.Body.Close()
	require.Equal(t, status, resp.StatusCode)
	var body T
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

func registerUser(t *testing.T, app *testapp.TestApp, email, password string) {
	t.Helper()
	resp := app.Do(t, testapp.Request{
		Method: http.MethodPost,
		Path:   "/register",
		Body: dto.RegisterRequest{
			Email:    email,
			Password: password,
		},
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}

func loginUser(
	t *testing.T,
	app *testapp.TestApp,
	client *http.Client,
	email, password string,
) (string, *http.Cookie) {
	t.Helper()
	resp := app.Do(t, testapp.Request{
		Method: http.MethodPost,
		Path:   "/login",
		Body: dto.LoginRequest{
			Email:    email,
			Password: password,
		},
		Client: client,
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var tokens dto.AccessTokenResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&tokens))
	require.NoError(t, resp.Body.Close())
	require.NotEmpty(t, tokens.AccessToken)
	cookie := refreshCookieFromResponse(resp)
	require.NotNil(t, cookie)
	require.NotEmpty(t, cookie.Value)
	return tokens.AccessToken, cookie
}

func requireHTTPError(t *testing.T, resp *http.Response, status int, code string) {
	t.Helper()
	defer resp.Body.Close()
	require.Equal(t, status, resp.StatusCode)
	var body httperror.ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, code, body.Error.Code)
}

func adminAccessToken(t *testing.T, app *testapp.TestApp) string {
	t.Helper()
	adminID := uuid.New()
	email := adminID.String() + "@admin.example.test"
	const adminPassword = "test-admin-password"

	hasher, err := password.NewHasher(bcrypt.MinCost)
	require.NoError(t, err)
	hash, err := hasher.Hash(adminPassword)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = app.DB.Exec(ctx,
		"INSERT INTO users (id, email, role, password_hash) VALUES ($1, $2, 'admin', $3)",
		adminID, email, hash,
	)
	require.NoError(t, err)

	accessToken, _ := loginUser(t, app, app.Client, email, adminPassword)
	return accessToken
}

func seedFutureSlot(t *testing.T, app *testapp.TestApp) (uuid.UUID, uuid.UUID, time.Time) {
	t.Helper()
	roomID := uuid.New()
	scheduleID := uuid.New()
	slotID := uuid.New()
	date := time.Now().UTC().Truncate(24 * time.Hour).Add(72 * time.Hour)
	start := date.Add(9 * time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := app.DB.Exec(ctx,
		"INSERT INTO rooms (id, name, capacity) VALUES ($1, $2, 10)",
		roomID, "fixture-room-"+roomID.String(),
	)
	require.NoError(t, err)
	_, err = app.DB.Exec(ctx,
		"INSERT INTO schedules (id, room_id) VALUES ($1, $2)",
		scheduleID, roomID,
	)
	require.NoError(t, err)
	_, err = app.DB.Exec(ctx,
		"INSERT INTO schedule_rules (schedule_id, day_of_week, start_at, end_at) VALUES ($1, $2, TIME '09:00', TIME '11:00')",
		scheduleID, int(entity.GetDayOfWeek(date)),
	)
	require.NoError(t, err)
	_, err = app.DB.Exec(ctx,
		"INSERT INTO slots (id, room_id, start_at, end_at) VALUES ($1, $2, $3, $4)",
		slotID, roomID, start, start.Add(30*time.Minute),
	)
	require.NoError(t, err)
	return roomID, slotID, date
}

func listFreeSlots(
	t *testing.T,
	app *testapp.TestApp,
	accessToken string,
	roomID uuid.UUID,
	date time.Time,
) []dto.SlotResponse {
	t.Helper()
	resp := app.Do(t, testapp.Request{
		Method:      http.MethodGet,
		Path:        fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date.UTC().Format("2006-01-02")),
		AccessToken: accessToken,
	})
	return decodeResponse[dto.ListFreeSlotsResponse](t, resp, http.StatusOK).Slots
}

func createBooking(
	t *testing.T,
	app *testapp.TestApp,
	accessToken string,
	slotID uuid.UUID,
) dto.BookingResponse {
	t.Helper()
	resp := app.Do(t, testapp.Request{
		Method:      http.MethodPost,
		Path:        "/bookings/create",
		AccessToken: accessToken,
		Body: dto.CreateBookingRequest{
			SlotID: slotID.String(),
		},
	})
	return decodeResponse[dto.CreateBookingResponse](t, resp, http.StatusCreated).Booking
}
