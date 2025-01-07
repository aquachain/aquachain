# Build AquaChain in a stock Go builder container
FROM golang:1-alpine AS builder

RUN apk add --no-cache make musl-dev git

ENV CGO_ENABLED=0

COPY . /aquachain
RUN cd /aquachain && make && cd / && \
    mv /aquachain/bin/aquachain* /usr/local/bin/aquachain && \
    rm -rf /aquachain

# Pull AquaChain into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /usr/local/bin/aquachain /usr/local/bin/

EXPOSE 8543 8544 21303/tcp 21303/udp
CMD ["aquachain"]
