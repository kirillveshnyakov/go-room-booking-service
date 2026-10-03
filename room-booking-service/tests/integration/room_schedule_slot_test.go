package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/tests/integration/testapp"
	"github.com/stretchr/testify/require"
)

func TestRoomScheduleSlotsFlow(t *testing.T) {
	app := testapp.New(t)
	adminToken := adminAccessToken(t, app)
	date := time.Now().UTC().Truncate(24 * time.Hour).Add(72 * time.Hour)

	roomResp := app.Do(t, testapp.Request{
		Method:      http.MethodPost,
		Path:        "/rooms/create",
		AccessToken: adminToken,
		Body: dto.CreateRoomRequest{
			Name: "Main meeting room", Capacity: 8,
		},
	})
	room := decodeResponse[dto.CreateRoomResponse](t, roomResp, http.StatusCreated).Room
	roomID, err := uuid.Parse(room.ID)
	require.NoError(t, err)

	scheduleResp := app.Do(t, testapp.Request{
		Method:      http.MethodPost,
		Path:        fmt.Sprintf("/rooms/%s/schedule/create", roomID),
		AccessToken: adminToken,
		Body: dto.CreateScheduleRequest{Rules: []dto.ScheduleRuleRequest{{
			DayOfWeek: int(entity.GetDayOfWeek(date)),
			StartTime: "09:00",
			EndTime:   "11:00",
		}}},
	})
	schedule := decodeResponse[dto.CreateScheduleResponse](t, scheduleResp, http.StatusCreated).Schedule
	require.Equal(t, room.ID, schedule.RoomID)
	require.Len(t, schedule.Rules, 1)

	slots := listFreeSlots(t, app, adminToken, roomID, date)
	require.NotEmpty(t, slots)
	windowStart := date.Add(9 * time.Hour)
	windowEnd := date.Add(11 * time.Hour)
	for _, slot := range slots {
		_, err = uuid.Parse(slot.ID)
		require.NoError(t, err)
		require.Equal(t, room.ID, slot.RoomID)
		require.True(t, slot.StartAt.Before(slot.EndAt))
		require.Equal(t, date.Format("2006-01-02"), slot.StartAt.UTC().Format("2006-01-02"))
		require.False(t, slot.StartAt.Before(windowStart))
		require.False(t, slot.EndAt.After(windowEnd))
	}
}
