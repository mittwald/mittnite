FROM        alpine:3.12
COPY        mittnite /usr/bin/mittnite
EXPOSE      9102
ENTRYPOINT  ["/usr/bin/mittnite"]
CMD         ["up","--config-dir", "/etc/mittnite.d"]
