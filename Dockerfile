FROM scratch

HEALTHCHECK --retries=10 CMD [ "/mailer", "-url", "https://localhost:1080/health" ]

ENTRYPOINT [ "/mailer" ]
EXPOSE 1080

COPY cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY templates/ /templates
COPY bin/mailer /mailer
