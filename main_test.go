package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetListenPortUsesAPIKey(t *testing.T) {
	const apiKey = "qbt_test_api_key"
	const listenPort = 51413

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v2/app/preferences", r.URL.Path)
		assert.Equal(t, "Bearer "+apiKey, r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(fmt.Sprintf(`{"listen_port": %d}`, listenPort)))
		require.NoError(t, err)
	}))
	defer server.Close()

	port, err := getQbitListenPort(server.Client(), server.URL, apiKey)
	assert.NoError(t, err)
	assert.Equal(t, listenPort, port)
}

func TestUpdateListenPortUsesAPIKey(t *testing.T) {
	const apiKey = "qbt_test_api_key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v2/app/setPreferences", r.URL.Path)
		assert.Equal(t, "Bearer "+apiKey, r.Header.Get("Authorization"))
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, `{"listen_port": 51413}`, r.PostForm.Get("json"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	assert.NoError(t, updateQbitListenPort(server.Client(), server.URL, apiKey, 51413))
}

func TestGetForwardedPortUsesAPIKey(t *testing.T) {
	const apiKey = "gluetun_test_api_key"
	const expectedPort = 1337

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/portforward", r.URL.Path)
		assert.Equal(t, apiKey, r.Header.Get(GluetunApiKeyHeader))
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(fmt.Sprintf(`{"port": %d}`, expectedPort)))
		require.NoError(t, err)
	}))
	defer server.Close()

	port, err := getGluetunForwardedPort(server.Client(), server.URL, apiKey)
	assert.NoError(t, err)
	assert.Equal(t, expectedPort, port)
}

func TestGetListenPortReturnsStatusError(t *testing.T) {
	const badAPIKey = "bad api key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, err := w.Write([]byte(badAPIKey))
		require.NoError(t, err)
	}))
	defer server.Close()

	_, err := getQbitListenPort(server.Client(), server.URL, "qbt_test_api_key")
	assert.Errorf(t, err, "getListenPort returned nil error, want status error")

	assert.Contains(t, err.Error(), "status code 403")
	assert.Contains(t, err.Error(), badAPIKey)
}
