// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"github.com/craftsail/craftsail-growth/internal/cli"
	"github.com/craftsail/craftsail-growth/internal/pkg/buildinfo"
	"os"
)

func main() {
	// Metadata must work without config, database migrations or other side effects.
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "--version":
			fmt.Println(buildinfo.Version)
			return
		case "--version-json":
			_ = json.NewEncoder(os.Stdout).Encode(buildinfo.Current())
			return
		}
	}
	cli.Execute()
}
