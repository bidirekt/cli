# Only GoReleaser builds this image: its build context holds the prebuilt binary at $TARGETPLATFORM/bidirekt.
FROM alpine:3.24.3
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/bidirekt /usr/local/bin/bidirekt
ENTRYPOINT ["bidirekt"]
