// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

// Token estimate per call. It is a planning number shown before a run, not a
// bill; actual usage is stored separately when a provider reports it.
const (
	estPromptTokens = 60
	estAnswerTokens = 900
)

func estimateTokens(prompts, engines, rounds int) int {
	return prompts * engines * rounds * (estPromptTokens + estAnswerTokens)
}

func runOutcome(ok, fail int) string {
	switch {
	case ok > 0 && fail == 0:
		return "succeeded"
	case ok > 0:
		return "partial"
	default:
		return "failed"
	}
}

// call is one planned question to one engine in one round.
type call struct {
	plat  string
	q     Question
	round int
}
