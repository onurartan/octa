# BuildApp Stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache build-base git

WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,direct
ENV GODEBUG=netdns=go
ENV CGO_ENABLED=1

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o octa ./cmd/octa
# RUN CGO_ENABLED=1 GOOS=linux go build \
#     -ldflags="-s -w -extldflags '-static'" \
#     -trimpath \
#     -o octa ./cmd/octa

RUN GOOS=linux go build \
    -ldflags="-s -w -linkmode external -extldflags '-static'" \
    -trimpath \
    -o octa ./cmd/octa

# RunApp Stage
FROM alpine:latest


RUN apk add --no-cache ca-certificates tzdata fontconfig ttf-dejavu

# Perms Users
RUN addgroup -S octagroup && adduser -S octauser -G octagroup -u 1000

WORKDIR /app

COPY --from=builder /app/octa .
COPY --from=builder /app/fonts ./fonts

#  Create Data Folder and give prems
RUN mkdir -p data

RUN chown -R octauser:octagroup /app && \
    chmod 700 /app/data

USER octauser

EXPOSE 9980

CMD ["./octa"]