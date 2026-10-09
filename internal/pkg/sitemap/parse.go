// SPDX-License-Identifier: AGPL-3.0-or-later

// Package sitemap reads protocol XML without confusing image/video loc tags
// with page URLs. lastmod is never interpreted as a publication date.
package sitemap

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

const MaxBytes = 52428800
const MaxEntries = 50000

type Entry struct {
	URL          string `xml:"loc"`
	LastModified string `xml:"lastmod"`
}
type Document struct {
	Index   bool
	Entries []Entry
}

func Parse(reader io.Reader) (Document, error) {
	var out Document
	raw, err := io.ReadAll(io.LimitReader(reader, MaxBytes+1))
	if err != nil {
		return out, err
	}
	if len(raw) > MaxBytes {
		return out, fmt.Errorf("sitemap exceeds size limit")
	}
	if len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		gz, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return out, err
		}
		defer gz.Close()
		raw, err = io.ReadAll(io.LimitReader(gz, MaxBytes+1))
		if err != nil {
			return out, err
		}
		if len(raw) > MaxBytes {
			return out, fmt.Errorf("expanded sitemap exceeds size limit")
		}
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	depth := 0
	root := ""
	closed := false
	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Document{}, err
		}
		switch x := token.(type) {
		case xml.StartElement:
			depth++
			if depth == 1 {
				if closed {
					return Document{}, fmt.Errorf("multiple sitemap roots")
				}
				root = x.Name.Local
				if root != "urlset" && root != "sitemapindex" {
					return Document{}, fmt.Errorf("unsupported sitemap root %s", root)
				}
				out.Index = root == "sitemapindex"
			}
			kind := "url"
			if out.Index {
				kind = "sitemap"
			}
			if depth == 2 && x.Name.Local == kind {
				var e Entry
				if err := dec.DecodeElement(&e, &x); err != nil {
					return Document{}, err
				}
				depth--
				e.URL = strings.TrimSpace(e.URL)
				e.LastModified = strings.TrimSpace(e.LastModified)
				if e.URL == "" {
					return Document{}, fmt.Errorf("sitemap entry missing loc")
				}
				if len(e.URL) > 8192 {
					return Document{}, fmt.Errorf("sitemap URL too long")
				}
				out.Entries = append(out.Entries, e)
				if len(out.Entries) > MaxEntries {
					return Document{}, fmt.Errorf("sitemap exceeds entry limit")
				}
			}
		case xml.EndElement:
			depth--
			if depth == 0 {
				closed = true
			}
		}
	}
	if root == "" || !closed {
		return Document{}, fmt.Errorf("empty or incomplete sitemap")
	}
	return out, nil
}
