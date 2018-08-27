/*
Sniperkit-Bot
- Status: analyzed
*/

package data

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"launchpad.net/goyaml"
)

type SerializedFile struct {
	Path   string      "-"
	Format interface{} "-"
}

func (f *SerializedFile) Marshal() ([]byte, error) {
	dOut("Marshalling %s\n", f.Path)
	return goyaml.Marshal(f.Format)
}

func (f *SerializedFile) Unmarshal(buf []byte) error {
	err := goyaml.Unmarshal(buf, f.Format)
	if err != nil {
		return err
	}

	dOut("Unmarshalling %s\n", f.Path)
	return nil
}

func (f *SerializedFile) Write(w io.Writer) error {
	buf, err := f.Marshal()
	if err != nil {
		return err
	}

	_, err = w.Write(buf)
	return err
}

func (f *SerializedFile) Read(r io.Reader) error {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	return f.Unmarshal(buf)
}

func (f *SerializedFile) WriteFile() error {
	if len(f.Path) < 1 {
		return fmt.Errorf("SerializedFile: No path provided for writing.")
	}

	buf, err := f.Marshal()
	if err != nil {
		return err
	}

	err = os.MkdirAll(path.Dir(f.Path), 0777)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(f.Path, buf, 0666)
}

func (f *SerializedFile) ReadFile() error {
	if len(f.Path) < 1 {
		return fmt.Errorf("SerializedFile: No path provided for reading.")
	}

	buf, err := ioutil.ReadFile(f.Path)
	if err != nil {
		return err
	}

	return f.Unmarshal(buf)
}

func (f *SerializedFile) ReadBlob(ref string) error {
	i, err := NewMainDataIndex()
	if err != nil {
		return err
	}

	r, err := i.BlobStore.Get(BlobKey(ref))
	if err != nil {
		return err
	}

	err = f.Read(r)
	if err != nil {
		return err
	}

	return nil
}

func Marshal(in interface{}) (io.Reader, error) {
	buf, err := goyaml.Marshal(in)
	if err != nil {
		return nil, err
	}

	// pOut("<Marshal>\n")
	// pOut("%s\n", buf)
	// pOut("</Marshal>\n")
	return bytes.NewReader(buf), nil
}

func Unmarshal(in io.Reader, out interface{}) error {
	buf, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	// pOut("<Unmarshal>\n")
	// pOut("%s\n", buf)
	// pOut("</Unmarshal>\n")
	return goyaml.Unmarshal(buf, out)
}

// Userful for converting between representations
func MarshalUnmarshal(in interface{}, out interface{}) error {
	// struct -> yaml -> map for easy access
	rdr, err := Marshal(in)
	if err != nil {
		return err
	}

	return Unmarshal(rdr, out)
}
