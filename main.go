package main

import (
    "encoding/hex"
    "errors"
    "fmt"
    "io"
    "log"
    "math/rand"
    "net/http"
    "os"
    "strings"
    "time"
)

const (
    MiB = 1 << 20
)

type Context struct {
    prng *rand.Rand
    basedir string
}

func read_paste(fn string, w http.ResponseWriter, c *Context) error {
    f, err := os.OpenFile(fn, os.O_RDONLY, 0444)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return nil  // already failed, so early return success
    }
    _, err = io.Copy(w, f)
    if err != nil {
        return errors.New("failed writing response: " + err.Error())
    }
    return nil
}

func save_paste(r *http.Request, c *Context) (string, error) {
    var f *os.File
    var fn string
    var err error
    buf := make([]byte, 2)
    for {
        if _, err = c.prng.Read(buf); err != nil {
            return "", errors.New("failed reading from prng")
        }
        fn = hex.EncodeToString(buf)
        f, err = os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
        if err != nil {
            defer f.Close()
            emsg := err.Error()
            // XXX: generally not great to rely on magic strings, but don't know
            //      of any other way to specifically test that O_EXCL is the failure reason.
            //      Could trade an extra stat call to remove this badness.
            if strings.Contains(emsg, "file exists") {
                continue  // reroll name generation
            }
            return "", errors.New("failed creating file " + fn + " : " + emsg)
        }
        defer os.Chmod(fn, 0444)
        break
    }
    _, err = io.Copy(f, r.Body)  // TODO: limit size of upload, can probably do this before save_paste
    if err != nil {
        return "", errors.New("failed writing to disk: " + err.Error())
    }
    // TODO: fn: mime guess -> ext, check against blacklist, append to fn
    return fn, nil
}

func (c *Context) handler (w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        fn := r.URL.Path[1:]
        err := read_paste(fn, w, c)
        if err != nil {
            http.Error(w, "failed reading paste: " + err.Error(), http.StatusInternalServerError)
        }
    case http.MethodPost:
        fn, err := save_paste(r, c)
        if err != nil {
            http.Error(w, "failed saving paste: " + err.Error(), http.StatusInternalServerError)
        }
        fmt.Fprintf(w, fn)
    default:
        http.Error(w, "only GET /filename or POST / is allowed", http.StatusMethodNotAllowed)
    }
}

func die(format string, v ...interface{}) {
    fmt.Fprintf(os.Stderr, format, v...)
    os.Stderr.Write([]byte("\n"))
    os.Exit(1)
}

func main () {
    // TODO: cli help and port arg, no flag parsing though just default to help if no argv 1
    storageDir, exists := os.LookupEnv("DPB_DIR")
    if ! exists {
        die("please set the value of DPB_DIR")
    }
    f, err := os.Open(storageDir)
    if err != nil {
        die(err.Error())
    }
    if fi, err := f.Stat(); err != nil || ! fi.IsDir() {
        die("%s does not exist or is not a directory", storageDir)
    }

    c := &Context{
        prng: rand.New(rand.NewSource(time.Now().UnixNano())),
        basedir: storageDir,
    }
    http.HandleFunc("/", c.handler)
    log.Fatal(http.ListenAndServe(":8888", nil))
}
