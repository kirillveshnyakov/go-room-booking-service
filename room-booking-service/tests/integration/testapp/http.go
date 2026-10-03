package testapp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type Request struct {
	Method      string
	Path        string
	Body        any
	RawBody     string
	AccessToken string
	Client      *http.Client
}

func (a *TestApp) Do(t *testing.T, r Request) *http.Response {
	t.Helper()

	var body io.Reader

	if r.Body != nil {
		data, err := json.Marshal(r.Body)
		require.NoError(t, err)

		body = bytes.NewReader(data)
	} else if r.RawBody != "" {
		body = strings.NewReader(r.RawBody)
	}

	req, err := http.NewRequest(
		r.Method,
		a.Server.URL+r.Path,
		body,
	)
	require.NoError(t, err)

	if r.Body != nil || r.RawBody != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	if r.AccessToken != "" {
		req.Header.Set(
			"Authorization",
			"Bearer "+r.AccessToken,
		)
	}

	client := a.Client
	if r.Client != nil {
		client = r.Client
	}

	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}
