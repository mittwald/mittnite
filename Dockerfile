FROM        alpine:3.9
COPY        mittnite /usr/bin/mittnite
EXPOSE      9102
ENTRYPOINT  ["mittnite"]
CMD         ["--config-dir", "/etc/mittnite.d"]