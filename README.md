# Why the name Fetch?
I don't know, maybe because it is a Pet Project? (Pun Intended!)

# Fetch
**Fetch** is a SABnzbd and qBittorrent mock API for automation tools like **Sonarr**, **Radarr**. It allows you to manage downloads, track queue and history, and integrate seamlessly with media automation tools.  

## Features

- Acts as SABnzbd and qBittorrent API for Sonarr, Radarr, etc.
- Handles download queue, history, and status.
- Dockerized for easy deployment.

## Supported Providers
I started with TB Usenet as I wanted to explore it.
- Torbox - [Referral Link](https://torbox.app/subscription?referral=b664682d-45d8-4320-9d4a-4271972d4abf)

## Planned Providers
I only have access to the below right now. So I might just add the integration while I have the subscription active.
- Real-Debrid
- Offcloud

## Supported Features
- SABnzbd and qBittorrent Mock APIs
- Multiple Downloader Options
    - Internal 
    - Strm Files
    - Symlinks
    - Do Not Download

## Planned Features
I plan to add the following features for now, any other feature requests are welcome,
- Better Validations (Right now it trusts you that the API keys are genuine, and it will work.)
- Retry Buttons to retry failed downloads/items.
- Filter Files based on Extensions? (I usually download zipped folders so based on demand)

Please raise a Feature Request if there is something you would like to get added.

## Docker Compose Setup

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  fetch:
    image: ghcr.io/pradumnk-mahanta/fetch:latest
    container_name: fetch
    user: 1000:1000 #If you get permission issues on the files created
    environment:
      - TZ=Asia/Kolkata
      - PUID=1000
      - PGID=1000
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

When configuring *Arr Stack for both clients.

```
Example:

http://<host>:9090/
```
This ensures proper routing to the correct mock API endpoints. Use the `username` and `password` set during initial setup of fetch.

You can also target specific downlader while adding. This will override the default downloader,
```
Example:

http://<host>:9090/intr - Internal Downloader
http://<host>:9090/syml - Symlink Downloader
http://<host>:9090/strm - STRM Downloader
http://<host>:9090/dndl - Do Not Download
```
Useful to use the same instance to download using different type.

## Disclaimer
This is just a downloader. it does not host or share any files.

## Credits
To the services, applications and community which drives these kind of projects.


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

See `go.mod` for full list of other packages.
