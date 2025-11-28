# GPhotos Docker Setup

This project includes a Docker Compose setup with the following services:

## Services

### 1. GPhotos Application (Go)
- **Port**: 8080
- **Endpoints**:
  - `/` - Main endpoint
  - `/health` - Health check endpoint

### 2. MailHog (Email Testing)
- **SMTP Port**: 1025
- **Web UI Port**: 8025
- **Web UI URL**: http://localhost:8025
- **Description**: Email testing tool that catches all emails sent by your application

### 3. ActiveMQ Artemis (Message Queue)
- **OpenWire Port**: 61616
- **STOMP Port**: 61613
- **Web Console Port**: 8161
- **Web Console URL**: http://localhost:8161/console
- **Credentials**: admin/admin

## Quick Start

1. **Start all services**:
   ```bash
   docker-compose up -d
   ```

2. **View logs**:
   ```bash
   docker-compose logs -f
   ```

3. **Stop all services**:
   ```bash
   docker-compose down
   ```

4. **Rebuild and start**:
   ```bash
   docker-compose up --build -d
   ```

## Service URLs

- **GPhotos App**: http://localhost:8080
- **MailHog Web UI**: http://localhost:8025
- **ActiveMQ Console**: http://localhost:8161/console

## Development

### Building the Go application locally
```bash
go mod tidy
go run main.go
```

### Testing email functionality
Send emails to MailHog SMTP server at `localhost:1025`. All emails will be caught and displayed in the web interface at http://localhost:8025.

### Testing message queues
Connect to ActiveMQ using STOMP protocol at `localhost:61613` with credentials `admin/admin`.

## Configuration

Environment variables can be customized in the `.env` file or directly in the `docker-compose.yml` file.

## Volumes

- `activemq-data`: Persists ActiveMQ data between container restarts

## Network

All services are connected via the `gphotos-network` bridge network, allowing them to communicate using service names as hostnames.