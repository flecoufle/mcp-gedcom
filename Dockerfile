# --- builder ---
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o mcp-gedcom ./cmd/mcp-gedcom/server

# --- final ---
FROM alpine:latest

RUN apk --no-cache add bash jq

WORKDIR /app

COPY --from=builder /app/mcp-gedcom ./mcp-gedcom
COPY test.sh /app/test.sh
COPY sample/gedcom.ged /app/gedcom.ged

RUN chmod +x /app/test.sh && /app/test.sh

ENTRYPOINT ["./mcp-gedcom"]