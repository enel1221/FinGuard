FROM golang:1.23-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /bin/finguard ./cmd/finguard

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tini
COPY --from=builder /bin/finguard /usr/local/bin/finguard
COPY configs/ /etc/finguard/configs/

RUN addgroup -S finguard && adduser -S -G finguard finguard
USER finguard

EXPOSE 8080
ENTRYPOINT ["tini", "--"]
CMD ["finguard"]
