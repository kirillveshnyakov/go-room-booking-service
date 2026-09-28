package room

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/usecase/room/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestRoomService_Create(t *testing.T) {
	admin := entity.Identity{UserID: uuid.New(), SessionID: uuid.New(), Role: entity.UserRoleAdmin}
	user := admin
	user.Role = entity.UserRoleUser
	params := port.CreateRoomParams{Name: "Room", Description: "Description", Capacity: 10}
	createdID := uuid.New()
	repositoryErr := errors.New("database unavailable")

	tests := []struct {
		name             string
		actor            entity.Identity
		params           port.CreateRoomParams
		repoErr, wantErr error
	}{
		{name: "success", actor: admin, params: params},
		{name: "normalized fields", actor: admin, params: port.CreateRoomParams{Name: "  Room\n", Description: " Description ", Capacity: 10}},
		{name: "empty description", actor: admin, params: port.CreateRoomParams{Name: "Room", Capacity: 1}},
		{name: "user forbidden", actor: user, params: params, wantErr: errs.ErrForbidden},
		{name: "invalid identity", params: params, wantErr: errs.ErrForbidden},
		{name: "missing session ID", actor: entity.Identity{UserID: admin.UserID, Role: entity.UserRoleAdmin}, params: params, wantErr: errs.ErrForbidden},
		{name: "invalid role", actor: entity.Identity{UserID: admin.UserID, SessionID: admin.SessionID, Role: "owner"}, params: params, wantErr: errs.ErrForbidden},
		{name: "empty name", actor: admin, params: port.CreateRoomParams{Capacity: 1}, wantErr: errs.ErrRoomNameRequired},
		{name: "whitespace name", actor: admin, params: port.CreateRoomParams{Name: " \n ", Capacity: 1}, wantErr: errs.ErrRoomNameRequired},
		{name: "zero capacity", actor: admin, params: port.CreateRoomParams{Name: "Room"}, wantErr: errs.ErrRoomCapacityInvalid},
		{name: "negative capacity", actor: admin, params: port.CreateRoomParams{Name: "Room", Capacity: -1}, wantErr: errs.ErrRoomCapacityInvalid},
		{name: "duplicate name", actor: admin, params: params, repoErr: errs.ErrRoomNameAlreadyExists, wantErr: errs.ErrRoomNameAlreadyExists},
		{name: "wrapped duplicate name", actor: admin, params: params, repoErr: fmt.Errorf("create: %w", errs.ErrRoomNameAlreadyExists), wantErr: errs.ErrRoomNameAlreadyExists},
		{name: "technical error", actor: admin, params: params, repoErr: repositoryErr, wantErr: repositoryErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockroomRepository(ctrl)
			expected := entity.Room{Name: strings.TrimSpace(tt.params.Name), Description: strings.TrimSpace(tt.params.Description), Capacity: tt.params.Capacity}
			created := expected
			created.ID = createdID
			if tt.wantErr == nil || tt.repoErr != nil {
				repo.EXPECT().Create(gomock.Any(), expected).Return(created, tt.repoErr)
			}
			service := NewRoomService(repo, zap.NewNop())
			got, err := service.Create(context.Background(), tt.actor, tt.params)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, entity.Room{}, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, created, got)
		})
	}
}

func TestRoomService_List(t *testing.T) {
	user := entity.Identity{UserID: uuid.New(), SessionID: uuid.New(), Role: entity.UserRoleUser}
	admin := user
	admin.Role = entity.UserRoleAdmin
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name             string
		actor            entity.Identity
		wantList         []entity.Room
		repoErr, wantErr error
	}{
		{name: "user success", actor: user, wantList: []entity.Room{{ID: uuid.New()}}},
		{name: "admin success", actor: admin, wantList: []entity.Room{{ID: uuid.New()}}},
		{name: "empty result", actor: user, wantList: []entity.Room{}},
		{name: "invalid identity", wantErr: errs.ErrForbidden},
		{name: "missing session ID", actor: entity.Identity{UserID: user.UserID, Role: entity.UserRoleUser}, wantErr: errs.ErrForbidden},
		{name: "invalid role", actor: entity.Identity{UserID: user.UserID, SessionID: user.SessionID, Role: "owner"}, wantErr: errs.ErrForbidden},
		{name: "technical error", actor: user, repoErr: repositoryErr, wantErr: repositoryErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewMockroomRepository(gomock.NewController(t))
			if tt.wantErr == nil || tt.repoErr != nil {
				repo.EXPECT().List(gomock.Any()).Return(tt.wantList, tt.repoErr)
			}
			got, err := NewRoomService(repo, zap.NewNop()).List(context.Background(), tt.actor)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantList, got)
		})
	}
}
