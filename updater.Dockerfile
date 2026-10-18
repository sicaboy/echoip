FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/geoip-update ./cmd/geoip-update
RUN CGO_ENABLED=0 go build -o /geoip-update ./cmd/geoip-update

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /geoip-update /geoip-update
ENTRYPOINT ["/geoip-update"]
