FROM golang:1.12 as builder

ENV APP_NAME mailer

WORKDIR /app
COPY . .

RUN make ${APP_NAME} \
 && curl -s -o /app/cacert.pem https://curl.haxx.se/ca/cacert.pem

FROM scratch

ENV APP_NAME mailer
HEALTHCHECK --retries=10 CMD [ "/mailer", "-url", "http://localhost:1080/health" ]

ENTRYPOINT [ "/mailer" ]
EXPOSE 1080

COPY templates/ /templates
COPY --from=builder /app/cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/bin/${APP_NAME} /mailer
