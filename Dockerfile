# Stage 1: Build the React dashboard
FROM node:22-alpine AS frontend
WORKDIR /app/dashboard
COPY dashboard/package*.json ./
RUN npm ci
COPY dashboard/ .
RUN npm run build

# Stage 2: Build the Go binary
FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Embed the built frontend into the binary directory  
COPY --from=frontend /app/dashboard/dist ./dashboard/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o sovereign .

# Stage 3: Final minimal image
FROM alpine:3.20
RUN apk add --no-cache \
    ca-certificates \
    docker-cli \
    docker-cli-compose \
    curl \
    restic

COPY --from=backend /app/sovereign /usr/local/bin/sovereign

# Default config directory
RUN mkdir -p /root/.sovereign

EXPOSE 8080
ENTRYPOINT ["sovereign"]
CMD ["dashboard", "--port", "8080"]
