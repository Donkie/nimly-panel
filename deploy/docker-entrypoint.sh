#!/bin/sh
# Ensure /data is writable by the app user, then drop privileges.
chown -R nonroot:nonroot /data 2>/dev/null || true
exec su-exec nonroot "$@"
