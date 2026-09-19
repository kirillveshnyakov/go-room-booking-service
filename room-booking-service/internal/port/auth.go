package port

import "time"

type AuthTokens struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}
