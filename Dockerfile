# --- build web ---
FROM node:20-alpine AS web
WORKDIR /app
COPY web/package*.json web/
WORKDIR /app/web
RUN npm ci
COPY web/ /app/web/
RUN npm run build

# --- build go ---
FROM golang:1.24-alpine AS go
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/sipserver ./cmd/pbx

# --- runtime ---
FROM alpine:3.20
WORKDIR /app
COPY --from=go /bin/sipserver /app/sipserver
COPY --from=web /app/web/dist /app/web/dist
EXPOSE 8080 5060/udp
ENV HTTP_PORT=8080
CMD ["/app/sipserver"]
