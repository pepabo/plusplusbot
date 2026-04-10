FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o plusplusbot .

FROM gcr.io/distroless/static-debian12

LABEL org.opencontainers.image.source=https://github.com/pepabo/plusplusbot

WORKDIR /app

COPY --from=builder /app/plusplusbot .

CMD ["./plusplusbot"]
