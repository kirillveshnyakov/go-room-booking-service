package room

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
	"go.uber.org/zap"
)

type (
	DB interface {
		Create(ctx context.Context, room entity.Room) (entity.Room, error)
		GetByID(ctx context.Context, roomID uuid.UUID) (entity.Room, error)
		List(ctx context.Context) ([]entity.Room, error)
	}

	Cache interface {
		Get(ctx context.Context, key string) ([]byte, bool, error)
		Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
		Delete(ctx context.Context, key string) error
	}
)

const (
	roomsListKeyStr = "rooms:list:v1"
	roomByIDPrefix  = "rooms:id:v1:"

	cacheReadTimeout       = 150 * time.Millisecond
	cacheWriteTimeout      = 150 * time.Millisecond
	cacheInvalidateTimeout = 250 * time.Millisecond

	roomsListTTL    = 5 * time.Minute
	roomByIDTTL     = 5 * time.Minute
	roomNotFoundTTL = 15 * time.Second
)

type cachedRoomRepository struct {
	db     DB
	cache  Cache
	logger *zap.Logger
}

func NewCachedRoomRepository(db DB, cache Cache, logger *zap.Logger) *cachedRoomRepository {
	return &cachedRoomRepository{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

func (repo *cachedRoomRepository) Create(
	ctx context.Context,
	room entity.Room,
) (entity.Room, error) {
	log := requestctx.LoggerOrDefault(ctx, repo.logger).Named("room_cached_repository")
	createdRoom, err := repo.db.Create(ctx, room)
	if err != nil {
		return entity.Room{}, err
	}

	invalidateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cacheInvalidateTimeout)
	err = repo.cache.Delete(invalidateCtx, repo.roomsListKey())
	cancel()
	if err != nil {
		log.Warn("failed to delete rooms list cache", zap.Error(err))
	}

	return createdRoom, nil
}

func (repo *cachedRoomRepository) GetByID(
	ctx context.Context,
	roomID uuid.UUID,
) (entity.Room, error) {
	log := requestctx.LoggerOrDefault(ctx, repo.logger).Named("room_cached_repository")

	cacheCtx, cancel := context.WithTimeout(ctx, cacheReadTimeout)
	payload, ok, cacheErr := repo.cache.Get(cacheCtx, repo.roomKey(roomID))
	cancel()
	if cacheErr != nil {
		log.Warn("failed to get room by id cache", zap.Error(cacheErr))
		return repo.loadAndCacheRoom(ctx, roomID)
	}

	if !ok {
		return repo.loadAndCacheRoom(ctx, roomID)
	}

	room := entity.Room{}
	err := json.Unmarshal(payload, &room)
	if err != nil {
		log.Warn("failed to unmarshal room by id cache", zap.Error(err))
		return repo.loadAndCacheRoom(ctx, roomID)
	}
	return room, nil
}

func (repo *cachedRoomRepository) List(
	ctx context.Context,
) ([]entity.Room, error) {
	log := requestctx.LoggerOrDefault(ctx, repo.logger).Named("room_cached_repository")

	cacheCtx, cancel := context.WithTimeout(ctx, cacheReadTimeout)
	payload, ok, cacheErr := repo.cache.Get(cacheCtx, repo.roomsListKey())
	cancel()
	if cacheErr != nil {
		log.Warn("failed to get rooms list cache", zap.Error(cacheErr))
		return repo.loadAndCacheRooms(ctx)
	}

	if !ok {
		return repo.loadAndCacheRooms(ctx)
	}

	rooms := make([]entity.Room, 0)
	err := json.Unmarshal(payload, &rooms)
	if err != nil {
		log.Warn("failed to unmarshal rooms list cache", zap.Error(err))
		return repo.loadAndCacheRooms(ctx)
	}
	return rooms, nil
}

func (repo *cachedRoomRepository) loadAndCacheRoom(
	ctx context.Context,
	roomID uuid.UUID,
) (entity.Room, error) {
	if err := ctx.Err(); err != nil {
		return entity.Room{}, err
	}
	log := requestctx.LoggerOrDefault(ctx, repo.logger).Named("room_cached_repository")
	dbRoom, dbErr := repo.db.GetByID(ctx, roomID)
	if dbErr != nil {
		return entity.Room{}, dbErr
	}

	payload, err := json.Marshal(dbRoom)
	if err != nil {
		log.Warn("failed to marshal json room from db", zap.Error(err))
		return dbRoom, nil
	}

	cacheCtx, cancel := context.WithTimeout(ctx, cacheWriteTimeout)
	err = repo.cache.Set(cacheCtx, repo.roomKey(roomID), payload, roomByIDTTL)
	cancel()
	if err != nil {
		log.Warn("failed to set room by id cache", zap.Error(err))
		return dbRoom, nil
	}

	return dbRoom, nil
}

func (repo *cachedRoomRepository) loadAndCacheRooms(
	ctx context.Context,
) ([]entity.Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	log := requestctx.LoggerOrDefault(ctx, repo.logger).Named("room_cached_repository")
	dbRooms, dbErr := repo.db.List(ctx)
	if dbErr != nil {
		return nil, dbErr
	}

	payload, err := json.Marshal(dbRooms)
	if err != nil {
		log.Warn("failed to marshal json rooms list from db", zap.Error(err))
		return dbRooms, nil
	}

	cacheCtx, cancel := context.WithTimeout(ctx, cacheWriteTimeout)
	err = repo.cache.Set(cacheCtx, repo.roomsListKey(), payload, roomsListTTL)
	cancel()
	if err != nil {
		log.Warn("failed to set rooms list cache", zap.Error(err))
		return dbRooms, nil
	}

	return dbRooms, nil
}

func (repo *cachedRoomRepository) roomKey(roomID uuid.UUID) string {
	return roomByIDPrefix + roomID.String()
}

func (repo *cachedRoomRepository) roomsListKey() string {
	return roomsListKeyStr
}
