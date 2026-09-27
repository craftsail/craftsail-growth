// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (a *Asker) AskJSON(ctx context.Context, prompt string) (map[string]any, error) {
	if a == nil {
		return nil, fmt.Errorf("no asker")
	}
	plat := PickLLM("")
	if plat == "" {
		return nil, fmt.Errorf("no LLM key available")
	}
	res := a.AskCtx(ctx, plat, prompt+"\n\nOutput a single JSON object only, without markdown fences.")
	if !res.OK {
		return nil, fmt.Errorf("%s", res.Error)
	}
	return ExtractJSONMap(res.Answer)
}

func ExtractJSONMap(s string) (map[string]any, error) {
	s = strings.TrimSpace(s)
	i := strings.Index(s, "{")
	j := strings.LastIndex(s, "}")
	if i < 0 || j <= i {
		return nil, fmt.Errorf("the answer contains no JSON object")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s[i:j+1]), &m); err != nil {
		return nil, err
	}
	return m, nil
}
