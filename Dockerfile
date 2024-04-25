FROM gcr.io/distroless/static-debian12:nonroot

COPY warden-exporter /usr/bin/warden-exporter

# metrics server
EXPOSE 8008

ENTRYPOINT [ "/usr/bin/warden-exporter" ]

CMD [ "--help" ]
