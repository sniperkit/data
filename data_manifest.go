package data

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"github.com/jbenet/commander"
	"os"
	"path/filepath"
	"strings"
)

const DataManifest = ".data/manifest.yml"
const noHash = "h"

var cmd_data_manifest = &commander.Command{
	UsageLine: "manifest [ add | remove | hash <path>]",
	Short:     "Generate dataset manifest.",
	Long: `data manifest - Generate dataset manifest.

    Generates and manipulates this dataset's manifest. The manifest
    is a mapping of { <path>: <checksum>}, and describes all files
    that compose a dataset. This mapping is generated by adding and
    hashing (checksum) files.

    Running data-manifest without arguments will generate (or patch)
    the manifest and store it in the dataset's repository. Note that
    already hashed files will not be re-hashed unless forced to. Some
    files may be massive, and hashing every run would be expensive.

    Commands:

      add <file>      Adds <file> to manifest (does not hash).
      rm <file>       Removes <file> from manifest.
      hash <file>     Hashes <file> and adds checksum to manifest.


    Loosely, data-manifest's process is:

    - List all files in the working directory.
    - Add files to the manifest (effectively tracking them).
    - Hash tracked files, adding checksums to the manifest.
  `,
	Run: manifestCmd,
	// Subcommands: []*commander.Command{}
}

func manifestCmd(c *commander.Command, args []string) error {
	mf := NewManifest("")
	return mf.Generate()
}

type Manifest struct {
	file  "-"
	Files *map[string]string ""
}

func NewManifest(path string) *Manifest {
	if len(path) < 1 {
		path = DataManifest
	}

	mf := &Manifest{file: file{Path: path}}

	// initialize map
	mf.Files = &map[string]string{}
	mf.file.format = mf.Files

	// attempt to load
	mf.ReadFile()
	return mf
}

func NewGeneratedManifest(path string) (*Manifest, error) {
	mf := NewManifest(path)

	err := mf.Clear()
	if err != nil {
		return nil, err
	}

	err = mf.Generate()
	if err != nil {
		return nil, err
	}

	return mf, nil
}

func (mf *Manifest) Generate() error {
	pOut("Generating manifest...\n")

	// add new files to manifest file
	// (for now add everything. `data manifest {add,rm}` in future)
	for _, f := range listAllFiles(".") {
		mf.Add(f)
	}

	// warn about manifest-listed files missing from directory
	// (basically, missing things. User removes individually, or `rm --missing`)

	// Once all files are listed, hash all the files, storing the hashes.
	for f, h := range *mf.Files {
		if h != noHash {
			continue
		}

		err := mf.Hash(f)
		if err != nil {
			return err
		}
	}

	return nil

}

func (mf *Manifest) Clear() error {
	for f, _ := range *mf.Files {
		delete(*mf.Files, f)
	}
	return mf.WriteFile()
}

func (mf *Manifest) Add(path string) {
	// check, dont override (could have hash value)
	_, exists := (*mf.Files)[path]
	if !exists {
		(*mf.Files)[path] = noHash
		pOut("data manifest: added %s\n", path)
	}
}

func (mf *Manifest) Hash(path string) error {
	h, err := hashFile(path)
	if err != nil {
		return err
	}

	(*mf.Files)[path] = h

	// Write out file (store incrementally)
	err = mf.WriteFile()
	if err != nil {
		return err
	}

	pOut("data manifest: hashed %.7s %s\n", h, path)
	return nil
}

func (mf *Manifest) StoredPath(hash string) (string, error) {
	for path, h := range *mf.Files {
		if h == hash {
			return path, nil
		}
	}
	return "", fmt.Errorf("Hash %v is not tracked in the manifest.", hash)
}

func (mf *Manifest) StoredHash(path string) (string, error) {
	hash, exists := (*mf.Files)[path]
	if exists {
		return hash, nil
	}
	return "", fmt.Errorf("Path %v is not tracked in the manifest.", path)
}

func (mf *Manifest) Pair(pathOrHash string) (hash string, path string, err error) {
	if isHash(pathOrHash) {
		hash = pathOrHash
		path, err = mf.StoredPath(hash)
	} else {
		path = pathOrHash
		hash, err = mf.StoredHash(path)
	}
	return
}

func listAllFiles(path string) []string {

	files := []string{}
	walkFn := func(path string, info os.FileInfo, err error) error {

		if info.IsDir() {

			// entirely skip hidden dirs
			if len(info.Name()) > 1 && strings.HasPrefix(info.Name(), ".") {
				dOut("data manifest: skipping %s/\n", info.Name())
				return filepath.SkipDir
			}

			// skip datasets/
			if path == DatasetDir {
				dOut("data manifest: skipping %s/\n", info.Name())
				return filepath.SkipDir
			}

			// dont store dirs
			return nil
		}

		// skip manifest file
		if path == DataManifest {
			dOut("data manifest: skipping %s\n", info.Name())
			return nil
		}

		files = append(files, path)
		return nil
	}

	filepath.Walk(path, walkFn)
	return files
}

func hashFile(path string) (string, error) {

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	bf := bufio.NewReader(f)
	h := sha1.New()
	_, err = bf.WriteTo(h)
	if err != nil {
		return "", err
	}

	hex := fmt.Sprintf("%x", h.Sum(nil))
	return hex, nil
}
