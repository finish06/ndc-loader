FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ndc-loader ./cmd/ndc-loader

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /ndc-loader /app/ndc-loader
COPY datasets.yaml /app/datasets.yaml
COPY migrations /app/migrations

EXPOSE 8081

ENTRYPOINT ["/app/ndc-loader"]
