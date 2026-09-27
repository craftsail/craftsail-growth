// SPDX-License-Identifier: AGPL-3.0-or-later

package router

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/webembed"
)

func SPA(engine *gin.Engine) {
	sub, err := fs.Sub(webembed.Dist, "dist")
	if err != nil {
		sub = webembed.Dist
	}
	serveSPA(engine, sub)
}

func serveSPA(engine *gin.Engine, sub fs.FS) {
	engine.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api") {
			env, st := resp.Fail(resp.CodeNotFound, http.StatusNotFound, "not found")
			c.JSON(st, env)
			return
		}
		rel := strings.TrimPrefix(p, "/")
		if rel == "" {
			rel = "index.html"
		}
		b, err := fs.ReadFile(sub, rel)
		if err != nil {
			b, err = fs.ReadFile(sub, "index.html")
			if err != nil {
				// A plain `go build` embeds no dashboard; `make build` or the
				// Docker image does.
				c.String(http.StatusNotFound, "The dashboard is not built into this binary. Run `make build`, or use the Docker image. The API is at /api.\n")
				return
			}
			rel = "index.html"
		}
		ctype := mime.TypeByExtension(path.Ext(rel))
		if ctype == "" {
			ctype = "text/html; charset=utf-8"
		}
		c.Data(http.StatusOK, ctype, b)
	})
}
