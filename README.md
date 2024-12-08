# qbittorrent-port-forward-gluetun-server

A go script and Docker container for automatically setting qBittorrent's listening port from Gluetun's control server.
The script is run every 30 seconds and checks if the port needs to be updated.

The original script was written in shell script by [@mjmeli](https://github.com/mjmeli/qbittorrent-port-forward-gluetun-server) and I rewrote it in Go so that we can get better logging and error handling.

## Config

### Environment Variables

| Variable     | Example                     | Default                      | Description                                                     |
|--------------|-----------------------------|------------------------------|-----------------------------------------------------------------|
| QBT_USERNAME | `username`                  | `admin`                      | qBittorrent username                                            |
| QBT_PASSWORD | `password`                  | `adminadmin`                 | qBittorrent password                                            |
| QBT_ADDR     | `http://192.168.1.100:8080` | `http://localhost:8080`      | HTTP URL for the qBittorrent web UI, with port                  |
| GTN_ADDR     | `http://192.168.1.100:8000` | `http://localhost:8000`      | HTTP URL for the gluetun control server, with port              |

## Example

### Docker-Compose

The following is an example docker-compose:

```yaml
  qbittorrent-port-forward-gluetun-server:
    image: kirari04/qbittorrent-port-forward-gluetun-server:latest
    container_name: qbittorrent-port-forward-gluetun-server
    restart: unless-stopped
    environment:
      - QBT_USERNAME=username
      - QBT_PASSWORD=password
      - QBT_ADDR=http://192.168.1.100:8080
      - GTN_ADDR=http://192.168.1.100:8000
```

## Development

### Build Image

`docker build . -t kirari04/qbittorrent-port-forward-gluetun-server:latest`

### Run Container

`docker run --rm -it -e QBT_USERNAME=admin -e QBT_PASSWORD=adminadmin -e QBT_ADDR=http://192.168.1.100:8080 -e GTN_ADDR=http://192.168.1.100:8000 kirari04/qbittorrent-port-forward-gluetun-server:latest`
