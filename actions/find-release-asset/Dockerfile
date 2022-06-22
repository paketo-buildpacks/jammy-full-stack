FROM alpine

RUN apk add \
    bash \
    curl \
    git \
    github-cli \
    jq \
  && rm -rf /var/cache/apk/*

COPY entrypoint /entrypoint
ENTRYPOINT ["/entrypoint"]
