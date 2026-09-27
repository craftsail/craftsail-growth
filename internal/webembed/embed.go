// SPDX-License-Identifier: AGPL-3.0-or-later

package webembed

import "embed"

// Dist is the production Vite build. `make web` copies web/dist here.
//
//go:embed all:dist
var Dist embed.FS
