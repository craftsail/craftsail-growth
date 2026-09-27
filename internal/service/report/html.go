// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import (
	"fmt"
	"html"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/pkg/mdhtml"
)

const docCSS = `:root{--bg:#fdfcfa;--fg:#1f2328;--mut:#6b7280;--line:#e5e1d8;--acc:#1f4e79;--warn:#b4451f;--card:#fff}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);font:16px/1.75 -apple-system,"PingFang SC","Hiragino Sans GB","Microsoft YaHei",sans-serif}
.wrap{max-width:920px;margin:0 auto;padding:40px 24px 96px}
h1{font-size:28px;margin:0 0 4px;letter-spacing:-.01em}
h2{font-size:20px;margin:44px 0 14px;padding-bottom:8px;border-bottom:2px solid var(--line);color:var(--acc)}
h3{font-size:16px;margin:26px 0 10px}
table{border-collapse:collapse;width:100%;margin:14px 0;font-size:14px}
th,td{border:1px solid var(--line);padding:8px 10px;text-align:left;vertical-align:top}
th{background:#eef3f8;font-weight:600}
code{background:#eee;padding:1px 5px;border-radius:4px;font-size:13px}
a{color:var(--acc)}
ul{padding-left:22px}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:12px;margin:20px 0}
.card{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:14px 16px}
.card .k{font-size:12px;color:var(--mut)}
.card .v{font-size:26px;font-weight:650;line-height:1.3}
blockquote{color:var(--mut);border-left:3px solid var(--line);margin:12px 0;padding:0 12px}
hr{border:0;border-top:1px solid var(--line);margin:36px 0}
.scroll{overflow-x:auto}`

func Document(title, md string, cards [][2]string) string {
	var cardHTML strings.Builder
	for _, c := range cards {
		fmt.Fprintf(&cardHTML, `<div class="card"><div class="k">%s</div><div class="v">%s</div></div>`, html.EscapeString(c[0]), html.EscapeString(c[1]))
	}
	body := mdhtml.Document(md)
	body = strings.ReplaceAll(body, "P0 ", `<span style="color:var(--warn);font-weight:700">P0</span> `)
	return fmt.Sprintf(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title><style>%s</style></head><body><div class="wrap"><div class="cards">%s</div>%s</div></body></html>`,
		html.EscapeString(title), docCSS, cardHTML.String(), body)
}
