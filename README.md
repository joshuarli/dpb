# dpb

dumb (as in simple) pastebin for the solo self-hoster

**features**

- it's simple

**anti-features**

these are some commonly expected features that are omitted in the interest of keeping the server as simple and straightforward as possible. if you want these, you may want to look elsewhere.

- configuration file
- logging
- sunsetting or one-time read pastes
- syntax detection + highlighting
- diceware-like paste ids
    - 5 digit hex + delayed 404 GET is a simple compromise


## download

cross-platform static builds are available [every version release](https://github.com/joshuarli/dpb/releases).


## server

start the server on port 9999, saving pastes of maximum size 1 MiB to `$PWD`.

    $ DPB_DIR="$PWD" DPB_MAX_MIB=1 dpb 9999


## client

because the server is made to be as simple as possible, it is up to the client to upload binary data and tell the server the mimetype to serve the paste with.

a simple client script `client.sh` has been provided. it uses `file` to detect the mimetype and add it as a `Content-Type` header to the `curl` upload. the server is designed to recognize and remember this header when serving the paste.

    $ ./client.sh < image.jpg
    9ed79
    $ ./client.sh 9ed79 > pasted-image.jpg
