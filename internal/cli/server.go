// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/api"
	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/router"
	"github.com/craftsail/craftsail-growth/internal/service/jobs"
	"github.com/craftsail/craftsail-growth/internal/service/monitor"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/webembed"
)

var sharedJobs *jobs.Service

func serverCmd() *cobra.Command {
	return httpServeCmd("server", "Start the dashboard HTTP server", true)
}

func uiCmd() *cobra.Command {
	return httpServeCmd("ui", "Start the dashboard and open it in a browser", false)
}

func httpServeCmd(use, short string, defaultNoOpen bool) *cobra.Command {
	var port int
	var noOpen bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			e := ego.New(ego.WithArguments(egoArgs()))
			host := firstNonEmpty(os.Getenv("CRAFTSAIL_GROWTH_HOST"), econf.GetString("server.http.host"), "127.0.0.1")
			token := strings.TrimSpace(os.Getenv("CRAFTSAIL_GROWTH_TOKEN"))
			if !isLoopback(host) && token == "" {
				return fmt.Errorf("binding to %s exposes the dashboard to the network; set CRAFTSAIL_GROWTH_TOKEN first", host)
			}
			if port <= 0 {
				port = econf.GetInt("server.http.port")
			}
			if port <= 0 {
				port = 8765
			}
			if !noOpen {
				go func() {
					time.Sleep(700 * time.Millisecond)
					openBrowser(formatListenURL(host, port))
				}()
			}
			// httpServer (and the api.Wire call inside it) needs invoker.DB,
			// so it is built inside the invoker, which ego runs right away.
			// If it fails, e.Run returns that error before anything serves.
			var srv *egin.Component
			e.Invoker(invoker.Init, invoker.Migrate, func() error {
				// Recompute citations for samples a crashed run left without them.
				if err := sample.BackfillCitations(cmd.Context(), invoker.DB); err != nil {
					return fmt.Errorf("backfill citations: %w", err)
				}
				sharedJobs = jobs.New(invoker.DB)
				jobs.Bind(sharedJobs, invoker.DB)
				if err := sharedJobs.ReapOrphans(cmd.Context()); err != nil {
					return err
				}
				sharedJobs.StartQueue(cmd.Context())
				monitor.Start(invoker.DB, sharedJobs)
				startLoopbackV6(loopbackPort(host, port))
				var err error
				srv, err = httpServer(host, token, port)
				return err
			})
			if srv == nil {
				return e.Run()
			}
			return e.Serve(srv).Run()
		},
	}
	cmd.Flags().IntVar(&port, "port", 0, "HTTP port (default from config)")
	cmd.Flags().BoolVar(&noOpen, "no-open", defaultNoOpen, "do not open a browser")
	return cmd
}

func httpServer(host, token string, port int) (*egin.Component, error) {
	opts := []egin.Option{
		egin.WithEmbedFs(webembed.Dist),
		egin.WithEmbedPath("dist"),
	}
	if host != "" {
		opts = append(opts, egin.WithHost(host))
	}
	if port > 0 {
		opts = append(opts, egin.WithPort(port))
	}
	srv := egin.Load("server.http").Build(opts...)
	h, err := api.Wire(invoker.DB, token, sharedJobs)
	if err != nil {
		return nil, fmt.Errorf("start dashboard: %w", err)
	}
	api.Mount(srv.Engine, h)
	router.SPA(srv.Engine)
	return srv, nil
}

// openBrowser is best effort; the URL is in the log either way.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func isLoopback(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "", "127.0.0.1", "localhost", "::1":
		return true
	default:
		return false
	}
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
