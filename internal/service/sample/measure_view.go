// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/metrics"
)

const measureRowCap = 20000

type MeasureQuery struct {
	Access   string // api | web; "" picks the larger group
	Range    string
	Platform string
	Tag      string
	TagList  []string
	QID      string
	Sort     string
}

type MeasurePoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type MeasureLeader struct {
	Name     string  `json:"name"`
	Mentions int     `json:"mentions"`
	Share    float64 `json:"share"`
	Prompts  int     `json:"prompts"`
	IsBrand  bool    `json:"is_brand"`
}

type MeasurePrompt struct {
	ID         string           `json:"id"`
	Text       string           `json:"text"`
	Group      string           `json:"group"`
	Runs       int              `json:"runs"`
	Failed     int              `json:"failed"`
	Visibility float64          `json:"visibility"`
	X          int              `json:"x"`
	N          int              `json:"n"`
	Series     []map[string]any `json:"series"`
}

type MeasureWord struct {
	Word  string  `json:"word"`
	Count int     `json:"count"`
	Share float64 `json:"share"`
}

type MeasureOpt struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type MeasureGap struct {
	QID     string   `json:"qid"`
	Text    string   `json:"text"`
	Domains []string `json:"domains"`
}

type MeasureChip struct {
	Name string `json:"name"`
	Self bool   `json:"self"`
}

type MeasureCite struct {
	URL      string `json:"url"`
	Domain   string `json:"domain"`
	Title    string `json:"title"`
	Category string `json:"category"`
	PageType string `json:"page_type"`
}

type MeasureRun struct {
	ID           uint64         `json:"id"`
	Platform     string         `json:"platform"`
	PlatformName string         `json:"platform_name"`
	Access       string         `json:"access"`
	At           int64          `json:"at"`
	OK           bool           `json:"ok"`
	Error        string         `json:"error,omitempty"`
	Queries      []string       `json:"queries"`
	Unknown      bool           `json:"unknown"`
	Mentioned    bool           `json:"mentioned"`
	Chips        []MeasureChip  `json:"chips"`
	Answer       string         `json:"answer"`
	Raw          map[string]any `json:"raw,omitempty"`
	Citations    []MeasureCite  `json:"citations"`
}

type MeasurePromptHead struct {
	ID         string           `json:"id"`
	Text       string           `json:"text"`
	Group      string           `json:"group"`
	Active     bool             `json:"active"`
	Next       string           `json:"next"`
	Runs       int              `json:"runs"`
	Visibility *float64         `json:"visibility"`
	Series     []map[string]any `json:"series"`
}

type MeasureNotes struct {
	Visibility    string `json:"visibility"`
	Share         string `json:"share"`
	CitationShare string `json:"citation_share"`
	Fanout        string `json:"fanout"`
	Stack         string `json:"stack"`
}

type MeasureView struct {
	Brand            string             `json:"brand"`
	Range            string             `json:"range"`
	Visibility       *float64           `json:"visibility"`
	Access           string             `json:"access"`
	Accesses         []string           `json:"accesses"`
	VisibilityX      int                `json:"visibility_x"`
	VisibilityN      int                `json:"visibility_n"`
	VisibilityCI     *metrics.Interval  `json:"visibility_ci"`
	LowSample        bool               `json:"low_sample"`
	Recognition      *float64           `json:"recognition"`
	RecognitionN     int                `json:"recognition_n"`
	OwnCited         *float64           `json:"own_cited"`
	VisibilitySeries []MeasurePoint     `json:"visibility_series"`
	Share            *float64           `json:"share"`
	ShareSeries      []MeasurePoint     `json:"share_series"`
	Prompts          int                `json:"prompts"`
	Runs             int                `json:"runs"`
	Failed           int                `json:"failed"`
	FailReason       string             `json:"fail_reason"`
	Citations        int                `json:"citations"`
	CitationShare    *float64           `json:"citation_share"`
	UniqueDomains    int                `json:"unique_domains"`
	Leaders          []MeasureLeader    `json:"leaders"`
	PromptCharts     []MeasurePrompt    `json:"prompt_charts"`
	FanoutTotal      int                `json:"fanout_total"`
	FanoutUnknown    int                `json:"fanout_unknown"`
	FanoutKnown      int                `json:"fanout_known"`
	FanoutAvg        float64            `json:"fanout_avg"`
	Words            []MeasureWord      `json:"words"`
	Added            []MeasureWord      `json:"added"`
	Preserved        []MeasureWord      `json:"preserved"`
	Dropped          []MeasureWord      `json:"dropped"`
	CategorySeries   []map[string]any   `json:"category_series"`
	PageTypeSeries   []map[string]any   `json:"page_type_series"`
	CategoryKeys     []string           `json:"category_keys"`
	PageTypeKeys     []string           `json:"page_type_keys"`
	Opportunities    []Opportunity      `json:"opportunities"`
	Gaps             []MeasureGap       `json:"gaps"`
	Competitors      []string           `json:"competitors"`
	Engines          []MeasureOpt       `json:"engines"`
	Tags             []MeasureOpt       `json:"tags"`
	Cadence          string             `json:"cadence"`
	UpdatedAt        int64              `json:"updated_at"`
	Truncated        bool               `json:"truncated"`
	Prompt           *MeasurePromptHead `json:"prompt,omitempty"`
	RunsDetail       []MeasureRun       `json:"run_list,omitempty"`
	Notes            MeasureNotes       `json:"notes"`
}

type measureBuild struct {
	Brand       string
	Aliases     []string
	Site        string
	Competitors []string
	Questions   []Question
	Rows        []model.Sample
	Cites       []model.SampleCitation
	Query       MeasureQuery
	Now         time.Time
}

func (s *Service) Measure(ctx context.Context, slug string, q MeasureQuery) (*MeasureView, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	q = normalizeMeasureQuery(q)
	return s.measure(ctx, p, q, measureSince(q.Range, time.Now()), time.Time{})
}

// MeasureWindow is Measure over [now-fromDaysAgo, now-toDaysAgo) for one
// access type, so two periods can be compared on the same basis.
func (s *Service) MeasureWindow(ctx context.Context, slug, access string, fromDaysAgo, toDaysAgo int) (*MeasureView, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	q := normalizeMeasureQuery(MeasureQuery{Range: "all", Access: access})
	return s.measure(ctx, p, q, now.AddDate(0, 0, -fromDaysAgo), now.AddDate(0, 0, -toDaysAgo))
}

func (s *Service) measure(ctx context.Context, p *model.Project, q MeasureQuery, since, until time.Time) (*MeasureView, error) {
	cfg, qs, err := s.cfgOf(ctx, p)
	if err != nil {
		return nil, err
	}
	var rows []model.Sample
	dbq := s.samples.DB.WithContext(ctx).Where("project_id = ?", p.ID)
	if !since.IsZero() {
		dbq = dbq.Where("sampled_on >= ?", since.Format("2006-01-02"))
	}
	if !until.IsZero() {
		dbq = dbq.Where("sampled_on < ?", until.Format("2006-01-02"))
	}
	if err := dbq.Order("sampled_on, id").Limit(measureRowCap).Find(&rows).Error; err != nil {
		return nil, err
	}
	cites, err := s.samples.ListCitations(ctx, p.ID, since)
	if err != nil {
		return nil, err
	}
	comps := make([]string, 0, len(cfg.Competitors))
	for _, c := range cfg.Competitors {
		if c.Name != "" {
			comps = append(comps, c.Name)
		}
	}
	view := buildMeasure(measureBuild{
		Brand: p.Name, Aliases: cfg.Aliases, Site: cfg.Site, Competitors: comps, Questions: qs,
		Rows: rows, Cites: cites, Query: q, Now: time.Now(),
	})
	view.Truncated = len(rows) >= measureRowCap
	return view, nil
}

func normalizeMeasureQuery(q MeasureQuery) MeasureQuery {
	switch q.Range {
	case "7d", "30d", "90d", "180d", "365d", "all":
	default:
		q.Range = "30d"
	}
	if q.Sort != "desc" {
		q.Sort = "asc"
	}
	q.Platform = strings.TrimSpace(q.Platform)
	q.Tag = strings.TrimSpace(q.Tag)
	q.QID = strings.TrimSpace(q.QID)
	q.TagList = parseTagList(q.Tag)
	if q.Access != "api" && q.Access != "web" {
		q.Access = ""
	}
	return q
}

func parseTagList(raw string) []string {
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		switch part {
		case "品牌题":
			part = model.TagBranded
		case "买家题":
			part = model.TagUnbranded
		}
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}

func measureSince(rangeKey string, now time.Time) time.Time {
	switch rangeKey {
	case "7d":
		return now.AddDate(0, 0, -7)
	case "90d":
		return now.AddDate(0, 0, -90)
	case "180d":
		return now.AddDate(0, 0, -180)
	case "365d":
		return now.AddDate(0, 0, -365)
	case "all":
		return time.Time{}
	default:
		return now.AddDate(0, 0, -30)
	}
}

func buildMeasure(in measureBuild) *MeasureView {
	q := normalizeMeasureQuery(in.Query)
	if in.Now.IsZero() {
		in.Now = time.Now()
	}
	view := &MeasureView{
		Brand: q.Range, Range: q.Range,
		Leaders: []MeasureLeader{}, PromptCharts: []MeasurePrompt{},
		Opportunities: []Opportunity{}, Gaps: []MeasureGap{},
		Competitors: append([]string{}, in.Competitors...),
		Engines:     []MeasureOpt{}, Tags: []MeasureOpt{},
		Cadence: "Up to 3 runs per engine per day",
		Notes: MeasureNotes{
			Visibility:    "Share of successful unbranded runs that mention the brand, for one access type (API or Web). The band is a 95% Wilson interval. Branded prompts are reported as recognition.",
			Share:         "Brand mentions divided by brand mentions plus competitor mentions, unbranded prompts only. Two competitors in one answer count twice. The leaderboard and donut use the same number.",
			CitationShare: "Owned-domain citation rows divided by citation rows in the same filter.",
			Fanout:        "Unknown plus known equals successful runs. Runs without a query string are unknown and stay out of the cloud. Average fan-out uses known runs only.",
			Stack:         "Each day is a 0–100% mix. The cards are the whole period, so the last day need not match the card.",
		},
	}
	view.Brand = in.Brand
	groups := map[string]string{}
	texts := map[string]string{}
	active := map[string]bool{}
	for _, item := range in.Questions {
		groups[item.ID] = item.Group
		texts[item.ID] = item.Text
		active[item.ID] = true
	}
	view.Engines = measureEngines(in.Rows)
	view.Tags = measureTagOptions(in.Questions)

	branded := map[uint64]bool{}
	allObs := make([]metrics.Obs, 0, len(in.Rows))
	for _, sm := range in.Rows {
		b := measureBranded(sm, in)
		branded[sm.ID] = b
		allObs = append(allObs, measureObs(sm, b))
	}
	view.Accesses = metrics.Accesses(allObs)
	if q.Access == "" {
		q.Access = metrics.DefaultAccess(allObs, q.Platform)
	}
	view.Access = q.Access

	var filtered []model.Sample
	var keptObs []metrics.Obs
	for _, sm := range in.Rows {
		if keepMeasureSample(sm, q, in) {
			filtered = append(filtered, sm)
			keptObs = append(keptObs, measureObs(sm, branded[sm.ID]))
		}
	}
	okIDs := map[uint64]bool{}
	byDay := map[string]*dayAcc{}
	byQ := map[string]*qAcc{}
	var qids []string
	compPrompts := map[string]map[string]bool{}
	brandPrompts := map[string]bool{}
	compMentions := map[string]int{}
	var queryIn []struct {
		Question string
		Queries  []string
	}
	tokenCounts := map[string]int{}
	failCounts := map[string]int{}
	for _, sm := range filtered {
		if !sm.OK {
			view.Failed++
			qa := ensureQ(byQ, &qids, sm, texts, groups)
			qa.failed++
			if qa.failErr == "" {
				qa.failErr = sm.Error
			}
			if reason := shortErr(sm.Error); reason != "" {
				failCounts[reason]++
			}
			if sm.CreatedAt > view.UpdatedAt {
				view.UpdatedAt = sm.CreatedAt
			}
			continue
		}
		okIDs[sm.ID] = true
		if sm.CreatedAt > view.UpdatedAt {
			view.UpdatedAt = sm.CreatedAt
		}
		day := sm.SampledOn.Format("2006-01-02")
		d := byDay[day]
		if d == nil {
			d = &dayAcc{}
			byDay[day] = d
		}
		isBranded := branded[sm.ID]
		if !isBranded {
			d.ok++
		}
		view.Runs++
		qa := ensureQ(byQ, &qids, sm, texts, groups)
		qa.ok++
		qa.dayN[day]++
		if qa.days[day] == nil {
			qa.days[day] = map[string]int{}
		}
		if sm.Mentioned {
			qa.mentioned++
			qa.days[day]["brand"]++
			if !isBranded {
				d.mentioned++
				d.brandMentions++
				brandPrompts[sm.QID] = true
			}
		}
		for _, name := range sm.CompetitorsMentioned {
			if name == "" {
				continue
			}
			qa.days[day][name]++
			qa.compSeen[name] = true
			if isBranded {
				continue
			}
			d.compEvents++
			compMentions[name]++
			if compPrompts[name] == nil {
				compPrompts[name] = map[string]bool{}
			}
			compPrompts[name][sm.QID] = true
		}
		known := knownQueries(sm.WebQueries)
		if len(known) == 0 {
			view.FanoutUnknown++
		} else {
			view.FanoutKnown++
			view.FanoutAvg += float64(len(known))
			for _, wq := range known {
				for tok := range tokenSet(wq) {
					if tok == WebQueriesUnavailable {
						continue
					}
					tokenCounts[tok]++
				}
			}
		}
		queryIn = append(queryIn, struct {
			Question string
			Queries  []string
		}{qa.text, known})
	}
	view.Prompts = len(qids)
	view.FailReason = topReason(failCounts)
	if view.Runs > 0 {
		view.FailReason = ""
	}
	view.FanoutTotal = view.Runs
	if view.FanoutKnown > 0 {
		view.FanoutAvg = view.FanoutAvg / float64(view.FanoutKnown)
	}
	snapM := metrics.Compute(keptObs, metrics.Filter{Access: q.Access, Platform: q.Platform})
	view.VisibilityX = snapM.Visibility.X
	view.VisibilityN = snapM.Visibility.N
	view.LowSample = snapM.Visibility.LowSample
	if snapM.Visibility.Value != nil {
		v := *snapM.Visibility.Value * 100
		view.Visibility = &v
		view.VisibilityCI = &metrics.Interval{Lo: snapM.Visibility.CI.Lo * 100, Hi: snapM.Visibility.CI.Hi * 100}
	}
	view.RecognitionN = snapM.Recognition.N
	if snapM.Recognition.Value != nil {
		r := *snapM.Recognition.Value * 100
		view.Recognition = &r
	}
	if snapM.OwnCited.Value != nil && strings.TrimSpace(in.Site) != "" {
		oc := *snapM.OwnCited.Value * 100
		view.OwnCited = &oc
	}
	var days []string
	for day := range byDay {
		days = append(days, day)
	}
	sort.Strings(days)
	brandEvents := 0
	for _, day := range days {
		d := byDay[day]
		brandEvents += d.brandMentions
		if d.ok > 0 {
			view.VisibilitySeries = append(view.VisibilitySeries, MeasurePoint{Date: day, Value: float64(d.mentioned) / float64(d.ok) * 100})
		}
		total := d.brandMentions + d.compEvents
		if total > 0 {
			view.ShareSeries = append(view.ShareSeries, MeasurePoint{Date: day, Value: float64(d.brandMentions) / float64(total) * 100})
		}
	}
	totalMentions := brandEvents
	for _, n := range compMentions {
		totalMentions += n
	}
	if snapM.ShareOfVoice != nil && totalMentions > 0 {
		share := *snapM.ShareOfVoice * 100
		view.Share = &share
		view.Leaders = append(view.Leaders, MeasureLeader{Name: in.Brand, Mentions: brandEvents, Share: share, Prompts: len(brandPrompts), IsBrand: true})
		for name, n := range compMentions {
			view.Leaders = append(view.Leaders, MeasureLeader{
				Name: name, Mentions: n, Share: float64(n) / float64(totalMentions) * 100, Prompts: len(compPrompts[name]),
			})
		}
		sort.Slice(view.Leaders, func(i, j int) bool {
			if view.Leaders[i].Mentions == view.Leaders[j].Mentions {
				return view.Leaders[i].Name < view.Leaders[j].Name
			}
			return view.Leaders[i].Mentions > view.Leaders[j].Mentions
		})
	}
	for _, qid := range qids {
		qa := byQ[qid]
		vis := 0.0
		if qa.ok > 0 {
			vis = float64(qa.mentioned) / float64(qa.ok) * 100
		}
		var qd []string
		for day := range qa.dayN {
			qd = append(qd, day)
		}
		sort.Strings(qd)
		series := []map[string]any{}
		var names []string
		for name := range qa.compSeen {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, day := range qd {
			row := map[string]any{"date": day, "brand": pct(qa.days[day]["brand"], qa.dayN[day])}
			for _, name := range names {
				row[name] = pct(qa.days[day][name], qa.dayN[day])
			}
			series = append(series, row)
		}
		view.PromptCharts = append(view.PromptCharts, MeasurePrompt{
			ID: qid, Text: qa.text, Group: qa.group, Runs: qa.ok, Failed: qa.failed, Visibility: vis, X: qa.mentioned, N: qa.ok, Series: series,
		})
	}
	sort.Slice(view.PromptCharts, func(i, j int) bool {
		if q.Sort == "desc" {
			return view.PromptCharts[i].Visibility > view.PromptCharts[j].Visibility
		}
		return view.PromptCharts[i].Visibility < view.PromptCharts[j].Visibility
	})
	snap := SummarizeQueries(queryIn)
	view.Words = wordsFrom(topWords(tokenCounts, 40), sumCounts(tokenCounts))
	view.Added = wordsFrom(snap.Added, sumStats(snap.Added))
	view.Preserved = wordsFrom(snap.Preserved, sumStats(snap.Preserved))
	view.Dropped = wordsFrom(snap.Dropped, sumStats(snap.Dropped))

	ownHost := bareHost(in.Site)
	var used []model.SampleCitation
	for _, c := range in.Cites {
		if c.SampleID != 0 && !okIDs[c.SampleID] {
			continue
		}
		if c.SampleID == 0 && !citationMatches(c, filtered) {
			continue
		}
		used = append(used, c)
	}
	view.Citations = len(used)
	domains := map[string]bool{}
	brandCites := 0
	catByDay := map[string]map[string]int{}
	pageByDay := map[string]map[string]int{}
	for _, c := range used {
		if c.Domain != "" {
			domains[c.Domain] = true
		}
		if isOwnCite(c, ownHost) {
			brandCites++
		}
		day := c.SampledOn.Format("2006-01-02")
		if catByDay[day] == nil {
			catByDay[day] = map[string]int{}
		}
		cat := c.Category
		if cat == "" {
			cat = "other"
		}
		catByDay[day][cat]++
		if pageByDay[day] == nil {
			pageByDay[day] = map[string]int{}
		}
		pt := c.PageType
		if pt == "" {
			pt = "other"
		}
		pageByDay[day][pt]++
	}
	view.UniqueDomains = len(domains)
	if view.Citations > 0 {
		share := float64(brandCites) / float64(view.Citations) * 100
		view.CitationShare = &share
	}
	view.CategoryKeys = stackKeys(catByDay, []string{"brand", "competitor", "reviews", "social", "reference", "ecommerce", "pr", "institutional", "other"})
	view.PageTypeKeys = stackKeys(pageByDay, []string{"homepage", "article", "comparison", "review", "howto", "forum", "product", "doc", "video", "other"})
	view.CategorySeries = stackSeries(catByDay, view.CategoryKeys)
	view.PageTypeSeries = stackSeries(pageByDay, view.PageTypeKeys)
	view.Gaps = citationGaps(used, texts, ownHost)
	view.Opportunities = measureOpportunities(byQ, used, ownHost)
	if q.QID != "" {
		head, runs := measurePrompt(in, q, groups, texts, active, used)
		view.Prompt = head
		view.RunsDetail = runs
	}
	return view
}

type dayAcc struct {
	ok, mentioned int
	brandMentions int
	compEvents    int
}

type qAcc struct {
	text          string
	group         string
	ok, mentioned int
	failed        int
	failErr       string
	days          map[string]map[string]int
	dayN          map[string]int
	compSeen      map[string]bool
}

func shortErr(err string) string {
	err = strings.TrimSpace(err)
	if err == "" {
		return ""
	}
	if strings.Contains(err, "INVALID_API_KEY") || strings.Contains(err, "401") {
		return "HTTP 401, invalid API key"
	}
	if i := strings.Index(err, "{"); i > 0 {
		err = strings.TrimSpace(err[:i])
	}
	r := []rune(err)
	if len(r) > 80 {
		return string(r[:80])
	}
	return err
}

func topReason(counts map[string]int) string {
	best, n := "", 0
	for reason, c := range counts {
		if c > n || (c == n && reason < best) {
			best, n = reason, c
		}
	}
	return best
}

func ensureQ(byQ map[string]*qAcc, qids *[]string, sm model.Sample, texts, groups map[string]string) *qAcc {
	qa := byQ[sm.QID]
	if qa != nil {
		return qa
	}
	text := sm.QuestionText
	if texts[sm.QID] != "" {
		text = texts[sm.QID]
	}
	qa = &qAcc{text: text, group: groups[sm.QID], days: map[string]map[string]int{}, dayN: map[string]int{}, compSeen: map[string]bool{}}
	byQ[sm.QID] = qa
	*qids = append(*qids, sm.QID)
	return qa
}

func keepMeasureSample(sm model.Sample, q MeasureQuery, in measureBuild) bool {
	if q.Platform != "" && sm.Platform != q.Platform {
		return false
	}
	if q.Access != "" && metrics.AccessOf(sm.SampleMode) != q.Access {
		return false
	}
	for _, item := range in.Questions {
		if item.ID == sm.QID && item.Off {
			return false
		}
	}
	if len(q.TagList) == 0 {
		return true
	}
	text, tags := measurePromptText(sm, in)
	system := model.ComputeSystemTags(text, in.Brand, in.Aliases, in.Site)
	return model.MatchTags(system, tags, q.TagList)
}

// measurePromptText returns the current prompt text and user tags for a sample.
func measurePromptText(sm model.Sample, in measureBuild) (string, []string) {
	text, tags := sm.QuestionText, []string{}
	for _, item := range in.Questions {
		if item.ID == sm.QID {
			if item.Text != "" {
				text = item.Text
			}
			tags = item.Tags
			break
		}
	}
	return text, tags
}

// measureBranded uses the single branded definition plus user tag overrides.
func measureBranded(sm model.Sample, in measureBuild) bool {
	text, tags := measurePromptText(sm, in)
	return model.EffectiveBranded(model.ComputeSystemTags(text, in.Brand, in.Aliases, in.Site), tags)
}

func measureObs(sm model.Sample, branded bool) metrics.Obs {
	rank := 0
	if sm.Rank != nil {
		rank = *sm.Rank
	}
	return metrics.Obs{
		QID: sm.QID, Platform: sm.Platform, Access: metrics.AccessOf(sm.SampleMode), Day: sm.SampledOn.Format("2006-01-02"),
		OK: sm.OK, Branded: branded, Mentioned: sm.Mentioned, Rank: rank, OwnCited: sm.OwnDomainCited,
		Competitors: sm.CompetitorsMentioned,
	}
}

func measureEngines(rows []model.Sample) []MeasureOpt {
	seenP := map[string]bool{}
	var engines []MeasureOpt
	for _, sm := range rows {
		if sm.Platform != "" && !seenP[sm.Platform] {
			seenP[sm.Platform] = true
			engines = append(engines, MeasureOpt{ID: sm.Platform, Label: LabelOf(sm.Platform)})
		}
	}
	sort.Slice(engines, func(i, j int) bool { return engines[i].Label < engines[j].Label })
	return engines
}

func measureTagOptions(questions []Question) []MeasureOpt {
	seen := map[string]bool{}
	var names []string
	for _, q := range questions {
		for _, tag := range model.SanitizeTags(q.Tags) {
			if tag == model.TagBranded || tag == model.TagUnbranded || seen[tag] {
				continue
			}
			seen[tag] = true
			names = append(names, tag)
		}
	}
	sort.Strings(names)
	out := []MeasureOpt{{ID: model.TagBranded, Label: "Branded"}, {ID: model.TagUnbranded, Label: "Unbranded"}}
	for _, name := range names {
		out = append(out, MeasureOpt{ID: name, Label: name})
	}
	return out
}

func knownQueries(qs []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, q := range qs {
		q = strings.TrimSpace(q)
		if q == "" || q == WebQueriesUnavailable || seen[q] {
			continue
		}
		seen[q] = true
		out = append(out, q)
	}
	return out
}

func wordsFrom(stats []WordStat, total int) []MeasureWord {
	var out []MeasureWord
	for _, s := range stats {
		if s.Word == "" || s.Word == WebQueriesUnavailable {
			continue
		}
		share := 0.0
		if total > 0 {
			share = float64(s.Count) / float64(total) * 100
		}
		out = append(out, MeasureWord{Word: s.Word, Count: s.Count, Share: share})
	}
	if out == nil {
		return []MeasureWord{}
	}
	return out
}

func sumCounts(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func sumStats(ss []WordStat) int {
	n := 0
	for _, s := range ss {
		n += s.Count
	}
	return n
}

func citationMatches(c model.SampleCitation, rows []model.Sample) bool {
	day := c.SampledOn.Format("2006-01-02")
	for _, sm := range rows {
		if !sm.OK {
			continue
		}
		if sm.Platform == c.Platform && sm.QID == c.QID && sm.SampledOn.Format("2006-01-02") == day {
			return true
		}
	}
	return false
}

func isOwnCite(c model.SampleCitation, ownHost string) bool {
	if c.Category == "brand" {
		return true
	}
	return ownHost != "" && hostMatch(bareHost(c.Domain), ownHost)
}

func stackKeys(byDay map[string]map[string]int, prefer []string) []string {
	seen := map[string]bool{}
	for _, counts := range byDay {
		for k := range counts {
			seen[k] = true
		}
	}
	var out []string
	for _, k := range prefer {
		if seen[k] {
			out = append(out, k)
			delete(seen, k)
		}
	}
	var rest []string
	for k := range seen {
		rest = append(rest, k)
	}
	sort.Strings(rest)
	return append(out, rest...)
}

func stackSeries(byDay map[string]map[string]int, keys []string) []map[string]any {
	var days []string
	for day := range byDay {
		days = append(days, day)
	}
	sort.Strings(days)
	var out []map[string]any
	for _, day := range days {
		counts := byDay[day]
		total := 0
		for _, n := range counts {
			total += n
		}
		row := map[string]any{"date": day}
		if total == 0 {
			out = append(out, row)
			continue
		}
		for _, k := range keys {
			row[k] = float64(counts[k]) / float64(total) * 100
		}
		out = append(out, row)
	}
	if out == nil {
		return []map[string]any{}
	}
	return out
}

func citationGaps(rows []model.SampleCitation, texts map[string]string, ownHost string) []MeasureGap {
	type acc struct {
		own     bool
		domains []string
	}
	byQ := map[string]*acc{}
	var order []string
	for _, c := range rows {
		a := byQ[c.QID]
		if a == nil {
			a = &acc{}
			byQ[c.QID] = a
			order = append(order, c.QID)
		}
		if isOwnCite(c, ownHost) {
			a.own = true
		}
		if c.Category == "competitor" && c.Domain != "" {
			a.domains = appendUnique(a.domains, c.Domain)
		}
	}
	var out []MeasureGap
	for _, qid := range order {
		a := byQ[qid]
		if a.own || len(a.domains) == 0 || qid == "" {
			continue
		}
		text := texts[qid]
		if text == "" {
			text = qid
		}
		out = append(out, MeasureGap{QID: qid, Text: text, Domains: a.domains})
	}
	if out == nil {
		return []MeasureGap{}
	}
	return out
}

func measureOpportunities(byQ map[string]*qAcc, cites []model.SampleCitation, ownHost string) []Opportunity {
	type bag struct {
		own, rivals, social, reviews []OppRef
	}
	bags := map[string]*bag{}
	for _, c := range cites {
		if c.QID == "" || c.URL == "" {
			continue
		}
		b := bags[c.QID]
		if b == nil {
			b = &bag{}
			bags[c.QID] = b
		}
		ref := OppRef{URL: c.URL, Domain: c.Domain, Title: c.Title}
		switch {
		case isOwnCite(c, ownHost):
			b.own = appendRef(b.own, ref)
		case c.Category == "social":
			b.social = appendRef(b.social, ref)
			b.rivals = appendRef(b.rivals, ref)
		case c.Category == "reviews":
			b.reviews = appendRef(b.reviews, ref)
			b.rivals = appendRef(b.rivals, ref)
		case c.Category == "competitor":
			b.rivals = appendRef(b.rivals, ref)
		}
	}
	var qids []string
	for qid, qa := range byQ {
		if qa.ok > 0 && qid != "" {
			qids = append(qids, qid)
		}
	}
	sort.Strings(qids)
	var out []Opportunity
	for _, qid := range qids {
		qa := byQ[qid]
		brandVis := float64(qa.mentioned) / float64(qa.ok) * 100
		rivalName, rivalVis := bestRival(qa)
		b := bags[qid]
		if b == nil {
			b = &bag{}
		}
		prompts := []OppRef{{QID: qid, Text: qa.text}}
		var op Opportunity
		switch {
		case len(b.social) > 0 && len(b.own) == 0:
			op = Opportunity{Category: "social", Title: "Show up in the discussion: " + trimTitle(qa.text), URLs: refURLs(b.social)}
			op.Why = "Answers for this prompt cite " + joinDomains(domainsOf(b.social)) + "."
		case len(b.reviews) > 0 && len(b.own) == 0:
			op = Opportunity{Category: "outreach", Title: "Get into the reviews: " + trimTitle(qa.text), URLs: refURLs(b.reviews)}
			op.Why = "Review sites " + joinDomains(domainsOf(b.reviews)) + " already feed this answer. The brand domain is not cited."
		case len(b.own) > 0 && brandVis < 50:
			op = Opportunity{Category: "existing-content", Title: "Refresh an owned page that is sometimes cited: " + trimTitle(qa.text), URLs: refURLs(b.own)}
			op.Why = "An owned page has been cited for this prompt. Check that it is still live before refreshing it."
		case rivalName != "" && math.Round(rivalVis) > math.Round(brandVis) && len(b.rivals) > 0:
			op = Opportunity{Category: "creation", Title: "Write a comparison: " + trimTitle(qa.text), URLs: refURLs(b.rivals)}
			op.Why = fmt.Sprintf("Visibility on this prompt is %.0f%%. %s is %.0f%%.", math.Round(brandVis), rivalName, math.Round(rivalVis))
		default:
			continue
		}
		if rivalName != "" && math.Round(rivalVis) != math.Round(brandVis) && !strings.Contains(op.Why, rivalName) {
			op.Why += fmt.Sprintf(" Visibility on this prompt is %.0f%%. %s is %.0f%%.", math.Round(brandVis), rivalName, math.Round(rivalVis))
		}
		op.QID = qid
		op.Prompts = prompts
		op.Own = b.own
		op.Rivals = b.rivals
		if len(op.Prompts) == 0 && len(op.Own) == 0 && len(op.Rivals) == 0 {
			continue
		}
		out = append(out, op)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	if len(out) > 8 {
		out = out[:8]
	}
	if out == nil {
		return []Opportunity{}
	}
	return out
}

func bestRival(qa *qAcc) (string, float64) {
	name := ""
	best := -1.0
	for rival := range qa.compSeen {
		n := 0
		for _, day := range qa.days {
			n += day[rival]
		}
		vis := float64(n) / float64(qa.ok) * 100
		if vis > best || (vis == best && (name == "" || rival < name)) {
			best = vis
			name = rival
		}
	}
	if name == "" {
		return "", 0
	}
	return name, best
}

func appendRef(ss []OppRef, v OppRef) []OppRef {
	if v.URL == "" {
		return ss
	}
	for _, s := range ss {
		if s.URL == v.URL {
			return ss
		}
	}
	if len(ss) >= 20 {
		return ss
	}
	return append(ss, v)
}

func refURLs(ss []OppRef) []string {
	var out []string
	for _, s := range ss {
		out = appendUnique(out, s.URL)
	}
	return out
}

func domainsOf(ss []OppRef) []string {
	var out []string
	for _, s := range ss {
		out = appendUnique(out, s.Domain)
	}
	return out
}

func measurePrompt(in measureBuild, q MeasureQuery, groups, texts map[string]string, active map[string]bool, cites []model.SampleCitation) (*MeasurePromptHead, []MeasureRun) {
	var rows []model.Sample
	for _, sm := range in.Rows {
		if sm.QID != q.QID {
			continue
		}
		if q.Platform != "" && sm.Platform != q.Platform {
			continue
		}
		rows = append(rows, sm)
	}
	sort.Slice(rows, func(i, j int) bool { return runAt(rows[i]) > runAt(rows[j]) })
	text := texts[q.QID]
	if text == "" {
		for _, sm := range rows {
			if sm.QuestionText != "" {
				text = sm.QuestionText
				break
			}
		}
	}
	if text == "" {
		text = q.QID
	}
	head := &MeasurePromptHead{ID: q.QID, Text: text, Group: groups[q.QID], Active: active[q.QID], Runs: len(rows)}
	ok := 0
	mentioned := 0
	days := map[string]map[string]int{}
	dayN := map[string]int{}
	compSeen := map[string]bool{}
	today := in.Now.Format("2006-01-02")
	byPlat := map[string]int{}
	for _, sm := range rows {
		if !sm.OK {
			continue
		}
		ok++
		if sm.Mentioned {
			mentioned++
		}
		day := sm.SampledOn.Format("2006-01-02")
		dayN[day]++
		if days[day] == nil {
			days[day] = map[string]int{}
		}
		if sm.Mentioned {
			days[day]["brand"]++
		}
		for _, name := range sm.CompetitorsMentioned {
			if name == "" {
				continue
			}
			days[day][name]++
			compSeen[name] = true
		}
		if day == today {
			byPlat[sm.Platform]++
		}
	}
	if ok > 0 {
		v := float64(mentioned) / float64(ok) * 100
		head.Visibility = &v
	}
	var qd []string
	for day := range dayN {
		qd = append(qd, day)
	}
	sort.Strings(qd)
	var names []string
	for name := range compSeen {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, day := range qd {
		row := map[string]any{"date": day, "brand": pct(days[day]["brand"], dayN[day])}
		for _, name := range names {
			row[name] = pct(days[day][name], dayN[day])
		}
		head.Series = append(head.Series, row)
	}
	done := 0
	if q.Platform != "" {
		done = byPlat[q.Platform]
	} else if len(byPlat) > 0 {
		done = 99
		for _, n := range byPlat {
			if n < done {
				done = n
			}
		}
	}
	if done >= 3 {
		head.Next = "Next run tomorrow"
	} else {
		head.Next = fmt.Sprintf("Done today %d/3", done)
	}
	citesBySample := map[uint64][]MeasureCite{}
	for _, c := range cites {
		if c.QID != q.QID {
			continue
		}
		citesBySample[c.SampleID] = append(citesBySample[c.SampleID], MeasureCite{
			URL: c.URL, Domain: c.Domain, Title: c.Title, Category: c.Category, PageType: c.PageType,
		})
	}
	var runs []MeasureRun
	for _, sm := range rows {
		known := knownQueries(sm.WebQueries)
		name := LabelOf(sm.Platform)
		run := MeasureRun{
			ID: sm.ID, Platform: sm.Platform, PlatformName: name, Access: accessOf(sm.SampleMode),
			At: runAt(sm), OK: sm.OK, Error: sm.Error, Queries: known, Unknown: len(known) == 0,
			Mentioned: sm.Mentioned, Answer: sm.Answer, Raw: sm.Raw, Citations: citesBySample[sm.ID],
		}
		if run.Citations == nil {
			run.Citations = []MeasureCite{}
		}
		if run.Queries == nil {
			run.Queries = []string{}
		}
		if sm.Mentioned && in.Brand != "" {
			run.Chips = append(run.Chips, MeasureChip{Name: in.Brand, Self: true})
		}
		for _, n := range sm.CompetitorsMentioned {
			if n != "" {
				run.Chips = append(run.Chips, MeasureChip{Name: n})
			}
		}
		if run.Chips == nil {
			run.Chips = []MeasureChip{}
		}
		runs = append(runs, run)
	}
	if runs == nil {
		runs = []MeasureRun{}
	}
	return head, runs
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d) * 100
}

func accessOf(mode string) string {
	if mode == "manual" {
		return "Manual"
	}
	return "API"
}

func runAt(sm model.Sample) int64 {
	if sm.CreatedAt > 0 {
		return sm.CreatedAt
	}
	return sm.SampledOn.Unix()
}
