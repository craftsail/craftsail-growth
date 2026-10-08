// SPDX-License-Identifier: AGPL-3.0-or-later

package sitemap

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

func TestXMLGzipNamespaceAndNestedMedia(t *testing.T) {
	xml := `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:image="urn:image"><url><loc>https://a.com/a?x=1&amp;y=2</loc><lastmod>2026-01-01</lastmod><image:image><image:loc>https://a.com/image.jpg</image:loc></image:image></url></urlset>`
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	gz.Write([]byte(xml))
	gz.Close()
	for _, body := range [][]byte{[]byte(xml), b.Bytes()} {
		doc, err := Parse(bytes.NewReader(body))
		if err != nil || doc.Index || len(doc.Entries) != 1 || doc.Entries[0].URL != "https://a.com/a?x=1&y=2" {
			t.Fatalf("%#v %v", doc, err)
		}
	}
	doc, err := Parse(strings.NewReader(`<sitemapindex><sitemap><loc>https://a.com/child.xml.gz</loc></sitemap></sitemapindex>`))
	if err != nil || !doc.Index || len(doc.Entries) != 1 {
		t.Fatalf("%#v %v", doc, err)
	}
	for _, bad := range []string{`<html></html>`, `<urlset><url>`, `<urlset><url/></urlset>`, `<urlset/><urlset/>`} {
		if _, err := Parse(strings.NewReader(bad)); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}
