package requestctx

import (
	"context"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

type identityKey struct{}

func WithIdentity(ctx context.Context, identity entity.Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

func Identity(ctx context.Context) (entity.Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(entity.Identity)
	return identity, ok
}
