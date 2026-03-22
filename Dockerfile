FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY ghlabels /usr/local/bin/ghlabels
ENTRYPOINT ["ghlabels"]
