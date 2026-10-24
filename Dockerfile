# syntax=docker/dockerfile:1

FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/extrabadges ./cmd/extrabadges
# Pre-create the data dir so a fresh named volume inherits nonroot ownership.
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/extrabadges /extrabadges
COPY --from=build --chown=nonroot:nonroot /out/data /data
ENV EXTRABADGES_ADDR=:8080 \
    EXTRABADGES_DB=/data/extrabadges.db
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/extrabadges"]
