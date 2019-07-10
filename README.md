# dpb

dumb (as in simple) pastebin for the solo self-hoster

**features**

- it's simple


**anti-features**

- sunsetting or one-time read pastes

**possible future features**

- http DELETE


## download

cross-platform static builds are available [every version release](https://github.com/joshuarli/dpb/releases).


## server example

    $ DPB_DIR="$PWD" dpb 9999


## client upload example

    $ curl -X POST \
        -H "Content-Type: application/octet-stream" --data-binary '@-' \
        http://localhost:9999/
        < file


## client download

to keep things simple, the server doesn't try and infer the extension of the uploaded data blob.

modern web browsers can do some naive detection for things like text, images, and documents, and will add that information to the download dialog. but if you're using something like curl/wget, use `file` to inspect what was downloaded.

there _might_ be a feature in the future where you can pass a custom `Content-Type` to be reflected in the download headers. this would require server changes such as `X-Content-Type-Options=nosniff`, and the client to wrap bsd `file` - i'd like to avoid parsing multipart forms entirely because i've already tried this and it was unwieldy.
