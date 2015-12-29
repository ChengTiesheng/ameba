package utils

import (
	"archive/tar"
	"fmt"
	"io"

	"github.com/appc/spec/pkg/acirenderer"
)

func Quote(l []string) []string {
	var quoted []string

	for _, s := range l {
		quoted = append(quoted, fmt.Sprintf("%q", s))
	}

	return quoted
}

func ReverseImages(s acirenderer.Images) acirenderer.Images {
	var o acirenderer.Images
	for i := len(s) - 1; i >= 0; i-- {
		o = append(o, s[i])
	}

	return o
}

func In(list []string, el string) bool {
	return IndexOf(list, el) != -1
}

func IndexOf(list []string, el string) int {
	for i, x := range list {
		if el == x {
			return i
		}
	}
	return -1
}

// TarFile is a representation of a file in a tarball. It consists of two parts,
// the Header and the Stream. The Header is a regular tar header, the Stream
// is a byte stream that can be used to read the file's contents
type TarFile struct {
	Header    *tar.Header
	TarStream io.Reader
}

// Name returns the name of the file as reported by the header
func (t *TarFile) Name() string {
	return t.Header.Name
}

// Linkname returns the Linkname of the file as reported by the header
func (t *TarFile) Linkname() string {
	return t.Header.Linkname
}

// WalkFunc is a func for handling each file (header and byte stream) in a tarball
type WalkFunc func(t *TarFile) error

// Walk walks through the files in the tarball represented by tarstream and
// passes each of them to the WalkFunc provided as an argument
func Walk(tarReader tar.Reader, walkFunc func(t *TarFile) error) error {
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			return fmt.Errorf("Error reading tar entry: %v", err)
		}
		if err := walkFunc(&TarFile{Header: hdr, TarStream: &tarReader}); err != nil {
			return err
		}
	}
	return nil
}
