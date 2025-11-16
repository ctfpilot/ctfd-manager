FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ ./
RUN go build -o /app/app

FROM alpine:3

# Handle Version
ARG VERSION="Development"
ENV VERSION=${VERSION}

COPY --from=builder /app/app /app/app

RUN chmod +x /app/app

CMD ["/app/app"]
