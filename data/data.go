/*
Sniperkit-Bot
- Status: analyzed
*/

package main

import (
	"fmt"
	"os"

	"github.com/jbenet/data"
)

// This package (data/data) builds the `data` commandline tool.
// Everything is in the proper data library package. This extra
// package is necessary because packages must yield _either_ a
// library or executable. `data` needed to be both, hence this.

func main() {
	err := data.Cmd_data.Dispatch(os.Args[1:])
	if err != nil {
		if len(err.Error()) > 0 {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}
	return
}
