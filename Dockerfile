FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /vau .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /vau /usr/local/bin/vau
ENV TERM=xterm-256color
ENV COLORTERM=truecolor
ENTRYPOINT ["vau"]
