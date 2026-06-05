# ---- Build stage -----------------------------------------------------------
FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod ./

RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

# ---- Runtime stage ---------------------------------------------------------
FROM alpine:3.20

WORKDIR /app

COPY --from=build /out/server /app/server
COPY web /app/web

ENV PORT=8080
EXPOSE 8080

RUN adduser -D -u 10001 appuser
USER appuser

ENTRYPOINT ["/app/server"]
