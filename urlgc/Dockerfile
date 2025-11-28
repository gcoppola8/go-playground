# Stage 1: Build the frontend
FROM node:25-alpine AS frontend
WORKDIR /app/web/spa
COPY web/spa/package.json web/spa/package-lock.json* ./
RUN npm install
COPY web/spa/ ./
RUN npm run build

# Stage 2: Build the backend
FROM golang:1.24-alpine AS backend
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=1 GOOS=linux go build -o /server main.go

# Stage 3: Final image
FROM alpine:latest
RUN apk add --no-cache sqlite-libs
WORKDIR /app
RUN mkdir -p /app/data
COPY --from=backend /server .
COPY --from=frontend /app/web/dist ./web/spa/dist
EXPOSE 8080
CMD ["/app/server"]
