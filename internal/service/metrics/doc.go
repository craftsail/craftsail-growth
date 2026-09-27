// SPDX-License-Identifier: AGPL-3.0-or-later

// Package metrics is the only place that turns answer samples into rates.
//
// Every formula here is explained in the help center (How the numbers work).
// Dashboards, reports, opportunities and verification must call this package
// instead of dividing counts themselves.
package metrics
