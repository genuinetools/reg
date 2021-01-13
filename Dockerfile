FROM alpine:3.12 as builder

RUN	apk add --no-cache \
	ca-certificates

FROM scratch

# Allow ssl comms
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# So we can set the user
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# This should need no privileges by default
USER nobody:nogroup
COPY reg /reg

ENTRYPOINT [ "/reg" ]
CMD [ "--help" ]
