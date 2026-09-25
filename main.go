package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	_ "golang.org/x/crypto/x509roots/fallback" // Embeds x509root certificates into the binary
)

type config struct {
	qbitAPIKey    string
	qbitURL       string
	gluetunURL    string
	gluetunAPIKey string
	delayDuration time.Duration
}

var (
	ErrQbitAPIKeyRequired = errors.New("QBT_API_KEY is required")
	ErrInvalidPort        = errors.New("got invalid forwarded port")
	ErrGettingListenPort  = errors.New("could not get current listen port")
	ErrCantUpdatePort     = errors.New("could not update listen port")
	ErrGettingGluetunPort = errors.New("could not get current Gluetun port")
	ErrParsingDuration    = errors.New("could not parse duration")
)

const (
	GluetunApiKeyHeader = "X-API-Key"
	QbitApiKeyHeader    = "Authorization"
)

func loadConfig() (*config, error) {
	qbitAPIKey, exists := os.LookupEnv("QBT_API_KEY")
	if !exists {
		return &config{}, ErrQbitAPIKeyRequired
	}
	gluetunAPIKey, exists := os.LookupEnv("GTN_API_KEY")
	if !exists {
		gluetunAPIKey = ""
	}

	qbtAddr, exists := os.LookupEnv("QBT_ADDR")
	if !exists {
		qbtAddr = "http://localhost:8080"
	}
	gtnAddr, exists := os.LookupEnv("GTN_ADDR")
	if !exists {
		gtnAddr = "http://localhost:8000"
	}

	delayDurationStr, exists := os.LookupEnv("DELAY_DURATION")
	if !exists {
		delayDurationStr = "1m"
	}
	delayDuration, err := time.ParseDuration(delayDurationStr)
	if err != nil {
		return &config{}, fmt.Errorf("%w: failed parsing DELAY_DURATION", ErrParsingDuration)
	}

	return &config{
		qbitAPIKey:    qbitAPIKey,
		qbitURL:       qbtAddr,
		gluetunURL:    gtnAddr,
		gluetunAPIKey: gluetunAPIKey,
		delayDuration: delayDuration,
	}, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("Invalid configuration", slog.Any("error", err))
		os.Exit(1)
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	for {
		if err = setPort(cfg, client); err != nil {
			slog.Error("Error setting port:", slog.Any("error", err))
		}
		time.Sleep(cfg.delayDuration)
	}
}

func setPort(cfg *config, client *http.Client) error {
	// Get the forwarded port from gluetun
	slog.Debug("Getting forwarded port from gluetun")
	newPort, err := getGluetunForwardedPort(client, cfg.gluetunURL, cfg.gluetunAPIKey)
	if err != nil {
		slog.Error("", slog.Any("error", err))
		return fmt.Errorf("%w: %w", ErrGettingGluetunPort, err)
	}
	if newPort == 0 {
		return ErrInvalidPort
	}

	// Get the current listen port from qBittorrent
	oldPort, err := getQbitListenPort(client, cfg.qbitURL, cfg.qbitAPIKey)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGettingListenPort, err)
	}
	slog.Info("Current listen port", slog.Int("port", oldPort))

	// Check if the port needs to be updated
	if newPort == oldPort {
		slog.Info("Port already set, skipping...", slog.Int("port", newPort))
		return nil
	}

	// Update the listen port in qBittorrent
	slog.Info("Updating port", slog.Int("new_port", newPort))
	err = updateQbitListenPort(client, cfg.qbitURL, cfg.qbitAPIKey, newPort)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCantUpdatePort, err)
	}

	slog.Info("Successfully updated port", slog.Int("new_port", newPort))
	return nil
}

func getGluetunForwardedPort(client *http.Client, gluetunURL, gluetunAPIKey string) (int, error) {
	req, err := newGluetunRequest(http.MethodGet, gluetunURL, "/v1/portforward", gluetunAPIKey, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	portStr := gjson.GetBytes(body, "port").String()
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, err
	}

	return port, nil
}

func newQbitRequest(method, baseURL, path, apiKey string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set(QbitApiKeyHeader, "Bearer "+apiKey)

	return req, nil
}

func newGluetunRequest(method, baseURL, path, apiKey string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set(GluetunApiKeyHeader, apiKey)
	}

	return req, nil
}

func getQbitListenPort(client *http.Client, qbtAddr, apiKey string) (int, error) {
	req, err := newQbitRequest(http.MethodGet, qbtAddr, "/api/v2/app/preferences", apiKey, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get listen port with status code %d\n %s", resp.StatusCode, string(body))
	}

	portStr := gjson.GetBytes(body, "listen_port").String()
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, err
	}

	return port, nil
}

func updateQbitListenPort(client *http.Client, qbtAddr, apiKey string, portNumber int) error {
	data := url.Values{}
	data.Set("json", fmt.Sprintf(`{"listen_port": %d}`, portNumber))

	req, err := newQbitRequest(http.MethodPost, qbtAddr, "/api/v2/app/setPreferences", apiKey, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update listen port with status code %d\n %s", resp.StatusCode, string(body))
	}

	return nil
}
