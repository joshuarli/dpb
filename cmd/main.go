package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/joshuarli/dpb"
)

const (
	mib              = 1 << 20
	defaultNmibLimit = 10
)

func (c *context) handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := r.URL.Path[1:]
		if id == "" {
			fmt.Fprintf(w, "dpb ver. %s", VERSION)
			return
		}
		w.Header()["Date"] = nil // suppress go generated Date header
		reader, f, mimetype, err := getPaste(id, c)
		defer f.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", mimetype)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		io.Copy(w, reader)
	case http.MethodPost:
		mimetype := r.Header.Get("Content-Type")
		if mimetype == "" {
			mimetype = "application/octet-stream"
		}
		data := http.MaxBytesReader(w, r.Body, c.maxsize)
		id, err := savePaste(&data, mimetype, c)
		if err != nil {
			http.Error(w, "failed saving paste ("+err.Error()+")", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s", id)
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
		die(`dpb ver. %s

usage: %s port

environment:

	DPB_DIR        base directory to store paste files
	DPB_MAX_MIB    per-paste upload limit, in MiB
`, VERSION, os.Args[0])
	}

	var nmibLimit int
	DPB_MAX_MIB, exists := os.LookupEnv("DPB_MAX_MIB")
	if !exists {
		nmibLimit = defaultNmibLimit
	} else {
		var err error
		nmibLimit, err = strconv.Atoi(DPB_MAX_MIB)
		if err != nil || nmibLimit < 1 {
			die("DPB_MAX_MIB must be an integer >= 1")
		}
	}

	DPB_DIR, exists := os.LookupEnv("DPB_DIR")
	if !exists {
		die("please set the value of DPB_DIR")
	}
	f, err := os.Open(DPB_DIR)
	if err != nil {
		die(err.Error())
	}
	if fi, err := f.Stat(); err != nil || !fi.IsDir() {
		die("%s does not exist or is not a directory", DPB_DIR)
	}

	c := &dpb.Context{
		prng:    rand.New(rand.NewSource(time.Now().UnixNano())),
		basedir: DPB_DIR,
		maxsize: int64(nmibLimit * mib),
		idlen:   5,
	}
	http.HandleFunc("/", c.handler)
	log.Fatal(http.ListenAndServe(":"+os.Args[1], nil))
}
