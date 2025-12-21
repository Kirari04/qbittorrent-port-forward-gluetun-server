# qBittorrent Port Forwarding for Gluetun

<p align="center">
  <img src="https://raw.githubusercontent.com/qdm12/gluetun/refs/heads/master/doc/logo.svg" alt="Gluetun Logo" width="150">
  <br>
  <strong>A robust Go utility to automatically sync Gluetun's forwarded port with qBittorrent.</strong>
</p>

This utility, running in a lightweight Docker container, periodically checks the VPN port forwarded by **Gluetun** and updates **qBittorrent's Listening Port** accordingly. This ensures your torrent client is always configured for optimal performance behind the VPN.

This project is a Go rewrite of the original shell script by [@mjmeli](https://github.com/mjmeli/qbittorrent-port-forward-gluetun-server), offering improved logging, error handling, and stability.

---

## 🔧 Configuration

### Environment Variables

The container is configured using the following environment variables:

| Variable | Description | Default | Required |
| :--- | :--- | :--- | :--- |
| `QBT_USERNAME` | Your qBittorrent WebUI username. | `admin` | No |
| `QBT_PASSWORD` | Your qBittorrent WebUI password. | `adminadmin` | No |
| `QBT_ADDR` | The full HTTP URL for the qBittorrent WebUI. | `http://localhost:8080` | **Yes** |
| `GTN_ADDR` | The full HTTP URL for the Gluetun control server. | `http://localhost:8000` | **Yes** |

---

## 🚀 Example with Docker Compose

This is an example of how to integrate this utility with `gluetun` and `qbittorrent` services in a `docker-compose.yml` file.

### Gluetun Control Server Setup

For this script to read the forwarded port, you must enable Gluetun's HTTP control server and give this utility permission to access the port information.

1.  **Enable Control Server:** You must set the `HTTP_CONTROL_SERVER_ADDRESS` environment variable in your `gluetun` service.
2.  **Create Auth Config:** The control server needs a `config.toml` file to define access rules. Create this file in a directory on your host that you will mount into the container (e.g., `./gluetun-data/auth/config.toml`).

    **`config.toml` content:**
```
[[roles]]
name = "port-forward"
# Allow access to the port forwarding endpoint
routes = ["GET /v1/portforward"]
auth = "none"
```

### Docker-Compose Example

```yaml
version: "3.7"

services:
  gluetun:
    image: qmcgaw/gluetun:latest
    container_name: gluetun
    restart: always
    ports:
      - "8112:8112" # exposing the qbt webui
    cap_add:
      - NET_ADMIN
    devices:
      - /dev/net/tun:/dev/net/tun
    volumes:
      # Mount the directory containing config.toml to /gluetun/auth
      - ./path/to/your/auth/folder:/gluetun/auth
    environment:
      # VPN Configuration (replace with your provider)
      - VPN_SERVICE_PROVIDER=protonvpn
      - OPENVPN_USER=xxxxxxxx+pmp
      - OPENVPN_PASSWORD=xxxxxxxx
      - SERVER_COUNTRIES=Netherlands
      # Port Forwarding
      - VPN_PORT_FORWARDING=on
      - PORT_FORWARD_ONLY=on
      # Enable the control server so the port-forward script can read the port
      - HTTP_CONTROL_SERVER_ADDRESS=:8000
    networks:
      - arr_network

  qbittorrent:
    image: lscr.io/linuxserver/qbittorrent:latest
    restart: always
    # This forces all of qbittorrent's traffic through the gluetun container
    network_mode: service:gluetun
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=Europe/Berlin
      - WEBUI_PORT=8112
    depends_on:
      - gluetun

  port-forward:
    image: kirari04/qbittorrent-port-forward-gluetun-server:latest
    restart: always
    networks:
      - arr_network
    environment:
      # Use the service name and internal port for qBittorrent
      - QBT_ADDR=http://gluetun:8112
      # Use the service name for the Gluetun control server
      - GTN_ADDR=http://gluetun:8000
      # Optional: qBittorrent credentials if not using defaults
      # - QBT_USERNAME=xxxx
      # - QBT_PASSWORD=xxxxxxxxxxxxxx
    depends_on:
      - gluetun
      - qbittorrent

networks:
  arr_network:
    driver: bridge
```

---

## 🛠️ Development

If you wish to build the image yourself.

### Build the Docker Image

```bash
docker build . -t kirari04/qbittorrent-port-forward-gluetun-server:latest
```

### Run the Container Manually

```bash
docker run --rm -it \
  -e QBT_USERNAME=admin \
  -e QBT_PASSWORD=adminadmin \
  -e QBT_ADDR=http://192.168.1.100:8080 \
  -e GTN_ADDR=http://192.168.1.100:8000 \
  kirari04/qbittorrent-port-forward-gluetun-server:latest
```
