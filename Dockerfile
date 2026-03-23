FROM alpine:3.21
RUN apk add --no-cache ca-certificates
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/ghlabels /usr/local/bin/ghlabels
ENTRYPOINT ["ghlabels"]
