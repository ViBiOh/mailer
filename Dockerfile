FROM scratch

HEALTHCHECK --retries=10 CMD https://localhost:1080/health

ENTRYPOINT [ "/bin/sh" ]
EXPOSE 1080

COPY cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY bin/mailer /bin/sh
