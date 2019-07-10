# dpb

dumb (as in simple) pastebin for the solo self-hoster

**features**

- it's simple


**anti-features**

- sunsetting or one-time read pastes



## server

todo setup


## client upload example

    curl -X POST \
        -H "Content-Type: application/octet-stream" --data-binary '@-' \
        http://localhost:9999/
        < file

the server will guess the mimetype of the uploaded data, and map it to a file extension.
