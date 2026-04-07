FROM rg.fr-par.scw.cloud/vibioh/scratch

ENV MAILER_PORT 1080
EXPOSE 1080

COPY cacert.pem /etc/ssl/cert.pem

COPY templates/ /templates

HEALTHCHECK --retries=10 CMD [ "/mailer", "-url", "http://127.0.0.1:1080/health" ]
ENTRYPOINT [ "/mailer" ]

ARG VERSION
ENV VERSION=${VERSION}

ARG GIT_SHA
ENV GIT_SHA=${GIT_SHA}

ARG TARGETOS
ARG TARGETARCH

COPY release/mailer_${TARGETOS}_${TARGETARCH} /mailer
