// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"github.com/craftsail/craftsail-growth/internal/model"
	"net/url"
	"os"
	"strings"
)

type Provider struct {
	Code     string
	Name     string
	Market   string
	Base     string
	BaseEnv  string
	Model    string
	ModelEnv string
	KeyEnv   string
	Search   bool
	Protocol string
	Note     string
	Manual   bool
}

var Providers = []Provider{
	{Code: "glm", Name: "Zhipu GLM", Market: "cn", Base: "https://open.bigmodel.cn/api/paas/v4", BaseEnv: "GLM_BASE", Model: "glm-4-flash", ModelEnv: "GLM_MODEL", KeyEnv: "ZHIPUAI_API_KEY"},
	{Code: "doubao", Name: "Doubao (Ark API)", Market: "cn", Base: "https://ark.cn-beijing.volces.com/api/v3", BaseEnv: "ARK_BASE", Model: "doubao-seed-1-6-250615", ModelEnv: "ARK_MODEL", KeyEnv: "ARK_API_KEY", Search: true, Protocol: "ark"},
	{Code: "deepseek", Name: "DeepSeek", Market: "cn", Base: "https://api.deepseek.com/v1", BaseEnv: "DEEPSEEK_BASE", Model: "deepseek-v4-flash", ModelEnv: "DEEPSEEK_MODEL", KeyEnv: "DEEPSEEK_API_KEY"},
	{Code: "kimi", Name: "Kimi", Market: "cn", Base: "https://api.moonshot.cn/v1", BaseEnv: "MOONSHOT_BASE", Model: "kimi-k2-0905-preview", ModelEnv: "MOONSHOT_MODEL", KeyEnv: "MOONSHOT_API_KEY"},
	{Code: "minimax", Name: "MiniMax", Market: "cn", Base: "https://api.minimaxi.com/v1", BaseEnv: "MINIMAX_BASE", Model: "MiniMax-M2", ModelEnv: "MINIMAX_MODEL", KeyEnv: "MINIMAX_API_KEY"},
	{Code: "gemini", Name: "Gemini", Market: "global", Base: "https://generativelanguage.googleapis.com/v1beta/openai", BaseEnv: "GEMINI_BASE", Model: "gemini-2.5-flash", ModelEnv: "GEMINI_MODEL", KeyEnv: "GEMINI_API_KEY"},
	{Code: "openai", Name: "OpenAI(ChatGPT)", Market: "global", Base: "https://api.openai.com/v1", BaseEnv: "OPENAI_BASE", Model: "gpt-4o-mini", ModelEnv: "OPENAI_MODEL", KeyEnv: "OPENAI_API_KEY"},
	{Code: "claude", Name: "Claude", Market: "global", Base: "https://api.anthropic.com/v1", BaseEnv: "ANTHROPIC_BASE", Model: "claude-sonnet-5", ModelEnv: "ANTHROPIC_MODEL", KeyEnv: "ANTHROPIC_API_KEY", Protocol: "anthropic"},
	{Code: "grok", Name: "Grok", Market: "global", Base: "https://api.x.ai/v1", BaseEnv: "GROK_BASE", Model: "grok-3-mini", ModelEnv: "GROK_MODEL", KeyEnv: "XAI_API_KEY"},
	{Code: "perplexity", Name: "Perplexity", Market: "global", Base: "https://api.perplexity.ai", BaseEnv: "PERPLEXITY_BASE", Model: "sonar", ModelEnv: "PERPLEXITY_MODEL", KeyEnv: "PERPLEXITY_API_KEY", Search: true},
}

var ManualOnly = []Provider{
	{Code: "nano_ai", Name: "Nano AI Search (360)", Market: "cn", Manual: true},
	{Code: "baidu", Name: "Baidu AI Search", Market: "cn", Manual: true},
	{Code: "doubao_app", Name: "Doubao app / web (differs from the Ark API; sample separately)", Market: "cn", Manual: true},
	{Code: "chatgpt", Name: "ChatGPT web (Search on)", Market: "global", Manual: true},
	{Code: "claude_web", Name: "Claude web (Web Search on)", Market: "global", Manual: true},
	{Code: "google_aio", Name: "Google AI Overviews (record \"not triggered\" when absent)", Market: "global", Manual: true},
	{Code: "metaso", Name: "Metaso AI Search (citations are footnotes, often no links)", Market: "cn", Manual: true},
}

func Lookup(code string) (Provider, bool) {
	for _, p := range Providers {
		if p.Code == code {
			return p, true
		}
	}
	for _, p := range ManualOnly {
		if p.Code == code {
			return p, true
		}
	}
	return Provider{}, false
}

func MarketOf(platform string) string {
	if p, ok := Lookup(platform); ok {
		return p.Market
	}
	return "unknown"
}

func LabelOf(platform string) string {
	if p, ok := Lookup(platform); ok {
		return p.Name
	}
	return platform
}

func Available(platform string) bool {
	p, ok := Lookup(platform)
	if !ok || p.Manual {
		return false
	}
	return os.Getenv(p.KeyEnv) != ""
}

func ModelFor(p Provider) string {
	if p.ModelEnv != "" {
		if v := os.Getenv(p.ModelEnv); v != "" {
			return v
		}
	}
	return p.Model
}

func NormalizeBase(custom, official string) string {
	off := strings.TrimRight(strings.TrimSpace(official), "/")
	s := strings.TrimSpace(custom)
	if s == "" {
		return off
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		s = "https://" + s
	}
	cu, err := url.Parse(s)
	if err != nil || cu.Host == "" {
		return strings.TrimRight(s, "/")
	}
	ou, _ := url.Parse(off)
	if (cu.Path == "" || cu.Path == "/") && ou != nil && ou.Path != "" && ou.Path != "/" {
		cu.Path = ou.Path
	}
	return strings.TrimRight(cu.String(), "/")
}

func BaseForEnv(p Provider, env func(string) string) string {
	if p.BaseEnv != "" && env != nil {
		if v := strings.TrimSpace(env(p.BaseEnv)); v != "" {
			return NormalizeBase(v, p.Base)
		}
	}
	return strings.TrimRight(p.Base, "/")
}

func BaseFor(p Provider) string {
	return BaseForEnv(p, os.Getenv)
}

func EffectiveBaseUpdate(env, value string) string {
	if env == "" {
		return value
	}
	for _, p := range Providers {
		if p.BaseEnv == env {
			got := NormalizeBase(value, p.Base)
			if got == strings.TrimRight(p.Base, "/") {
				return ""
			}
			return got
		}
	}
	return strings.TrimSpace(value)
}

// QuestionsFor returns the prompts to ask an engine: every enabled prompt.
// Prompts are not routed by market.
func QuestionsFor(qs []Question) []Question {
	var out []Question
	for _, q := range qs {
		if !q.Off {
			out = append(out, q)
		}
	}
	return out
}

type Question struct {
	Language, Revision      string
	Record                  model.Question
	ID, Group, Market, Text string
	Tags                    []string
	Enabled                 bool
	Off                     bool
}

func KeyStatus() []map[string]any {
	var out []map[string]any
	for _, p := range Providers {
		v := os.Getenv(p.KeyEnv)
		tail := ""
		if len(v) >= 4 {
			tail = v[len(v)-4:]
		} else if v != "" {
			tail = v
		}
		custom := os.Getenv(p.BaseEnv)
		modelOverride := p.ModelEnv != "" && strings.TrimSpace(os.Getenv(p.ModelEnv)) != ""
		out = append(out, map[string]any{
			"code": p.Code, "name": p.Name, "market": p.Market,
			"key_env": p.KeyEnv, "model_env": p.ModelEnv, "model": ModelFor(p),
			"ready": v != "", "key_tail": tail, "note": p.Note,
			"search": p.Search, "manual": false,
			"official_base": strings.TrimRight(p.Base, "/"),
			"base":          BaseFor(p),
			"base_env":      p.BaseEnv,
			"custom_base":   strings.TrimSpace(custom) != "",
			"custom_model":  modelOverride,
		})
	}
	for _, p := range ManualOnly {
		out = append(out, map[string]any{
			"code": p.Code, "name": p.Name, "market": p.Market,
			"ready": false, "manual": true, "note": p.Note,
		})
	}
	return out
}
