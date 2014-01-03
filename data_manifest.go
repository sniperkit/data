package data

import (
	"bytes"
	"fmt"
	"github.com/gonuts/flag"
	"github.com/jbenet/commander"
	"os"
	"path/filepath"
	"strings"
)

const DataManifest = "Manifest"
const noHash = "<to be hashed>"

var cmd_data_manifest = &commander.Command{
	UsageLine: "manifest [[ add | remove | hash | check ] <path>]",
	Short:     "Generate and manipulate dataset manifest.",
	Long: `data manifest - Generate and manipulate dataset manifest.

    Generates and manipulates this dataset's manifest. The manifest
    is a mapping of { <path>: <checksum>}, and describes all files
    that compose a dataset. This mapping is generated by adding and
    hashing (checksum) files.

    Running data-manifest without arguments will generate (or patch)
    the manifest. Note that already hashed files will not be re-hashed
    unless forced to. Some files may be massive, and hashing every run
    would be prohibitively expensive.

    Commands:

      add <file>      Adds <file> to manifest (does not hash).
      rm <file>       Removes <file> from manifest.
      hash <file>     Hashes <file> and adds checksum to manifest.
      check <file>    Verifies <file> checksum matches manifest.

    (use the --all flag to do it to all available files)

    Loosely, data-manifest's process is:

    - List all files in the working directory.
    - Add files to the manifest (effectively tracking them).
    - Hash tracked files, adding checksums to the manifest.
  `,
	Run: manifestCmd,
	Subcommands: []*commander.Command{
		cmd_data_manifest_add,
		cmd_data_manifest_rm,
		cmd_data_manifest_hash,
	},
}

var cmd_data_manifest_add = &commander.Command{
	UsageLine: "add <file>",
	Short:     "Adds <file> to manifest (does not hash).",
	Long: `data manifest add - Adds <file> to manifest (does not hash).

    Adding files to the manifest ensures they are tracked. This command
    adds the given <file> to the manifest, saves it, and exits. It does
    not automatically hash the file (run 'data manifest hash').

    See 'data manifest'.

Arguments:

    <file>   path of the file to add.

  `,
	Run:  manifestAddCmd,
	Flag: *flag.NewFlagSet("data-manifest-add", flag.ExitOnError),
}

var cmd_data_manifest_rm = &commander.Command{
	UsageLine: "rm <file>",
	Short:     "Removes <file> from manifest.",
	Long: `data manifest rm - Removes <file> from manifest.

    Removing files from the manifest stops tracking them. This command
    removes the given <file> (and hash) from the manifest, and exits.

    See 'data manifest'.

Arguments:

    <file>   path of the file to remove.

  `,
	Run:  manifestRmCmd,
	Flag: *flag.NewFlagSet("data-manifest-rm", flag.ExitOnError),
}

var cmd_data_manifest_hash = &commander.Command{
	UsageLine: "hash <file>",
	Short:     "Hashes <file> and adds checksum to manifest.",
	Long: `data manifest hash - Hashes <file> and adds checksum to manifest.

		Hashing files in the manifest calculates the file checksums. This command
    hashes the given <file>, adds it to the manifest, and exits.

    See 'data manifest'.

Arguments:

    <file>   path of the file to hash.

  `,
	Run:  manifestHashCmd,
	Flag: *flag.NewFlagSet("data-manifest-hash", flag.ExitOnError),
}

func init() {
	cmd_data_manifest_add.Flag.Bool("all", false, "add all available files")
	cmd_data_manifest_rm.Flag.Bool("all", false, "remove all tracked files")
	cmd_data_manifest_hash.Flag.Bool("all", false, "hash all tracked files")
}

func manifestCmd(c *commander.Command, args []string) error {
	mf := NewManifest("")
	return mf.Generate()
}

func manifestCmdPaths(c *commander.Command, args []string) ([]string, error) {
	mf := NewManifest("")
	paths := args

	// Use all files available if --all is passed in.
	all := c.Flag.Lookup("all").Value.Get().(bool)
	if all {
		paths = []string{}
		for path, _ := range *mf.Files {
			paths = append(paths, path)
		}
	}

	if len(paths) < 1 {
		return nil, fmt.Errorf("%v: no files specified.", c.FullName())
	}

	return paths, nil
}

func manifestAddCmd(c *commander.Command, args []string) error {
	mf := NewManifest("")
	paths := args

	// Use all files available if --all is passed in.
	all := c.Flag.Lookup("all").Value.Get().(bool)
	if all {
		paths = listAllFiles(".")
	}

	if len(paths) < 1 {
		return fmt.Errorf("%v: no files specified.", c.FullName())
	}

	// add files to manifest file
	for _, f := range paths {
		err := mf.Add(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func manifestRmCmd(c *commander.Command, args []string) error {
	mf := NewManifest("")

	paths, err := manifestCmdPaths(c, args)
	if err != nil {
		return err
	}

	// remove files from manifest file
	for _, f := range paths {
		err := mf.Remove(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func manifestHashCmd(c *commander.Command, args []string) error {
	mf := NewManifest("")

	paths, err := manifestCmdPaths(c, args)
	if err != nil {
		return err
	}

	// hash files in manifest file
	for _, f := range paths {
		err := mf.Hash(f)
		if err != nil {
			return err
		}
	}

	return nil
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
		err := mf.Add(f)
		if err != nil {
			return err
		}
	}

	// warn about manifest-listed files missing from directory
	// (basically, missing things. User removes individually, or `rm --missing`)

	// Once all files are listed, hash all the files, storing the hashes.
	for f, h := range *mf.Files {
		if isHash(h) && h != noHash {
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

func (mf *Manifest) Add(path string) error {
	// check, dont override (could have hash value)
	_, exists := (*mf.Files)[path]
	if exists {
		return nil
	}

	(*mf.Files)[path] = noHash

	// Write out file (store incrementally)
	err := mf.WriteFile()
	if err != nil {
		return err
	}

	pOut("data manifest: added %s\n", path)
	return nil
}

func (mf *Manifest) Remove(path string) error {
	// check, dont remove nonexistent path
	_, exists := (*mf.Files)[path]
	if !exists {
		return nil
	}

	delete(*mf.Files, path)

	// Write out file (store incrementally)
	err := mf.WriteFile()
	if err != nil {
		return err
	}

	pOut("data manifest: removed %s\n", path)
	return nil
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

func (mf *Manifest) Check(path string) error {
	oldHash, found := (*mf.Files)[path]
	if !found {
		return fmt.Errorf("data manifest: file not in manifest %s", path)
	}

	newHash, err := hashFile(path)
	if err != nil {
		return err
	}

	mfmt := "data manifest: checksum %.7s %s %s"
	if newHash != oldHash {
		pOut(mfmt, oldHash, path, "FAIL\n")
		return fmt.Errorf(mfmt, oldHash, path, "FAIL")
	}

	dOut(mfmt, oldHash, path, "PASS\n")
	return nil
}

func (mf *Manifest) PathsForHash(hash string) ([]string, error) {
	l := []string{}
	for path, h := range *mf.Files {
		if h == hash {
			l = append(l, path)
		}
	}

	if len(l) > 0 {
		return l, nil
	}

	return l, fmt.Errorf("Hash %v is not tracked in the manifest.", hash)
}

func (mf *Manifest) HashForPath(path string) (string, error) {
	hash, exists := (*mf.Files)[path]
	if exists {
		return hash, nil
	}
	return "", fmt.Errorf("Path %v is not tracked in the manifest.", path)
}

func (mf *Manifest) AllPaths() []string {
	l := []string{}
	for p, _ := range *mf.Files {
		l = append(l, p)
	}
	return l
}

func (mf *Manifest) AllHashes() []string {
	l := []string{}
	for _, h := range *mf.Files {
		l = append(l, h)
	}
	return l
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

	return readerHash(f)
}

func (mf *Manifest) ManifestHash() (string, error) {
	buf, err := mf.Marshal()
	if err != nil {
		return "", err
	}

	r := bytes.NewReader(buf)
	return readerHash(r)
}
