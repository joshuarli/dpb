#!/usr/bin/env sh

die () { 2>&1 printf %s\\n "$1"; exit 1; }

command -v curl 2>&1 > /dev/null || die '`curl` is required'
command -v file 2>&1 > /dev/null || die '`file` is required'

# fill this out manually
DPB_SERVER="http://localhost:9999"

# pipe to client stdin: upload
# pass a paste id as arg 1: download (to stdout)

if [ -z "$1" ]; then
    tmp="$(mktemp)"
    trap "rm ${tmp}" INT TERM
    cat -> "$tmp"
    mimetype="$(file --mime-type -b "$tmp")"
    curl -X POST \
        -H "Content-Type: ${mimetype}" \
        --data-binary '@-' \
        "${DPB_SERVER}/" < "$tmp"
else
    curl -s -X GET "${DPB_SERVER}/${1}"
fi
