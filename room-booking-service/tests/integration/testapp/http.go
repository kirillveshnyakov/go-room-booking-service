package testapp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type Request struct {
	Method      string
	Path        string
	Body        any
	AccessToken string
}

func (a *TestApp) Do(t *testing.T, r Request) *http.Response {
	t.Helper()

	var body io.Reader

	if r.Body != nil {
		data, err := json.Marshal(r.Body)
		require.NoError(t, err)

		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(
		r.Method,
		a.Server.URL+r.Path,
		body,
	)
	require.NoError(t, err)

	if r.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if r.AccessToken != "" {
		req.Header.Set(
			"Authorization",
			"Bearer "+r.AccessToken,
		)
	}

	resp, err := a.Client.Do(req)
	require.NoError(t, err)

	return resp
}
