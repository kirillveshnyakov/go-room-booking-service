package integration

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/tests/integration/testapp"
	"github.com/stretchr/testify/require"
)

const refreshCookieName = "refresh_token"

func TestAuth_RegisterLoginAndProtectedAccess(t *testing.T) {
	app := testapp.New(t)

	email := "test@gmail.ru"
	password := "123456"

	registerResp := app.Do(t, testapp.Request{
		Method: http.MethodPost,
		Path:   "/register",
		Body: dto.RegisterRequest{
			Email:    email,
			Password: password,
		},
	})
	require.Equal(t, http.StatusCreated, registerResp.StatusCode)
	require.NoError(t, registerResp.Body.Close())

	loginResp := app.Do(t, testapp.Request{
		Method: http.MethodPost,
		Path:   "/login",
		Body: dto.LoginRequest{
			Email:    email,
			Password: password,
		},
	})
	require.Equal(t, http.StatusOK, loginResp.StatusCode)

	var tokens dto.AccessTokenResponse
	require.NoError(t, json.NewDecoder(loginResp.Body).Decode(&tokens))
	require.NoError(t, loginResp.Body.Close())
	require.NotEmpty(t, tokens.AccessToken)

	unauthorizedResp := app.Do(t, testapp.Request{
		Method: http.MethodGet,
		Path:   "/rooms/list",
	})
	require.Equal(t, http.StatusUnauthorized, unauthorizedResp.StatusCode)
	require.NoError(t, unauthorizedResp.Body.Close())

	protectedResp := app.Do(t, testapp.Request{
		Method:      http.MethodGet,
		Path:        "/rooms/list",
		AccessToken: tokens.AccessToken,
	})
	require.Equal(t, http.StatusOK, protectedResp.StatusCode)
	require.NoError(t, protectedResp.Body.Close())
}

func TestAuth_RegisterValidation(t *testing.T) {
	tests := []struct {
		name    string
		body    any
		rawBody string
	}{
		{
			name: "invalid email",
			body: dto.RegisterRequest{Email: "not-an-email", Password: "123456"},
		},
		{
			name: "missing password",
			body: map[string]string{"email": "test@example.com"},
		},
		{
			name:    "malformed JSON",
			rawBody: `{"email":`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := testapp.New(t)
			resp := app.Do(t, testapp.Request{
				Method:  http.MethodPost,
				Path:    "/register",
				Body:    tt.body,
				RawBody: tt.rawBody,
			})
			requireHTTPError(t, resp, http.StatusBadRequest, httperror.CodeInvalidRequest)
		})
	}
}

func TestAuth_RegisterDuplicateEmail(t *testing.T) {
	app := testapp.New(t)
	email := "duplicate@example.com"
	password := "123456"

	registerUser(t, app, email, password)
	resp := app.Do(t, testapp.Request{
		Method: http.MethodPost,
		Path:   "/register",
		Body: dto.RegisterRequest{
			Email:    email,
			Password: password,
		},
	})
	requireHTTPError(t, resp, http.StatusBadRequest, httperror.CodeInvalidRequest)
}

func TestAuth_LoginInvalidCredentials(t *testing.T) {
	app := testapp.New(t)
	email := "wrong-password@example.com"
	registerUser(t, app, email, "correct-password")

	resp := app.Do(t, testapp.Request{
		Method: http.MethodPost,
		Path:   "/login",
		Body: dto.LoginRequest{
			Email:    email,
			Password: "wrong-password",
		},
	})
	requireHTTPError(t, resp, http.StatusUnauthorized, httperror.CodeUnauthorized)
}

func TestAuth_RefreshRotation(t *testing.T) {
	app := testapp.New(t)
	email := "rotation@example.com"
	password := "123456"
	registerUser(t, app, email, password)
	_, loginCookie := loginUser(t, app, app.Client, email, password)
	require.True(t, loginCookie.HttpOnly)
	require.Equal(t, "/", loginCookie.Path)

	firstCookie := refreshCookieInJar(t, app, app.Client)
	require.NotNil(t, firstCookie)
	firstToken := firstCookie.Value
	require.NotEmpty(t, firstToken)

	refreshResp := app.Do(t, testapp.Request{Method: http.MethodPost, Path: "/refresh"})
	require.Equal(t, http.StatusOK, refreshResp.StatusCode)
	var tokens dto.AccessTokenResponse
	require.NoError(t, json.NewDecoder(refreshResp.Body).Decode(&tokens))
	require.NoError(t, refreshResp.Body.Close())
	require.NotEmpty(t, tokens.AccessToken)

	secondCookie := refreshCookieInJar(t, app, app.Client)
	require.NotNil(t, secondCookie)
	require.NotEmpty(t, secondCookie.Value)
	require.NotEqual(t, firstToken, secondCookie.Value)

	staleClient := newIndependentClient(t, app)
	serverURL, err := url.Parse(app.Server.URL)
	require.NoError(t, err)
	staleClient.Jar.SetCookies(serverURL, []*http.Cookie{{
		Name: refreshCookieName, Value: firstToken, Path: "/",
	}})
	staleResp := app.Do(t, testapp.Request{
		Method: http.MethodPost,
		Path:   "/refresh",
		Client: staleClient,
	})
	requireHTTPError(t, staleResp, http.StatusUnauthorized, httperror.CodeUnauthorized)
}

func TestAuth_Logout(t *testing.T) {
	app := testapp.New(t)
	email := "logout@example.com"
	password := "123456"
	registerUser(t, app, email, password)
	accessToken, _ := loginUser(t, app, app.Client, email, password)
	oldCookie := refreshCookieInJar(t, app, app.Client)
	require.NotNil(t, oldCookie)

	logoutResp := app.Do(t, testapp.Request{
		Method:      http.MethodPost,
		Path:        "/logout",
		AccessToken: accessToken,
	})
	require.Equal(t, http.StatusNoContent, logoutResp.StatusCode)
	clearedCookie := refreshCookieFromResponse(logoutResp)
	require.NotNil(t, clearedCookie)
	require.Empty(t, clearedCookie.Value)
	require.Negative(t, clearedCookie.MaxAge)
	require.NoError(t, logoutResp.Body.Close())
	require.Nil(t, refreshCookieInJar(t, app, app.Client))

	refreshResp := app.Do(t, testapp.Request{Method: http.MethodPost, Path: "/refresh"})
	requireHTTPError(t, refreshResp, http.StatusUnauthorized, httperror.CodeUnauthorized)

	staleClient := newIndependentClient(t, app)
	serverURL, err := url.Parse(app.Server.URL)
	require.NoError(t, err)
	staleClient.Jar.SetCookies(serverURL, []*http.Cookie{{
		Name: refreshCookieName, Value: oldCookie.Value, Path: "/",
	}})
	staleResp := app.Do(t, testapp.Request{
		Method: http.MethodPost,
		Path:   "/refresh",
		Client: staleClient,
	})
	requireHTTPError(t, staleResp, http.StatusUnauthorized, httperror.CodeUnauthorized)
}

func TestAuth_LogoutAll(t *testing.T) {
	app := testapp.New(t)
	email := "logout-all@example.com"
	password := "123456"
	registerUser(t, app, email, password)
	clientA := app.Client
	clientB := newIndependentClient(t, app)
	require.NotSame(t, clientA, clientB)
	require.NotSame(t, clientA.Jar, clientB.Jar)

	accessTokenA, _ := loginUser(t, app, clientA, email, password)
	loginUser(t, app, clientB, email, password)
	cookieA := refreshCookieInJar(t, app, clientA)
	cookieB := refreshCookieInJar(t, app, clientB)
	require.NotNil(t, cookieA)
	require.NotNil(t, cookieB)
	require.NotEqual(t, cookieA.Value, cookieB.Value)

	logoutResp := app.Do(t, testapp.Request{
		Method:      http.MethodPost,
		Path:        "/logout-all",
		AccessToken: accessTokenA,
		Client:      clientA,
	})
	require.Equal(t, http.StatusNoContent, logoutResp.StatusCode)
	require.NoError(t, logoutResp.Body.Close())
	require.Nil(t, refreshCookieInJar(t, app, clientA))
	require.NotNil(t, refreshCookieInJar(t, app, clientB))

	refreshA := app.Do(t, testapp.Request{
		Method: http.MethodPost, Path: "/refresh", Client: clientA,
	})
	requireHTTPError(t, refreshA, http.StatusUnauthorized, httperror.CodeUnauthorized)

	refreshB := app.Do(t, testapp.Request{
		Method: http.MethodPost, Path: "/refresh", Client: clientB,
	})
	requireHTTPError(t, refreshB, http.StatusUnauthorized, httperror.CodeUnauthorized)
}

func newIndependentClient(t *testing.T, app *testapp.TestApp) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := *app.Client
	client.Jar = jar
	return &client
}

func refreshCookieInJar(t *testing.T, app *testapp.TestApp, client *http.Client) *http.Cookie {
	t.Helper()
	require.NotNil(t, client.Jar)
	serverURL, err := url.Parse(app.Server.URL)
	require.NoError(t, err)
	for _, cookie := range client.Jar.Cookies(serverURL) {
		if cookie.Name == refreshCookieName {
			return cookie
		}
	}
	return nil
}

func refreshCookieFromResponse(resp *http.Response) *http.Cookie {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == refreshCookieName {
			return cookie
		}
	}
	return nil
}
