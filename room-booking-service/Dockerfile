FROM golang:1.26.4-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app

COPY --from=builder /out/app /usr/local/bin/app
COPY --from=builder /out/migrate /usr/local/bin/migrate

USER app

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/app"]
