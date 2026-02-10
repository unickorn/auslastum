FROM golang:1.25-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o exporter

FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=build /app/exporter /exporter

EXPOSE 8080
ENTRYPOINT ["/exporter"]
