package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func main() {
	// Environment variables with default values
	qbtUsername := os.Getenv("QBT_USERNAME")
	if qbtUsername == "" {
		qbtUsername = "admin"
	}
	qbtPassword := os.Getenv("QBT_PASSWORD")
	if qbtPassword == "" {
		qbtPassword = "adminadmin"
	}
	qbtAddr := os.Getenv("QBT_ADDR")
	if qbtAddr == "" {
		qbtAddr = "http://localhost:8080"
	}
	gtnAddr := os.Getenv("GTN_ADDR")
	if gtnAddr == "" {
		gtnAddr = "http://localhost:8000"
	}

	// Create a cookie jar to store cookies
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
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
		portNumber, err := getForwardedPort(client, gtnAddr)
		if err != nil {
			fmt.Println("Could not get current forwarded port from gluetun:", err)
			continue // Continue to the next iteration
		}
		if portNumber == 0 {
			fmt.Println("Got invalid forwarded port, skipping...")
			continue // Continue to the next iteration
		}
		fmt.Println("Forwarded port:", portNumber)

		// Login to qBittorrent
		fmt.Println("Logging in to qBittorrent")
		err = loginToQbittorrent(client, qbtAddr, qbtUsername, qbtPassword)
		if err != nil {
			fmt.Println("Could not login to qBittorrent:", err)
			continue // Continue to the next iteration
		}
		fmt.Println("Logged in to qBittorrent")

		// Get the current listen port from qBittorrent
		fmt.Println("Getting current listen port from qBittorrent")
		listenPort, err := getListenPort(client, qbtAddr)
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
		err = updateListenPort(client, qbtAddr, portNumber)
		if err != nil {
			fmt.Println("Could not update listen port:", err)
			continue // Continue to the next iteration
		}

		fmt.Println("Successfully updated port")
	}
}

func getForwardedPort(client *http.Client, gtnAddr string) (int, error) {
	resp, err := client.Get(gtnAddr + "/v1/openvpn/portforwarded")
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

func loginToQbittorrent(client *http.Client, qbtAddr, username, password string) error {
	resp, err := client.PostForm(qbtAddr+"/api/v2/auth/login", url.Values{
		"username": {username},
		"password": {password},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status code %d\n %s", resp.StatusCode, string(body))
	}

	return nil
}

func getListenPort(client *http.Client, qbtAddr string) (int, error) {
	resp, err := client.Get(qbtAddr + "/api/v2/app/preferences")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	portStr := gjson.GetBytes(body, "listen_port").String()
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, err
	}

	return port, nil
}

func updateListenPort(client *http.Client, qbtAddr string, portNumber int) error {
	data := url.Values{}
	data.Set("json", fmt.Sprintf(`{"listen_port": %d}`, portNumber))

	req, err := http.NewRequest(http.MethodPost, qbtAddr+"/api/v2/app/setPreferences", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded") // Correct content type

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
