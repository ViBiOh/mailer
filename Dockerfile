FROM golang:1.11 as builder

ENV APP_NAME mailer
ENV WORKDIR ${GOPATH}/src/github.com/ViBiOh/mailer

WORKDIR ${WORKDIR}
COPY ./ ${WORKDIR}/

RUN make ${APP_NAME} \
 && mkdir -p /app \
 && curl -s -o /app/cacert.pem https://curl.haxx.se/ca/cacert.pem \
 && cp bin/${APP_NAME} /app/

FROM scratch

ENV APP_NAME mailer
HEALTHCHECK --retries=10 CMD [ "/mailer", "-url", "https://localhost:1080/health" ]

ENTRYPOINT [ "/mailer" ]
EXPOSE 1080

COPY templates/ /templates
COPY --from=builder /app/cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/${APP_NAME} /mailer
