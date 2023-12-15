FROM vibioh/scratch

ENV MAILER_PORT 1080
EXPOSE 1080

ENV ZONEINFO /zoneinfo.zip
COPY zoneinfo.zip /zoneinfo.zip
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY templates/ /templates

HEALTHCHECK --retries=10 CMD [ "/mailer", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/mailer" ]

ARG VERSION
ENV VERSION ${VERSION}

ARG GIT_SHA
ENV GIT_SHA ${GIT_SHA}

ARG TARGETOS
ARG TARGETARCH

COPY release/mailer_${TARGETOS}_${TARGETARCH} /mailer
