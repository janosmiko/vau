# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 go build -ldflags "\
    -s -w \
    -X github.com/janosmiko/vau/internal/version.Version=${VERSION} \
    -X github.com/janosmiko/vau/internal/version.GitCommit=${GIT_COMMIT} \
    -X github.com/janosmiko/vau/internal/version.BuildDate=${BUILD_DATE}" \
    -o /vau .

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates && \
    adduser -D -h /home/vau vau

COPY --from=builder /vau /usr/local/bin/vau
ENV TERM=xterm-256color
ENV COLORTERM=truecolor

USER vau
ENTRYPOINT ["vau"]
