// SPDX-License-Identifier: AGPL-3.0-or-later

// Command demo fills a fresh SQLite database with a fictional brand,
// Quillpad, so the dashboard can be tried and screenshotted without engine
// keys. Everything except the engines' answers runs through the real code:
// a local copy of the brand's website is crawled and audited, and each
// simulated day goes through sample.Run with generated answers, so runs,
// citations, metrics and opportunities are computed as in production.
//
//	go run ./scripts/demo -db /tmp/demo/data/craftsail-growth.db
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"hash/fnv"
	"io/fs"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
	"github.com/craftsail/craftsail-growth/internal/service/account"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/bootstrap"
	"github.com/craftsail/craftsail-growth/internal/service/crawl"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/report"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
)

//go:embed site
var siteFS embed.FS

const (
	siteURL = "http://quillpad.example"
	days    = 30
)

func main() {
	dbPath := flag.String("db", "data/demo.db", "SQLite file to create; must not exist")
	user := flag.String("user", "demo", "admin username to create")
	pass := flag.String("password", "quillpad-demo-2026", "admin password")
	lang := flag.String("lang", "en", "language of prompts and answers: en or zh")
	flag.Parse()
	c, ok := locales[*lang]
	if !ok {
		log.Fatalf("unknown -lang %q", *lang)
	}
	loc = c
	if _, err := os.Stat(*dbPath); err == nil {
		log.Fatalf("%s exists; the demo only writes to a new database", *dbPath)
	}
	if err := run(*dbPath, *user, *pass); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("demo ready: %s (sign in as %s)\n", *dbPath, *user)
}

func run(dbPath, user, pass string) error {
	proxy, err := serveSite()
	if err != nil {
		return err
	}
	// The crawler uses Go's default transport, which honours HTTP_PROXY, so
	// quillpad.example resolves to the local copy of the site.
	os.Setenv("HTTP_PROXY", proxy)
	os.Setenv("NO_PROXY", "")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return err
	}
	db, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_txlock=immediate"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return err
	}
	db.CreateBatchSize = 500
	if err := model.AutoMigrate(db); err != nil {
		return err
	}
	ctx := context.Background()

	if _, err := account.New(db).CreateUser(ctx, user, pass, model.RoleAdmin); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Quillpad", URL: siteURL, Slug: "quillpad", MaxPages: 25})
	if err != nil {
		return err
	}
	boot := bootstrap.New(db)
	if _, err := boot.SaveBrand(ctx, p.Slug, "Quillpad", brand()); err != nil {
		return err
	}
	if err := boot.SaveQuestions(ctx, p.Slug, questions()); err != nil {
		return err
	}
	if err := boot.SaveCompetitors(ctx, p.Slug, competitors()); err != nil {
		return err
	}

	log.Print("crawl and audit")
	if _, err := crawl.New(db, httputil.New()).Run(ctx, p.Slug, 25); err != nil {
		return err
	}
	if _, err := audit.New(db).Run(ctx, p.Slug); err != nil {
		return err
	}

	log.Printf("sample %d days", days)
	svc := sample.New(db, nil)
	svc.BypassAvailable = true
	svc.Sleep = nil
	gen := &answers{}
	svc.Ask = gen.ask
	today := time.Now()
	for d := days - 1; d >= 0; d-- {
		gen.day = days - 1 - d // 0 is the oldest day
		gen.calls = 0
		res, err := svc.Run(ctx, p.Slug, sample.RunInput{Platforms: loc.engines, FillTo: 3, Trigger: "schedule"})
		if err != nil {
			return err
		}
		if d > 0 {
			if err := backdate(db, p.ID, res.RunID, today, today.AddDate(0, 0, -d)); err != nil {
				return err
			}
		}
	}

	log.Print("opportunities and report")
	opp := opportunity.NewDB(db)
	items, err := opp.List(ctx, p.Slug, opportunity.ListFilter{})
	if err != nil {
		return err
	}
	// Accept the top two so the action plan shows work in progress.
	for i := 0; i < len(items) && i < 2; i++ {
		if _, err := opp.Accept(ctx, p.Slug, items[i].Key); err != nil {
			return err
		}
	}
	_, err = report.New(db).Build(ctx, p.Slug)
	return err
}

// backdate moves one run and everything it wrote from today to day, so the
// next run starts on an empty today.
func backdate(db *gorm.DB, projectID, runID uint64, today, day time.Time) error {
	from := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	to := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	shift := to.Unix() - from.Unix()
	var ids []uint64
	if err := db.Model(&model.Sample{}).Where("run_id = ?", runID).Pluck("id", &ids).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Sample{}).Where("run_id = ?", runID).Updates(map[string]any{
			"sampled_on": to, "created_at": gorm.Expr("created_at + ?", shift), "updated_at": gorm.Expr("updated_at + ?", shift),
		}).Error; err != nil {
			return err
		}
		if len(ids) > 0 {
			if err := tx.Model(&model.SampleCitation{}).Where("sample_id IN ?", ids).Update("sampled_on", to).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&model.Metric{}).Where("project_id = ? AND metric_on = ?", projectID, from).Update("metric_on", to).Error; err != nil {
			return err
		}
		return tx.Model(&model.SampleRun{}).Where("id = ?", runID).Updates(map[string]any{
			"started_at": gorm.Expr("started_at + ?", shift), "finished_at": gorm.Expr("finished_at + ?", shift),
			"created_at": gorm.Expr("created_at + ?", shift),
		}).Error
	})
}

// serveSite runs the brand's website on a loopback port and returns it as
// an HTTP proxy URL: proxied requests arrive with the full URL, and every
// host is answered from the embedded site.
func serveSite() (string, error) {
	sub, err := fs.Sub(siteFS, "site")
	if err != nil {
		return "", err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "quillpad.example" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		name := strings.Trim(r.URL.Path, "/")
		candidates := []string{name, name + ".html", name + "/index.html"}
		if name == "" {
			candidates = []string{"index.html"}
		}
		for _, c := range candidates {
			b, err := fs.ReadFile(sub, c)
			if err != nil {
				continue
			}
			switch {
			case strings.HasSuffix(c, ".txt"):
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			case strings.HasSuffix(c, ".xml"):
				w.Header().Set("Content-Type", "application/xml")
			default:
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
			}
			_, _ = w.Write(b)
			return
		}
		http.NotFound(w, r)
	}))
	return "http://" + ln.Addr().String(), nil
}

func brand() model.Brand {
	return model.Brand{
		Aliases:      []string{"Quillpad AI", "quillpad.example"},
		Products:     []string{"Quillpad Free", "Quillpad Pro", "Quillpad Business"},
		Industry:     "AI meeting assistants",
		TargetUsers:  "Remote and hybrid teams, sales teams, startups",
		BusinessGoal: "Be the AI meeting assistant engines recommend for remote teams",
		Definition:   "Quillpad is an AI meeting assistant that records, transcribes and summarizes video calls and turns decisions into action items.",
		Offers: []model.Offer{
			{Name: "Free", Price: "0", Currency: "USD", Desc: "10 meetings per month"},
			{Name: "Pro", Price: "12", Currency: "USD", Desc: "per user per month"},
			{Name: "Business", Price: "24", Currency: "USD", Desc: "per user per month, SSO and retention controls"},
		},
		KeyNumbers: []model.KeyNumber{
			{Fact: "Teams using Quillpad", Value: "4,000+", Source: siteURL + "/"},
			{Fact: "Languages", Value: "30", Source: siteURL + "/features"},
		},
		Suitable:   []string{"Remote teams with many video calls", "Sales teams that need call summaries"},
		Unsuitable: []string{"In-person meetings without a microphone"},
	}
}

func questions() []model.Question {
	out := make([]model.Question, len(loc.prompts))
	for i, q := range loc.prompts {
		out[i] = model.Question{QID: fmt.Sprintf("q%03d", i+1), GroupName: q.group, Market: model.MarketAll, Text: q.text, Enabled: true}
	}
	return out
}

type rival struct {
	name, site string
	base       float64 // chance to appear in an unbranded answer
}

var rivals = []rival{
	{"Scribely", "https://scribely.example", 0.62},
	{"NoteNest", "https://notenest.example", 0.46},
	{"Minutely", "https://minutely.example", 0.33},
	{"EchoNotes", "https://echonotes.example", 0.24},
}

func competitors() []model.Competitor {
	yes := true
	out := make([]model.Competitor, len(rivals))
	for i, r := range rivals {
		out[i] = model.Competitor{Name: r.name, Site: r.site, Market: model.MarketAll, Confirmed: &yes}
	}
	return out
}

var ownSources = []sample.Citation{
	{URL: siteURL + "/blog/ai-meeting-notes-guide", Title: "How to choose an AI meeting notes app in 2026"},
	{URL: siteURL + "/security", Title: "Security and privacy – Quillpad"},
	{URL: siteURL + "/pricing", Title: "Pricing – Quillpad"},
}

type answers struct {
	day   int // 0 is the oldest simulated day
	calls int
}

func (a *answers) rng(parts ...string) *rand.Rand {
	h := fnv.New64a()
	fmt.Fprintf(h, "%d|%d|%s", a.day, a.calls, strings.Join(parts, "|"))
	return rand.New(rand.NewSource(int64(h.Sum64())))
}

// ask writes an answer the way the engines tend to: a short ranked list.
// Quillpad starts behind Scribely and NoteNest and gains ground over the
// month, as if its content work were paying off.
func (a *answers) ask(platform, question string) sample.AskResult {
	a.calls++
	r := a.rng(platform, question)
	if r.Float64() < 0.02 {
		return sample.AskResult{Error: "HTTP 429 Too Many Requests: rate limit reached"}
	}
	progress := float64(a.day) / float64(days-1)
	branded := strings.Contains(question, "Quillpad")
	blurbs := loc.blurbs

	type entry struct {
		name, blurb string
		weight      float64
	}
	var list []entry
	brandP := loc.engineBase[platform] + 0.30*progress
	if branded {
		brandP = 0.97
	}
	if r.Float64() < brandP {
		list = append(list, entry{"Quillpad", blurbs["Quillpad"], 0.4 + progress + r.Float64()})
	}
	for _, c := range rivals {
		p := c.base
		if c.name == "EchoNotes" {
			p -= 0.10 * progress
		}
		if branded {
			p *= 0.6
		}
		if r.Float64() < p {
			list = append(list, entry{c.name, blurbs[c.name], c.base + r.Float64()})
		}
	}
	for i := range list {
		for j := i + 1; j < len(list); j++ {
			if list[j].weight > list[i].weight {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	var b strings.Builder
	switch {
	case branded:
		b.WriteString(loc.introBranded)
	case len(list) == 0:
		b.WriteString(loc.noAnswer)
	default:
		b.WriteString(loc.intro)
	}
	for i, e := range list {
		fmt.Fprintf(&b, loc.item, i+1, e.name, e.blurb)
	}
	if len(list) > 0 {
		b.WriteString(loc.outro)
	}

	res := sample.AskResult{OK: true, Answer: b.String(), Model: platform + "-demo"}
	if loc.searches[platform] {
		res.Searched = true
		n := 2 + r.Intn(3)
		for _, i := range r.Perm(len(loc.sources))[:n] {
			res.Citations = append(res.Citations, loc.sources[i])
		}
		if r.Float64() < 0.15+0.45*progress {
			res.Citations = append(res.Citations, ownSources[r.Intn(len(ownSources))])
		}
		res.WebQueries = webQueries(r, question)
	}
	return res
}

func webQueries(r *rand.Rand, question string) []string {
	base := strings.ToLower(strings.TrimRight(question, "?？"))
	pool := append([]string{base}, loc.queries...)
	n := 1 + r.Intn(3)
	out := []string{pool[0]}
	for _, i := range r.Perm(len(pool) - 1)[:n] {
		out = append(out, pool[i+1])
	}
	return out
}

// locale holds everything that depends on the language of the demo.
type locale struct {
	engines                                    []string
	engineBase                                 map[string]float64 // how often each engine knows the brand at the start
	searches                                   map[string]bool    // engines that cite sources and report web queries
	prompts                                    []struct{ group, text string }
	blurbs                                     map[string]string
	intro, introBranded, noAnswer, item, outro string
	sources                                    []sample.Citation
	queries                                    []string
}

var loc locale

var locales = map[string]locale{
	"en": {
		engines:    []string{"openai", "claude", "gemini", "perplexity", "deepseek"},
		engineBase: map[string]float64{"openai": 0.34, "claude": 0.26, "gemini": 0.40, "perplexity": 0.48, "deepseek": 0.18},
		searches:   map[string]bool{"openai": true, "gemini": true, "perplexity": true},
		prompts: []struct{ group, text string }{
			{"推荐", "What is the best AI meeting notes app for remote teams?"},
			{"推荐", "Which AI note taker works with Zoom and Google Meet?"},
			{"推荐", "What is the best tool to summarize sales calls automatically?"},
			{"比较", "Which AI meeting assistant is the most accurate for accents and crosstalk?"},
			{"替代", "What are good alternatives to writing meeting minutes by hand?"},
			{"价格", "What is the cheapest AI transcription tool for a small team?"},
			{"场景", "How do I get action items out of meetings automatically?"},
			{"场景", "Which AI meeting assistant is safest for GDPR and EU data?"},
			{"品牌验证", "Is Quillpad good for meeting notes?"},
			{"品牌验证", "Quillpad vs Scribely: which is better?"},
		},
		blurbs: map[string]string{
			"Quillpad":  "summaries and action items from every call, with EU data residency",
			"Scribely":  "accurate transcripts and a generous free tier",
			"NoteNest":  "notes organized into a shared team wiki",
			"Minutely":  "the lowest price per seat",
			"EchoNotes": "strong support for sales calls and CRM sync",
		},
		intro:        "Here are the tools teams mention most often:\n\n",
		introBranded: "Quillpad is an AI meeting assistant for remote teams. Here is how it compares with the tools people usually consider:\n\n",
		noAnswer:     "There is no single best tool; it depends on your meeting platform, languages and privacy needs. Test two or three options on real calls before you decide.",
		item:         "%d. **%s** – %s.\n",
		outro:        "\nStart with the free plans and compare how each one handles your own meetings.",
		sources: []sample.Citation{
			{URL: "https://www.reddit.com/r/productivity/comments/ai_meeting_notes", Title: "Which AI note taker do you actually use?"},
			{URL: "https://www.g2.com/categories/ai-meeting-assistants", Title: "Best AI meeting assistants"},
			{URL: "https://www.youtube.com/watch?v=meeting-notes-review", Title: "I tested 5 AI meeting note takers"},
			{URL: "https://medium.com/@remoteops/ai-meeting-tools-2026", Title: "AI meeting tools for remote teams"},
			{URL: "https://www.capterra.com/meeting-software/", Title: "Meeting software reviews"},
			{URL: "https://scribely.example/pricing", Title: "Scribely pricing"},
			{URL: "https://notenest.example/blog/meeting-notes-template", Title: "Meeting notes template"},
			{URL: "https://en.wikipedia.org/wiki/Minutes", Title: "Minutes"},
		},
		queries: []string{
			"best ai meeting notes app 2026",
			"ai note taker zoom google meet",
			"ai meeting assistant reviews reddit",
			"meeting transcription pricing per user",
			"ai meeting notes gdpr",
		},
	},
	"zh": {
		engines:    []string{"deepseek", "doubao", "kimi", "glm", "minimax"},
		engineBase: map[string]float64{"deepseek": 0.30, "doubao": 0.42, "kimi": 0.36, "glm": 0.24, "minimax": 0.18},
		searches:   map[string]bool{"doubao": true, "kimi": true},
		prompts: []struct{ group, text string }{
			{"推荐", "远程团队用什么 AI 会议纪要工具最好？"},
			{"推荐", "有哪些支持腾讯会议和飞书的 AI 会议记录工具？"},
			{"推荐", "销售电话自动生成总结，用什么工具？"},
			{"比较", "哪款 AI 会议助手对口音和多人插话识别最准？"},
			{"替代", "不想手写会议纪要，有什么替代办法？"},
			{"价格", "小团队用的 AI 转写工具，哪个最便宜？"},
			{"场景", "怎么从会议里自动提取待办事项？"},
			{"场景", "哪款 AI 会议助手在数据安全和合规上最放心？"},
			{"品牌验证", "Quillpad 做会议纪要好用吗？"},
			{"品牌验证", "Quillpad 和 Scribely 哪个更好？"},
		},
		blurbs: map[string]string{
			"Quillpad":  "每次会议自动生成总结和待办，支持数据境内存储",
			"Scribely":  "转写准确，免费额度多",
			"NoteNest":  "把会议记录整理成团队共享知识库",
			"Minutely":  "按人头计价最便宜",
			"EchoNotes": "适合销售电话，能同步到 CRM",
		},
		intro:        "大家最常提到的几款工具：\n\n",
		introBranded: "Quillpad 是一款面向远程团队的 AI 会议助手。和常见的几款工具相比：\n\n",
		noAnswer:     "没有绝对最好的工具，要看你用的会议软件、语言和数据安全要求。建议先用真实会议试用两三款再决定。",
		item:         "%d. **%s**：%s。\n",
		outro:        "\n可以先用免费版，拿自己的会议实际对比一下效果。",
		sources: []sample.Citation{
			{URL: "https://www.zhihu.com/question/ai-meeting-notes", Title: "有哪些好用的 AI 会议纪要工具？"},
			{URL: "https://sspai.com/post/ai-meeting-assistant", Title: "AI 会议助手横评"},
			{URL: "https://www.bilibili.com/video/ai-meeting-review", Title: "实测 5 款 AI 会议记录工具"},
			{URL: "https://36kr.com/p/ai-meeting-tools", Title: "AI 会议工具赛道观察"},
			{URL: "https://www.xiaohongshu.com/explore/meeting-notes", Title: "打工人必备的会议纪要神器"},
			{URL: "https://scribely.example/pricing", Title: "Scribely 价格"},
			{URL: "https://notenest.example/blog/meeting-notes-template", Title: "会议纪要模板"},
			{URL: "https://baike.baidu.com/item/会议纪要", Title: "会议纪要"},
		},
		queries: []string{
			"AI 会议纪要工具 推荐 2026",
			"AI 会议记录 腾讯会议 飞书",
			"AI 会议助手 测评 知乎",
			"会议转写 价格 按人收费",
			"AI 会议纪要 数据安全",
		},
	},
}
