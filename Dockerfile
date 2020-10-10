FROM vibioh/scratch

ENV MAILER_PORT 1080

EXPOSE 1080

COPY templates/ /templates

HEALTHCHECK --retries=10 CMD [ "/mailer", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/mailer" ]

ARG VERSION
ENV VERSION=${VERSION}

ARG TARGETOS
ARG TARGETARCH

COPY cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY zoneinfo.zip /
COPY release/mailer_${TARGETOS}_${TARGETARCH} /mailer
