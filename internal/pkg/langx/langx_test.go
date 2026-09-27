// SPDX-License-Identifier: AGPL-3.0-or-later

package langx

import "testing"

func TestDetect(t *testing.T) {
	cases := map[string]string{
		"":                           "en",
		"AcmeCLI converts documents": "en",
		"办公文档转换工具，支持 Word 和 Excel":     "zh",
		"Hello 你好 world of many words": "en",
	}
	for in, want := range cases {
		if got := Detect(in); got != want {
			t.Fatalf("Detect(%q) = %s, want %s", in, got, want)
		}
	}
}
