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
	return []gscSlice{
		{Name: "query_page", Dimensions: []string{"date", "query", "page"}, DataState: "all", Types: all},
		{Name: "query_page_country_device", Dimensions: []string{"date", "query", "page", "country", "device"}, DataState: "all", Types: all},
		{Name: "country", Dimensions: []string{"date", "country"}, DataState: "all", Types: all},
		{Name: "device", Dimensions: []string{"date", "device"}, DataState: "all", Types: all},
		{Name: "appearance", Dimensions: []string{"date", "searchAppearance"}, DataState: "all", Types: all},
		{Name: "hour", Dimensions: []string{"date", "hour"}, DataState: "hourly_all", Types: []string{"web"}},
	}
}

func gscSliceByName(name string) (gscSlice, bool) {
	for _, s := range gscSlices() {
		if s.Name == name {
			return s, true
		}
	}
	return gscSlice{}, false
}
