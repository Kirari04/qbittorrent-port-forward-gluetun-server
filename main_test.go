package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetListenPortUsesAPIKey(t *testing.T) {
	const apiKey = "qbt_test_api_key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/api/v2/app/preferences" {
			t.Errorf("path = %s, want /api/v2/app/preferences", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+apiKey {
			t.Errorf("Authorization = %q, want %q", got, "Bearer "+apiKey)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"listen_port": 51413}`))
	}))
	defer server.Close()

	port, err := getListenPort(server.Client(), server.URL, apiKey)
	if err != nil {
		t.Fatalf("getListenPort returned error: %v", err)
	}
	if port != 51413 {
		t.Fatalf("port = %d, want 51413", port)
	}
}

func TestUpdateListenPortUsesAPIKey(t *testing.T) {
	const apiKey = "qbt_test_api_key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/v2/app/setPreferences" {
			t.Errorf("path = %s, want /api/v2/app/setPreferences", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+apiKey {
			t.Errorf("Authorization = %q, want %q", got, "Bearer "+apiKey)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm returned error: %v", err)
		}
		if got := r.PostForm.Get("json"); got != `{"listen_port": 51413}` {
			t.Errorf("json form value = %q, want %q", got, `{"listen_port": 51413}`)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := updateListenPort(server.Client(), server.URL, apiKey, 51413); err != nil {
		t.Fatalf("updateListenPort returned error: %v", err)
	}
}

func TestGetListenPortReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("bad api key"))
	}))
	defer server.Close()

	_, err := getListenPort(server.Client(), server.URL, "qbt_test_api_key")
	if err == nil {
		t.Fatal("getListenPort returned nil error, want status error")
	}
	if !strings.Contains(err.Error(), "status code 403") {
		t.Fatalf("error = %q, want status code", err.Error())
	}
	if !strings.Contains(err.Error(), "bad api key") {
		t.Fatalf("error = %q, want response body", err.Error())
	}
}

func TestMissingAPIKeyFailsConfig(t *testing.T) {
	t.Setenv("QBT_API_KEY", "   ")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("loadConfig returned nil error, want missing API key error")
	}
	if !strings.Contains(err.Error(), "QBT_API_KEY is required") {
		t.Fatalf("error = %q, want QBT_API_KEY requirement", err.Error())
	}
}
