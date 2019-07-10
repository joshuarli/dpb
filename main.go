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
	"strconv"
	"time"
)

const (
	mib = 1 << 20
	defaultNmibLimit = 10
)

type context struct {
	prng    *rand.Rand
	basedir string
	maxsize	int64
}

func readPaste(r *http.Request, w http.ResponseWriter, c *context) error {
	fn := r.URL.Path[1:]
	f, err := os.OpenFile(fn, os.O_RDONLY, 0444)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return nil // already failed, so early return success
	}
	_, err = io.Copy(w, f)
	if err != nil {
		return errors.New("failed writing response: " + err.Error())
	}
	return nil
}

func savePaste(r *http.Request, w http.ResponseWriter, c *context) (string, error) {
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
				continue // reroll name generation
			}
			return "", errors.New("failed creating file " + fn + " : " + emsg)
		}
		defer os.Chmod(fn, 0444)
		break
	}
	_, err = io.Copy(f, http.MaxBytesReader(w, r.Body, c.maxsize))
	if err != nil {
		defer os.Remove(fn)
		return "", errors.New("failed writing to disk: " + err.Error())
	}
	return fn, nil
}

func (c *context) handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		err := readPaste(r, w, c)
		if err != nil {
			http.Error(w, "failed reading paste: " + err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPost:
		fn, err := savePaste(r, w, c)
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

func main() {
	if len(os.Args) != 2 {
		die("usage: %s port", os.Args[0])  // TODO: add env var info
	}

	var nmibLimit int
	ev, exists := os.LookupEnv("DPB_MAX_MIB")
	if !exists {
		nmibLimit = defaultNmibLimit
	} else {
		var err error
		nmibLimit, err = strconv.Atoi(ev)
		if err != nil || nmibLimit < 1 {
			die("DPB_MAX_MIB must be an integer >= 1")
		}
	}

	basedir, exists := os.LookupEnv("DPB_DIR")
	if !exists {
		die("please set the value of DPB_DIR")
	}
	f, err := os.Open(basedir)
	if err != nil {
		die(err.Error())
	}
	if fi, err := f.Stat(); err != nil || !fi.IsDir() {
		die("%s does not exist or is not a directory", basedir)
	}

	c := &context{
		prng:    rand.New(rand.NewSource(time.Now().UnixNano())),
		basedir: basedir,
		maxsize: int64(nmibLimit * mib),
	}
	http.HandleFunc("/", c.handler)
	log.Fatal(http.ListenAndServe(":" + os.Args[1], nil))
}
