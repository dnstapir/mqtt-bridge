FROM fedora:42

COPY mqtt-bridge /usr/bin/
ENTRYPOINT ["mqtt-bridge", "-config-file", "/etc/dnstapir/mqtt-bridge/config.toml"]
