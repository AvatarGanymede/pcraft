#!/bin/sh
set -e

if [ "$(id -u)" = '0' ]; then
    mkdir -p /data/home
    chown -R pcraft:pcraft /data
    exec gosu pcraft "$@"
fi

exec "$@"
