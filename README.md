# Fetch
**Fetch** is a unified download management service designed to act as a SABnzbd and qBittorrent mock API for automation tools like **Sonarr**, **Radarr**. It allows you to manage downloads, track queue and history, and integrate seamlessly with media automation tools.  

## Features

- Acts as SABnzbd and qBittorrent API for Sonarr, Radarr, etc.
- Handles download queue, history, and status.
- Dockerized for easy deployment.

## Supported Providers
- Torbox

## Docker Compose Setup

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  fetch:
    image: pradumnk-mahanta/fetch:latest
    container_name: fetch
    environment:
      - TZ=Asia/Kolkata
      - PUID=0
      - PGID=0
    volumes:
      - data_vol:/data
      - downloads_vol:/downloads:rw
    logging:
       driver: json-file
       options:
          max-size: 10m
    ports:
      - 9090:9090
    restart: unless-stopped

volumes:
  data_vol:
  downloads_vol:
```

## Usage with *Arr Stack

When configuring Radarr, Sonarr, Lidarr, or Readarr, you must specify a URL base path depending on the client type:

SABnzbd client: /sabnzbd

qBittorrent client: /qbittorrent

```
Example:

http://<host>:9090/sabnzbd
http://<host>:9090/qbittorrent
```

This ensures proper routing to the correct mock API endpoints.

## Credits

Fetch uses and integrates the following notable Go libraries:

- **[github.com/anacrolix/torrent](https://github.com/anacrolix/torrent)**  
  Torrent file parsing and info extraction

- **[gorm.io/gorm](https://gorm.io/)**  
  ORM for SQLite and database management

- **[gorm.io/driver/sqlite](https://gorm.io/docs/connecting_to_the_database.html)**  
  SQLite driver for GORM

- **[github.com/forest6511/gdl](https://github.com/forest6511/gdl)**  
  Download manager library for Go

- **[go.uber.org/zap](https://github.com/uber-go/zap)**  
  Fast structured logging

- **[github.com/mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)**  
  SQLite driver for Go

- **[github.com/google/uuid](https://github.com/google/uuid)**  
  UUID generation for unique download IDs

- **[github.com/dustin/go-humanize](https://github.com/dustin/go-humanize)**  
  Human readable file sizes and durations

See `go.mod` for full list of packages.
