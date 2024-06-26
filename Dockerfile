FROM        alpine:3.20
COPY        mittnite /usr/bin/mittnite
COPY        mittnitectl /usr/bin/mittnitectl
EXPOSE      9102
ENTRYPOINT  ["/usr/bin/mittnite"]
CMD         ["up","--config-dir", "/etc/mittnite.d"]
