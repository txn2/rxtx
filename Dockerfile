FROM alpine:3

RUN apk update \
    && apk add --no-cache curl ca-certificates \
    && rm -rf /var/cache/apk/*

COPY rxtx /
WORKDIR /
ENTRYPOINT ["/rxtx"]