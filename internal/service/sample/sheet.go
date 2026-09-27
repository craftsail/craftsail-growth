// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var BuyerGroups = map[string]bool{"价格": true, "推荐": true, "比较": true, "替代": true}

type SheetRec struct {
	Platform, QID, Question, Answer, SessionMode string
	WebQueries                                   []string
	Citations                                    []Citation
}

func RenderSheet(brand string, platforms []string, qs []Question, intent string, limit int, available func(string) bool) string {
	var plats []string
	for _, code := range platforms {
		p, ok := Lookup(code)
		if !ok {
			continue
		}
		if p.Manual || (available != nil && !available(code)) {
			plats = append(plats, code)
		}
	}
	tag := ""
	if intent == "buyer" {
		tag = " (buyer-intent weekly check)"
	}
	L := []string{
		fmt.Sprintf("# %s · Manual AI answer sampling sheet · %s%s", brand, time.Now().Format("2006-01-02"), tag),
		"",
		"How to use: ask each prompt on each engine and paste the **full answer text** into the ```answer block.",
		"",
		"**Rules:** private window, a new conversation for every prompt, paste the full text, and paste it even when the brand is not mentioned.",
		"",
	}
	for _, plat := range plats {
		qset := QuestionsFor(qs)
		if intent == "buyer" {
			var f []Question
			for _, q := range qset {
				if BuyerGroups[q.Group] {
					f = append(f, q)
				}
			}
			qset = f
		}
		if limit > 0 && len(qset) > limit {
			qset = qset[:limit]
		}
		if len(qset) == 0 {
			continue
		}
		L = append(L, fmt.Sprintf("## platform: %s", plat), fmt.Sprintf("> %s (%d prompts)", LabelOf(plat), len(qset)), "")
		for _, q := range qset {
			L = append(L, fmt.Sprintf("### %s · %s", q.ID, q.Text), "", "```answer", "", "```", "")
		}
	}
	return strings.Join(L, "\n")
}

var platSplit = regexp.MustCompile(`(?m)^##\s+platform:\s*(\S+)\s*$`)
var ansBlock = regexp.MustCompile(`(?ms)^###\s+(\S+)\s*·\s*(.+?)\n.*?` + "```answer\n(.*?)```")

func ParseSheet(text string) []SheetRec {
	blocks := platSplit.Split(text, -1)
	codes := platSplit.FindAllStringSubmatch(text, -1)
	var out []SheetRec
	for i, m := range codes {
		plat := m[1]
		body := ""
		if i+1 < len(blocks) {
			body = blocks[i+1]
		}
		for _, sm := range ansBlock.FindAllStringSubmatch(body, -1) {
			ans := strings.TrimSpace(sm[3])
			if ans == "" {
				continue
			}
			out = append(out, SheetRec{Platform: plat, QID: sm[1], Question: strings.TrimSpace(sm[2]), Answer: ans})
		}
	}
	return out
}
