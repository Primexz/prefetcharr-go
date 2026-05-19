FROM alpine
COPY prefetcharr-go /usr/bin/prefetcharr-go
ENTRYPOINT ["/usr/bin/prefetcharr-go"]
CMD ["-config", "/config/config.yaml"]
