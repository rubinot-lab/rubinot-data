FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/rubinot-data ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache \
    chromium \
    nss \
    ca-certificates \
    tzdata \
    ttf-freefont
ENV CHROME_BIN=/usr/bin/chromium-browser
WORKDIR /
COPY --from=build /out/rubinot-data /rubinot-data
EXPOSE 8080
ENTRYPOINT ["/rubinot-data"]
