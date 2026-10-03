package testapp

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/app"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/tests/util/testdb"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type TestApp struct {
	Server *httptest.Server
	Client *http.Client
	DB     *pgxpool.Pool
}

func New(t *testing.T) *TestApp {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db := testdb.New(t)

	logger := zap.NewNop()

	cfg := newTestConfig()
	require.NoError(t, cfg.Validate())

	handler, err := app.NewApplication(cfg, db.Pool, logger)
	require.NoError(t, err)

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := server.Client()
	client.Jar = jar
	client.Timeout = 10 * time.Second

	return &TestApp{
		Server: server,
		Client: client,
		DB:     db.Pool,
	}
}
