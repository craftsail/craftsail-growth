// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import "errors"

var ErrSliceUnsupported = errors.New("gsc slice unsupported")

var GSCSearchTypes = []string{"web", "image", "video", "news", "discover", "googleNews"}

type gscSlice struct {
	Name       string
	Dimensions []string
	DataState  string
	Types      []string
}

func gscSlices() []gscSlice {
	all := GSCSearchTypes
	searchable := []string{"web", "image", "video", "news"}
	return []gscSlice{
		{Name: "page", Dimensions: []string{"date", "page"}, DataState: "final", Types: all},
		{Name: "query", Dimensions: []string{"date", "query"}, DataState: "final", Types: searchable},
		{Name: "country_device", Dimensions: []string{"date", "country", "device"}, DataState: "final", Types: all},
		{Name: "page_country_device", Dimensions: []string{"date", "page", "country", "device"}, DataState: "final", Types: all},
		{Name: "query_country_device", Dimensions: []string{"date", "query", "country", "device"}, DataState: "final", Types: searchable},
		{Name: "query_page", Dimensions: []string{"date", "query", "page"}, DataState: "all", Types: searchable},
		{Name: "query_page_country_device", Dimensions: []string{"date", "query", "page", "country", "device"}, DataState: "all", Types: searchable},
		{Name: "country", Dimensions: []string{"date", "country"}, DataState: "all", Types: all},
		{Name: "device", Dimensions: []string{"date", "device"}, DataState: "all", Types: all},
		{Name: "appearance", Dimensions: []string{"searchAppearance"}, DataState: "all", Types: all},
		{Name: "hour", Dimensions: []string{"date", "hour"}, DataState: "hourly_all", Types: []string{"web"}},
	}
}

func gscSliceByName(name string) (gscSlice, bool) {
	if name == "appearance_types" {
		return gscSlice{Name: name, Dimensions: []string{"searchAppearance"}, DataState: "final", Types: GSCSearchTypes}, true
	}
	for _, s := range gscSlices() {
		if s.Name == name {
			return s, true
		}
	}
	return gscSlice{}, false
}
