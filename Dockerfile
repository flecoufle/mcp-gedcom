ARG GO_VERSION=1.26

# --- dependencies ---
FROM golang:${GO_VERSION}-alpine AS base

RUN apk add --no-cache git bash jq make

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# --- source ---
FROM base AS source

COPY . .

# --- build ---
FROM source AS build

ARG VERSION=dev

RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /out/mcp-gedcom ./cmd/mcp-gedcom/server

# --- unit tests ---
FROM source AS unit

RUN go vet ./... && go test -count=1 ./...

# --- integration tests ---
FROM build AS integration

RUN BIN=/out/mcp-gedcom GEDCOM_FILE=/src/gedcom.ged bash ./test.sh

# --- runtime ---
FROM alpine:latest AS runtime

WORKDIR /app

COPY --from=build /out/mcp-gedcom ./mcp-gedcom
COPY sample/gedcom.ged ./gedcom.ged

ENTRYPOINT ["./mcp-gedcom"]
