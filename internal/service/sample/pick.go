// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

var LLMPrefs = []string{"deepseek", "glm", "doubao", "openai", "gemini"}

func PickLLM(prefer string) string {
	cands := LLMPrefs
	if prefer != "" {
		cands = append([]string{prefer}, LLMPrefs...)
	}
	for _, c := range cands {
		if Available(c) {
			return c
		}
	}
	return ""
}
