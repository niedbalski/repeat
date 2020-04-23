FROM quay.io/prometheus/busybox:latest

ARG OS=linux
ARG ARCH=amd64

LABEL maintainer="Jorge Niedbalski <jnr@metaklass.org>"

COPY .build/$OS-$ARCH/repeat /

ENTRYPOINT ["/repeat"]