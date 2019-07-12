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
)

const (
	mib              = 1 << 20
	defaultNmibLimit = 10
)

type context struct {
	prng    *rand.Rand
	basedir string
	maxsize int64
}

func getPaste(id string, c *context) (*bufio.Reader, *os.File, string, error) {
	f, err := os.OpenFile(path.Join(c.basedir, id), os.O_RDONLY, 0444)
	if err != nil {
		time.Sleep(3 * time.Second) // deter rogue enumeration attempts
		return nil, f, "", errors.New("not found")
	}
	reader := bufio.NewReader(f)
	// XXX: this may go badly if the paste wasn't saved to disk via savePaste, which writes a mimetype to the 1st line
	mimetype, err := reader.ReadString('\n')
	if err != nil {
		return nil, f, "", errors.New("failed to read mimetype prelude in paste ")
	}
	return reader, f, mimetype, nil
}

func savePaste(data *io.ReadCloser, mimetype string, c *context) (string, error) {
	var f *os.File
	var id, fp string
	var err error
	buf := make([]byte, 3)
	for {
		if _, err = c.prng.Read(buf); err != nil {
			return "", errors.New("failed reading from prng")
		}
		id = hex.EncodeToString(buf)[:5]
		fp = path.Join(c.basedir, id)
		f, err = os.OpenFile(fp, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
		defer f.Close()
		if err != nil {
			emsg := err.Error()
			// XXX: generally not great to rely on magic strings, but don't know
			//      of any other way to specifically test that O_EXCL is the failure reason.
			//      Could trade an extra stat call to remove this badness.
			if strings.Contains(emsg, "file exists") {
				continue // reroll name generation
			}
			return "", errors.New("failed creating file " + fp + " : " + emsg)
		}
		break
	}
	// we write the client-provided mimetype to the beginning of the paste file
	// so the server doesn't have to do this (lots of added complexity)
	// golang's net.http.sniff.DetectContentType is not nearly as complete as bsd file
	fmt.Fprintln(f, mimetype)
	_, err = io.Copy(f, *data)
	if err != nil {
		defer os.Remove(fp)
		return "", errors.New("failed writing to disk: " + err.Error())
	}
	return id, nil
}

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

	c := &context{
		prng:    rand.New(rand.NewSource(time.Now().UnixNano())),
		basedir: DPB_DIR,
		maxsize: int64(nmibLimit * mib),
	}
	http.HandleFunc("/", c.handler)
	log.Fatal(http.ListenAndServe(":"+os.Args[1], nil))
}
