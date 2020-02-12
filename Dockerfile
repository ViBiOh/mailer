FROM alpine as fetcher

WORKDIR /app

RUN apk --update add curl \
 && curl -q -sSL --max-time 10 -o /app/cacert.pem https://curl.haxx.se/ca/cacert.pem

FROM scratch

EXPOSE 1080

HEALTHCHECK --retries=10 CMD [ "/mailer", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/mailer" ]

ARG APP_VERSION
ENV VERSION=${APP_VERSION}

ARG OS
ARG ARCH

ENV MAILER_SWAGGER_TITLE=Mailer

COPY templates/ /templates
COPY --from=fetcher /app/cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY release/mailer_${OS}_${ARCH} /mailer
