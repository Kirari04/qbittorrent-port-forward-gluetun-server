package main

import (
	"fmt"
	"io"
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
	qbtAPIKey string
	qbtAddr   string
	gtnAddr   string
}

func loadConfig() (config, error) {
	qbtAPIKey := strings.TrimSpace(os.Getenv("QBT_API_KEY"))
	if qbtAPIKey == "" {
		return config{}, fmt.Errorf("QBT_API_KEY is required")
	}

	qbtAddr := os.Getenv("QBT_ADDR")
	if qbtAddr == "" {
		qbtAddr = "http://localhost:8080"
	}
	gtnAddr := os.Getenv("GTN_ADDR")
	if gtnAddr == "" {
		gtnAddr = "http://localhost:8000"
	}

	return config{
		qbtAPIKey: qbtAPIKey,
		qbtAddr:   qbtAddr,
		gtnAddr:   gtnAddr,
	}, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("Invalid configuration:", err)
		os.Exit(1)
	}

	client := &http.Client{}
	nth := 0
	// Run the logic every 30 seconds
	for {
		if nth != 0 {
			nth++
			fmt.Println("Sleeping for 30 seconds")
			time.Sleep(30 * time.Second)
		} else {
			nth++
		}

		// Get the forwarded port from gluetun
		fmt.Println("Getting forwarded port from gluetun")
		portNumber, err := getForwardedPort(client, cfg.gtnAddr)
		if err != nil {
			fmt.Println("Could not get current forwarded port from gluetun:", err)
			continue // Continue to the next iteration
		}
		if portNumber == 0 {
			fmt.Println("Got invalid forwarded port, skipping...")
			continue // Continue to the next iteration
		}
		fmt.Println("Forwarded port:", portNumber)

		// Get the current listen port from qBittorrent
		fmt.Println("Getting current listen port from qBittorrent")
		listenPort, err := getListenPort(client, cfg.qbtAddr, cfg.qbtAPIKey)
		if err != nil {
			fmt.Println("Could not get current listen port:", err)
			continue // Continue to the next iteration
		}
		fmt.Println("Current listen port:", listenPort)

		// Check if the port needs to be updated
		if portNumber == listenPort {
			fmt.Println("Port already set, skipping...")
			continue // Continue to the next iteration
		}

		// Update the listen port in qBittorrent
		fmt.Printf("Updating port to %d\n", portNumber)
		err = updateListenPort(client, cfg.qbtAddr, cfg.qbtAPIKey, portNumber)
		if err != nil {
			fmt.Println("Could not update listen port:", err)
			continue // Continue to the next iteration
		}

		fmt.Println("Successfully updated port")
	}
}

func getForwardedPort(client *http.Client, gtnAddr string) (int, error) {
	resp, err := client.Get(gtnAddr + "/v1/portforward")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	fmt.Println("Got response from gluetun:", string(body))

	portStr := gjson.GetBytes(body, "port").String()
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, err
	}

	return port, nil
}

func newQbittorrentRequest(method, qbtAddr, path, apiKey string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, qbtAddr+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	return req, nil
}

func getListenPort(client *http.Client, qbtAddr, apiKey string) (int, error) {
	req, err := newQbittorrentRequest(http.MethodGet, qbtAddr, "/api/v2/app/preferences", apiKey, nil)
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

func updateListenPort(client *http.Client, qbtAddr, apiKey string, portNumber int) error {
	data := url.Values{}
	data.Set("json", fmt.Sprintf(`{"listen_port": %d}`, portNumber))

	req, err := newQbittorrentRequest(http.MethodPost, qbtAddr, "/api/v2/app/setPreferences", apiKey, strings.NewReader(data.Encode()))
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
