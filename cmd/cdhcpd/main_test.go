package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	log "github.com/Sirupsen/logrus"
)

// func writeConfig(confType, path string, checksum []byte, generator func(io.Writer) error) ([]byte, error) {
func TestWriteConfig(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal("could not allocate temporary file", err)
	}
	if err := f.Close(); err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filename": f.Name(),
		}).Error("failed to close temp file")
	}

	wrapper := func(contents []byte) func(io.Writer) error {
		return func(w io.Writer) error {
			_, err := w.Write(contents)
			return err
		}
	}

	tests := []struct {
		contents []byte
		sum      string
		err      error
		fn       func(io.Writer) error
	}{
		{
			err: fmt.Errorf("test failure"),
			fn: func(w io.Writer) error {
				return errors.New("test failure")
			},
		},
		{
			contents: []byte("testing\n"),
			sum:      "eb1a3227cdc3fedbaec2fe38bf6c044a",
			err:      nil,
		},
		{
			contents: []byte("testing123\n"),
			sum:      "bad9425ff652b1bd52b49720abecf0ba",
			err:      nil,
		},
	}

	for _, test := range tests {
		fn := test.fn
		if fn == nil {
			fn = wrapper(test.contents)
		}

		hash, err := writeConfig("test", f.Name(), test.contents, fn)
		if !reflect.DeepEqual(err, test.err) {
			t.Fatalf("unexpected error, want:|%s|, got:|%s|\n", test.err, err)
		}
		if test.err != nil {
			continue
		}

		sum := fmt.Sprintf("%x", hash)
		if !reflect.DeepEqual(sum, test.sum) {
			t.Fatal("checksum mismatch, want:", test.sum, "got:", sum)
		}
		b, err := ioutil.ReadFile(f.Name())
		if err != nil {
			t.Fatal("failed to read file for verification", err)
		}
		if !reflect.DeepEqual(b, test.contents) {
			t.Fatalf("incorrect contents, want: %s, got: %s", test.contents, b)
		}

		hash, err = writeConfig("test", f.Name(), hash, fn)
		if err != nil {
			t.Fatal("got an unexpected error", err)
		}

		if hash != nil {
			t.Fatal("wanted a nil checksum to signify no changes, got:", string(hash))
		}

		if err := os.Remove(f.Name()); err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"filename": f.Name(),
			}).Error("failed to remove temp file")
		}
	}
}
