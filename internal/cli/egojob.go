// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/task/ejob"

	"github.com/craftsail/craftsail-growth/internal/invoker"
)

func runJob(name string, fn func(ejob.Context) error) error {
	return ego.New(ego.WithArguments(egoArgs("--job", name))).
		Invoker(invoker.Init, invoker.Migrate).
		Job(ejob.Job(name, fn)).
		Run()
}
