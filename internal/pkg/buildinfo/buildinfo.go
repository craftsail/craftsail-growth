// SPDX-License-Identifier: AGPL-3.0-or-later

// Package buildinfo contains metadata supplied by the release build.
package buildinfo

var (
	Version    = "dev"
	Commit     = "unknown"
	Date       = "unknown"
	BuildType  = "source"
	Repository = "craftsail/craftsail-growth"
)

type Info struct {
	Version    string `json:"version"`
	Commit     string `json:"commit"`
	Date       string `json:"date"`
	BuildType  string `json:"build_type"`
	Repository string `json:"repository"`
}

func Current() Info { return Info{Version, Commit, Date, BuildType, Repository} }
