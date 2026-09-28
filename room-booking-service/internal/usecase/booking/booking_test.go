package booking

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/usecase/booking/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestBookingService_Create(t *testing.T) {
	userID := uuid.New()
	slotID := uuid.New()

	user := entity.Identity{
		UserID:    userID,
		SessionID: uuid.New(),
		Role:      entity.UserRoleUser,
	}

	admin := entity.Identity{
		UserID:    uuid.New(),
		SessionID: uuid.New(),
		Role:      entity.UserRoleAdmin,
	}

	expectedBooking := entity.Booking{
		ID:     uuid.New(),
		SlotID: slotID,
		UserID: userID,
		Status: entity.BookingStatusActive,
	}
	expectedBookingWithLink := expectedBooking
	expectedBookingWithLink.ConferenceLink = "https://meet.example.com/test-conference"

	linkGenerateErr := errors.New("link generate error")
	repositoryErr := errors.New("database unavailable")
	repositoryErrorMocks := func(err error) func(*mocks.MockbookingRepository, *mocks.MockconferenceLinkGenerator) {
		return func(repo *mocks.MockbookingRepository, linkGenerator *mocks.MockconferenceLinkGenerator) {
			repo.EXPECT().
				Create(gomock.Any(), slotID, userID, "").
				Return(entity.Booking{}, err)
		}
	}

	tests := []struct {
		name       string
		actor      entity.Identity
		params     port.CreateBookingParams
		setupMocks func(
			repo *mocks.MockbookingRepository,
			linkGenerator *mocks.MockconferenceLinkGenerator,
		)
		wantBooking entity.Booking
		wantErr     error
	}{
		{
			name:  "success without conference link",
			actor: user,
			params: port.CreateBookingParams{
				SlotID:               slotID,
				CreateConferenceLink: false,
			},
			setupMocks: func(
				repo *mocks.MockbookingRepository,
				linkGenerator *mocks.MockconferenceLinkGenerator,
			) {
				repo.EXPECT().
					Create(
						gomock.Any(),
						slotID,
						userID,
						"",
					).
					Return(expectedBooking, nil)
			},
			wantBooking: expectedBooking,
		},
		{
			name:  "success with conference link",
			actor: user,
			params: port.CreateBookingParams{
				SlotID:               slotID,
				CreateConferenceLink: true,
			},
			setupMocks: func(
				repo *mocks.MockbookingRepository,
				linkGenerator *mocks.MockconferenceLinkGenerator,
			) {
				linkGenerator.EXPECT().
					Generate(gomock.Any()).
					Return(expectedBookingWithLink.ConferenceLink, nil)

				repo.EXPECT().
					Create(
						gomock.Any(),
						slotID,
						userID,
						expectedBookingWithLink.ConferenceLink,
					).
					Return(expectedBookingWithLink, nil)
			},

			wantBooking: expectedBookingWithLink,
		},
		{
			name:    "admin forbidden",
			actor:   admin,
			params:  port.CreateBookingParams{SlotID: slotID},
			wantErr: errs.ErrForbidden,
		},
		{
			name:    "missing user ID forbidden",
			actor:   entity.Identity{SessionID: user.SessionID, Role: entity.UserRoleUser},
			params:  port.CreateBookingParams{SlotID: slotID, CreateConferenceLink: true},
			wantErr: errs.ErrForbidden,
		},
		{
			name:    "missing session ID forbidden",
			actor:   entity.Identity{UserID: userID, Role: entity.UserRoleUser},
			params:  port.CreateBookingParams{SlotID: slotID, CreateConferenceLink: true},
			wantErr: errs.ErrForbidden,
		},
		{
			name:    "invalid role forbidden",
			actor:   entity.Identity{UserID: userID, SessionID: user.SessionID, Role: entity.UserRole("owner")},
			params:  port.CreateBookingParams{SlotID: slotID, CreateConferenceLink: true},
			wantErr: errs.ErrForbidden,
		},
		{
			name:  "conference link generation error",
			actor: user,
			params: port.CreateBookingParams{
				SlotID:               slotID,
				CreateConferenceLink: true,
			},
			setupMocks: func(
				repo *mocks.MockbookingRepository,
				linkGenerator *mocks.MockconferenceLinkGenerator,
			) {
				linkGenerator.EXPECT().
					Generate(gomock.Any()).
					Return("", linkGenerateErr)
			},
			wantErr: linkGenerateErr,
		},
		{
			name:       "slot already booked",
			actor:      user,
			params:     port.CreateBookingParams{SlotID: slotID},
			setupMocks: repositoryErrorMocks(errs.ErrSlotAlreadyBooked),
			wantErr:    errs.ErrSlotAlreadyBooked,
		},
		{
			name:       "slot not found",
			actor:      user,
			params:     port.CreateBookingParams{SlotID: slotID},
			setupMocks: repositoryErrorMocks(errs.ErrSlotNotFound),
			wantErr:    errs.ErrSlotNotFound,
		},
		{
			name:       "slot in past",
			actor:      user,
			params:     port.CreateBookingParams{SlotID: slotID},
			setupMocks: repositoryErrorMocks(errs.ErrSlotInPast),
			wantErr:    errs.ErrSlotInPast,
		},
		{
			name:       "user not found",
			actor:      user,
			params:     port.CreateBookingParams{SlotID: slotID},
			setupMocks: repositoryErrorMocks(errs.ErrUserNotFound),
			wantErr:    errs.ErrUserNotFound,
		},
		{
			name:       "repository technical error",
			actor:      user,
			params:     port.CreateBookingParams{SlotID: slotID},
			setupMocks: repositoryErrorMocks(repositoryErr),
			wantErr:    repositoryErr,
		},
		{
			name:       "wrapped slot already booked",
			actor:      user,
			params:     port.CreateBookingParams{SlotID: slotID},
			setupMocks: repositoryErrorMocks(fmt.Errorf("create: %w", errs.ErrSlotAlreadyBooked)),
			wantErr:    errs.ErrSlotAlreadyBooked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			repo := mocks.NewMockbookingRepository(ctrl)
			linkGenerator := mocks.NewMockconferenceLinkGenerator(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(repo, linkGenerator)
			}

			service := NewBookingService(
				repo,
				linkGenerator,
				zap.NewNop(),
			)

			got, gotErr := service.Create(context.Background(), tt.actor, tt.params)

			if tt.wantErr != nil {
				require.ErrorIs(t, gotErr, tt.wantErr)
				require.Equal(t, entity.Booking{}, got)
				return
			}

			require.NoError(t, gotErr)
			require.Equal(t, tt.wantBooking, got)
		})
	}
}

func TestBookingService_List(t *testing.T) {
	admin := entity.Identity{UserID: uuid.New(), SessionID: uuid.New(), Role: entity.UserRoleAdmin}
	user := admin
	user.Role = entity.UserRoleUser
	list := []entity.Booking{{ID: uuid.New()}}
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name             string
		actor            entity.Identity
		page, pageSize   int
		wantOffset       int
		wantList         []entity.Booking
		wantTotal        int64
		repoErr, wantErr error
	}{
		{name: "first page", actor: admin, page: 1, pageSize: 10, wantOffset: 0, wantList: list, wantTotal: 25},
		{name: "next page offset", actor: admin, page: 3, pageSize: 10, wantOffset: 20, wantList: list, wantTotal: 25},
		{name: "maximum page size", actor: admin, page: 1, pageSize: 100, wantOffset: 0, wantList: list, wantTotal: 1},
		{name: "minimum page size", actor: admin, page: 1, pageSize: 1, wantOffset: 0, wantList: list, wantTotal: 1},
		{name: "empty result", actor: admin, page: 1, pageSize: 10, wantOffset: 0, wantList: []entity.Booking{}},
		{name: "user forbidden", actor: user, page: 1, pageSize: 10, wantErr: errs.ErrForbidden},
		{name: "invalid identity", page: 1, pageSize: 10, wantErr: errs.ErrForbidden},
		{name: "missing session ID", actor: entity.Identity{UserID: admin.UserID, Role: entity.UserRoleAdmin}, page: 1, pageSize: 10, wantErr: errs.ErrForbidden},
		{name: "invalid role", actor: entity.Identity{UserID: admin.UserID, SessionID: admin.SessionID, Role: "owner"}, page: 1, pageSize: 10, wantErr: errs.ErrForbidden},
		{name: "zero page", actor: admin, pageSize: 10, wantErr: errs.ErrPaginationPageInvalid},
		{name: "negative page", actor: admin, page: -1, pageSize: 10, wantErr: errs.ErrPaginationPageInvalid},
		{name: "zero page size", actor: admin, page: 1, wantErr: errs.ErrPaginationPageSizeInvalid},
		{name: "negative page size", actor: admin, page: 1, pageSize: -1, wantErr: errs.ErrPaginationPageSizeInvalid},
		{name: "oversized page", actor: admin, page: 1, pageSize: 101, wantErr: errs.ErrPaginationPageSizeInvalid},
		{name: "offset overflow", actor: admin, page: math.MaxInt, pageSize: 100, wantErr: errs.ErrPaginationPageInvalid},
		{name: "largest valid offset", actor: admin, page: math.MaxInt/100 + 1, pageSize: 100, wantOffset: math.MaxInt - math.MaxInt%100, wantList: []entity.Booking{}},
		{name: "repository error", actor: admin, page: 1, pageSize: 10, wantOffset: 0, repoErr: repositoryErr, wantErr: repositoryErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockbookingRepository(ctrl)
			link := mocks.NewMockconferenceLinkGenerator(ctrl)
			if tt.wantErr == nil || tt.repoErr != nil {
				repo.EXPECT().List(gomock.Any(), tt.pageSize, tt.wantOffset).Return(tt.wantList, tt.wantTotal, tt.repoErr)
			}
			service := NewBookingService(repo, link, zap.NewNop())
			got, total, err := service.List(context.Background(), tt.actor, tt.page, tt.pageSize)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				require.Zero(t, total)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantList, got)
			require.Equal(t, tt.wantTotal, total)
		})
	}
}

func TestBookingService_ListMy(t *testing.T) {
	user := entity.Identity{UserID: uuid.New(), SessionID: uuid.New(), Role: entity.UserRoleUser}
	admin := user
	admin.Role = entity.UserRoleAdmin
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name             string
		actor            entity.Identity
		wantList         []entity.Booking
		repoErr, wantErr error
	}{
		{name: "success", actor: user, wantList: []entity.Booking{{ID: uuid.New(), UserID: user.UserID}}},
		{name: "empty result", actor: user, wantList: []entity.Booking{}},
		{name: "admin forbidden", actor: admin, wantErr: errs.ErrForbidden},
		{name: "invalid identity", wantErr: errs.ErrForbidden},
		{name: "missing session ID", actor: entity.Identity{UserID: user.UserID, Role: entity.UserRoleUser}, wantErr: errs.ErrForbidden},
		{name: "invalid role", actor: entity.Identity{UserID: user.UserID, SessionID: user.SessionID, Role: "owner"}, wantErr: errs.ErrForbidden},
		{name: "repository error", actor: user, repoErr: repositoryErr, wantErr: repositoryErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockbookingRepository(ctrl)
			link := mocks.NewMockconferenceLinkGenerator(ctrl)
			if tt.wantErr == nil || tt.repoErr != nil {
				repo.EXPECT().ListUserFuture(gomock.Any(), tt.actor.UserID).Return(tt.wantList, tt.repoErr)
			}
			service := NewBookingService(repo, link, zap.NewNop())
			got, err := service.ListMy(context.Background(), tt.actor)
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

func TestBookingService_Cancel(t *testing.T) {
	user := entity.Identity{UserID: uuid.New(), SessionID: uuid.New(), Role: entity.UserRoleUser}
	admin := entity.Identity{UserID: uuid.New(), SessionID: uuid.New(), Role: entity.UserRoleAdmin}
	booking := entity.Booking{ID: uuid.New(), UserID: user.UserID, SlotID: uuid.New(), Status: entity.BookingStatusActive, ConferenceLink: "https://meet.example.com/test"}
	other := booking
	other.UserID = uuid.New()
	cancelled := booking
	cancelled.Status = entity.BookingStatusCancelled
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name                       string
		actor                      entity.Identity
		booking                    entity.Booking
		getErr, cancelErr, wantErr error
	}{
		{name: "user cancels own booking", actor: user, booking: booking},
		{name: "admin cancels another user's booking", actor: admin, booking: booking},
		{name: "already cancelled", actor: user, booking: cancelled},
		{name: "another user's booking forbidden", actor: user, booking: other, wantErr: errs.ErrForbidden},
		{name: "invalid identity", wantErr: errs.ErrForbidden},
		{name: "missing session ID", actor: entity.Identity{UserID: user.UserID, Role: entity.UserRoleUser}, wantErr: errs.ErrForbidden},
		{name: "invalid role", actor: entity.Identity{UserID: user.UserID, SessionID: user.SessionID, Role: "owner"}, wantErr: errs.ErrForbidden},
		{name: "lookup not found", actor: user, getErr: errs.ErrBookingNotFound, wantErr: errs.ErrBookingNotFound},
		{name: "wrapped lookup not found", actor: user, getErr: fmt.Errorf("lookup: %w", errs.ErrBookingNotFound), wantErr: errs.ErrBookingNotFound},
		{name: "lookup technical error", actor: user, getErr: repositoryErr, wantErr: repositoryErr},
		{name: "cancel not found", actor: user, booking: booking, cancelErr: errs.ErrBookingNotFound, wantErr: errs.ErrBookingNotFound},
		{name: "wrapped cancel not found", actor: user, booking: booking, cancelErr: fmt.Errorf("cancel: %w", errs.ErrBookingNotFound), wantErr: errs.ErrBookingNotFound},
		{name: "cancel technical error", actor: user, booking: booking, cancelErr: repositoryErr, wantErr: repositoryErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockbookingRepository(ctrl)
			link := mocks.NewMockconferenceLinkGenerator(ctrl)
			if tt.booking.ID != uuid.Nil || tt.getErr != nil {
				repo.EXPECT().GetByID(gomock.Any(), booking.ID).Return(tt.booking, tt.getErr)
				if tt.getErr == nil && (tt.wantErr == nil || tt.cancelErr != nil) {
					repo.EXPECT().Cancel(gomock.Any(), booking.ID).Return(tt.cancelErr)
				}
			}
			service := NewBookingService(repo, link, zap.NewNop())
			got, err := service.Cancel(context.Background(), tt.actor, booking.ID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, entity.Booking{}, got)
				return
			}
			want := tt.booking
			want.Status = entity.BookingStatusCancelled
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}
