package dpb

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

type Context struct {
	prng    *rand.Rand
	basedir string
	maxsize int64
	idlen   int
}

func getPaste(id string, c *Context) (*bufio.Reader, *os.File, string, error) {
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

func savePaste(data *io.ReadCloser, mimetype string, c *Context) (string, error) {
	var f *os.File
	var id, fp string
	var err error
	buf := make([]byte, (c.idlen+1)/2)
	for {
		if _, err = c.prng.Read(buf); err != nil {
			return "", errors.New("failed reading from prng")
		}
		id = hex.EncodeToString(buf)[:c.idlen]
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
