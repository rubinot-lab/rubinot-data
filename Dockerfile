FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/rubinot-data ./cmd/server

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build /out/rubinot-data /rubinot-data
COPY --from=build /app/assets /assets
EXPOSE 8080
ENTRYPOINT ["/rubinot-data"]
