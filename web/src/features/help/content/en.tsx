// SPDX-License-Identifier: AGPL-3.0-or-later

import { Formula, FX, Table, Tip, Warn, type HelpDoc } from "../kit";

export const en: HelpDoc = {
  groups: { start: "Getting started", read: "Reading results", act: "Taking action", connect: "Connecting", help: "Help" },
  topics: {
    start: { group: "start", label: "Start here", keys: "overview loop what is quick start first" },
    setup: { group: "start", label: "Set up a project", keys: "project create checklist keys brand competitors questions prompts schedule onboarding" },
    prompts: { group: "start", label: "Write good questions", keys: "question prompt library group branded unbranded category generic zero" },
    terms: { group: "read", label: "Key terms", keys: "glossary visibility recognition share of voice citation fan-out period run sample access" },
    numbers: { group: "read", label: "How the numbers work", keys: "formula formulas interval ci wilson newcombe small sample not measured api web failed stability share of voice" },
    pages: { group: "read", label: "Page by page", keys: "overview visibility share citations fan-out answers search reports navigation menu" },
    opportunities: { group: "act", label: "Action plan", keys: "opportunities accept dismiss priority fix first verified regressed status" },
    audit: { group: "act", label: "Site audit", keys: "readiness layers access discover understand cite blocked severity evidence issues export" },
    manual: { group: "act", label: "Manual sampling", keys: "sheet import export chatgpt web google ai overviews baidu no api" },
    providers: { group: "connect", label: "Model providers", keys: "api key engine relay endpoint model web search cost" },
    google: { group: "connect", label: "Google Search & GA4", keys: "oauth client service account search console ga4 property sync redirect" },
    access: { group: "connect", label: "Users and access", keys: "user users role admin member permission view edit password share invite team login" },
    schedule: { group: "connect", label: "Schedule & runs", keys: "period every day runs per day tokens cost retry job interrupted" },
    troubleshooting: { group: "help", label: "Troubleshooting", keys: "problem error 0% not measured failed stuck reconnect empty spa blocked" },
    limits: { group: "help", label: "What this is not", keys: "limits promise guarantee keyword backlink" },
  },
  body: ({ page, topic, n }) => ({
    start: (
      <>
        <h2>Start here</h2>
        <p>craftsail-growth answers one question: <strong>when people ask AI engines about your category, do the answers mention and cite you, and is that getting better?</strong></p>
        <p>It works as a loop that repeats every period:</p>
        <ol>
          <li><strong>Measure.</strong> Ask each engine your questions several times and record every answer: who is mentioned, in what order, and which sources are cited.</li>
          <li><strong>Diagnose.</strong> Crawl and audit your site in four layers: can engines fetch it, find it, understand it, and quote it.</li>
          <li><strong>Act.</strong> Audit findings, citation gaps, search data and drops become one ranked action plan. You accept the items you will do.</li>
          <li><strong>Verify.</strong> The next period checks each accepted action against its rule and marks it verified, or regressed if it later slips.</li>
        </ol>
        <p>The sidebar follows the same loop: <strong>{n("nav.sections.measure")}</strong> measures, <strong>{n("nav.sections.improve")}</strong> diagnoses and acts, <strong>{n("nav.sections.project")}</strong> holds this project's setup and <strong>{n("nav.sections.workspace")}</strong> what all projects share.</p>
        <h3>Your first 15 minutes</h3>
        <ol>
          <li>Add at least one engine key under {page("settings/providers", "nav.providers")}. Do this <strong>before</strong> creating a project; the key is also used to draft your brand facts and questions.</li>
          <li>Create a project under {page("settings/projects", "nav.projects")} with “Run the first period now” on.</li>
          <li>While it runs, read {topic("prompts", "Write good questions")}, then check the drafted list under {page("settings/questions", "nav.questions")}.</li>
          <li>When the run finishes, open {page("overview", "nav.overview")}, then the {page("opportunities", "nav.actionPlan")}.</li>
        </ol>
        <p>Admins can invite teammates or clients afterwards under {page("settings/users", "nav.users")}; see {topic("access", "Users and access")}.</p>
        <Tip>The first period only tells you where you stand. The value comes from repeating it: turn on a schedule under {page("settings/schedule", "nav.schedule")}.</Tip>
      </>
    ),
    setup: (
      <>
        <h2>Set up a project</h2>
        <p>A project is one brand and its website. Everything else, from questions to reports, belongs to a project. Switch projects from the project name at the start of the breadcrumb, at the top of every page.</p>
        <Table head={["Step", "Where", "Why it matters"]} rows={[
          ["1. Connect engines", page("settings/providers", "nav.providers"), "Without a key nothing is sampled, and questions fall back to generic templates."],
          ["2. Create the project", page("settings/projects", "nav.projects"), "Give the site URL. Without a website, tick “No owned website” and give the brand name."],
          ["3. Check brand facts", page("settings/brand", "nav.brand"), "A short form: name and other names, one-line definition, category, audience, key numbers with sources, fit and limits. Drafted from the site; empty fields were not found there. Generated llms.txt and JSON-LD come from here."],
          ["4. Check competitors", page("settings/competitors", "nav.competitors"), "Three to six real rivals with aliases. Share of voice and rank are counted against this list."],
          ["5. Fix the questions", page("settings/questions", "nav.questions"), <>The most important step. See {topic("prompts", "Write good questions")}.</>],
          ["6. Connect Google (optional)", page("settings/google", "nav.google"), "Adds search clicks, queries and sessions, and search items in the action plan."],
          ["7. Schedule", page("settings/schedule", "nav.schedule"), "Run a period every week or day so trends and verification work."],
          ["8. Invite people (optional)", page("settings/users", "nav.users"), "Add teammates or clients and share each project for viewing or editing."],
        ]} />
        <Warn>Created the project before adding a key? Its questions are generic templates. Add a key, then open {page("settings/questions", "nav.questions")} and click <strong>Redraft with AI</strong>. It replaces the questions and competitors with a draft written from your site, so review both afterwards.</Warn>
      </>
    ),
    prompts: (
      <>
        <h2>Write good questions</h2>
        <p>Questions, also called prompts, are what the tool asks each engine. They decide what is measured; a weak list makes every other number meaningless. Edit them under {page("settings/questions", "nav.questions")}.</p>
        <h3>Name the category the way a buyer would</h3>
        <Table head={["Weak", "Better"]} rows={[
          ["What are the best tools in this category?", "What are the best command-line tools for converting Word documents to PDF?"],
          ["What should a small team use?", "Which document automation API should a small SaaS team use?"],
          ["Where should I start on day one?", "How do I generate Excel reports from Python without Microsoft Office?"],
        ]} />
        <p>A question that does not name the category makes engines ask a clarifying question instead of recommending anything. Visibility then stays at 0% whatever you do to the site.</p>
        <h3>Groups</h3>
        <p>Each question belongs to a group. Recommendation, Comparison, Alternatives and Pricing are <strong>buyer</strong> questions. Risks and Use case explain the category. <strong>Brand check</strong> names your brand on purpose.</p>
        <h3>Branded and unbranded</h3>
        <p>A question that contains your brand name, an alias or your domain is <strong>branded</strong>; the System column shows this. Branded questions mention you almost by definition, so they are reported as <strong>recognition</strong> and never count toward visibility. Aim for mostly unbranded buyer questions.</p>
        <h3>Practical rules</h3>
        <ul>
          <li>Write questions in your buyers' language. Every enabled question is asked on every connected engine.</li>
          <li>Start with 10 to 20 questions. Each one costs one call per engine per run.</li>
          <li>Use your own tags (for example a product line) to filter the AI visibility pages.</li>
          <li>Turn a question off instead of deleting it to keep its history.</li>
          <li><strong>Redraft with AI</strong> asks a connected model for a fresh list written from your site. It replaces the current questions and competitors.</li>
        </ul>
      </>
    ),
    terms: (
      <>
        <h2>Key terms</h2>
        <Table head={["Term", "Meaning"]} rows={[
          ["Engine / model provider", "An AI product the tool asks, such as DeepSeek, OpenAI or Perplexity."],
          ["Question (prompt)", "One entry in your list, asked on every engine."],
          ["Run / answer", "One engine answering one question once. Engines vary, so each question is asked several times per day."],
          ["Period", "One full pass: crawl, audit, sample, sync Google, verify, update the action plan, write the report."],
          ["Access (API / Web)", "Answers from an engine's API and answers collected by hand in its web or app product are kept apart; they cite different sources."],
          ["Visibility", "Share of successful answers to unbranded questions that mention your brand."],
          ["Recognition", "The same rate on branded questions. It shows whether the engine knows you, not whether it recommends you."],
          ["Share of voice", "Your mentions divided by your mentions plus competitor mentions."],
          ["Citation", "A source URL an engine attached to its answer."],
          ["Query fan-out", "The web searches an engine ran to answer a question, compared with the question's own words."],
          ["Suggestion", "An item in the action plan from the audit, citations, search data or a metric drop."],
          ["Action", "A suggestion you accepted. It has a baseline and a rule that checks whether it worked."],
          ["Readiness layer", "Access, Discover, Understand, Cite: the four things a page needs, in fix order."],
        ]} />
      </>
    ),
    numbers: (
      <>
        <h2>How the numbers work</h2>
        <p><strong>Visibility</strong> is the share of successful answers to unbranded questions that mention the brand. Questions that name the brand are reported separately as <strong>recognition</strong>.</p>
        <p>Answers change from run to run, so every rate is an estimate. Next to it you see a <strong>95% interval</strong> (for example “CI 0–22%”) and the number of answers (n). Read the interval, not just the number: 0% out of 14 answers means “probably below 22%”, not “never”.</p>
        <ul>
          <li><strong>Small sample</strong>: fewer than 30 answers. Treat the rate as a rough signal.</li>
          <li><strong>Not measured</strong> or <strong>—</strong>: nothing to count yet, for example no competitor was named. It is never shown as 0%.</li>
          <li><strong>Failed runs</strong> (timeouts, quota errors) are left out of rates and counted separately. Retry them from {page("settings/schedule", "nav.schedule")}.</li>
          <li><strong>API and Web</strong> answers are never added together. When both exist, a switch appears in the filter bar.</li>
          <li><strong>Up or down</strong> between periods is only called when the change is larger than the noise (a 95% Newcombe interval that excludes zero).</li>
        </ul>
        <h3>Formulas</h3>
        <p><strong>Visibility.</strong> Of the answers that came back, how many name you. Questions that contain your name are left out, because they name you almost by definition.</p>
        <Formula>{FX.visibility}</Formula>
        <p><strong>Recognition.</strong> The same count on the questions that do name you. It shows whether the engine knows you, not whether it recommends you.</p>
        <Formula>{FX.recognition}</Formula>
        <p><strong>Top 1 and top 3.</strong> Being named first matters more than being named last, so the order of first appearance is counted too.</p>
        <Formula>{FX.top}</Formula>
        <p><strong>Share of voice.</strong> Your part of all brand mentions, yours plus your competitors'. With no mentions at all it is not measured.</p>
        <Formula>{FX.sov}</Formula>
        <p><strong>Answers citing your domain</strong> and <strong>citation share.</strong> Citation counts are never compared across engines: some attach many more sources per answer than others.</p>
        <Formula>{FX.ownCite}</Formula>
        <h3>The interval next to every rate</h3>
        <p>An engine answers differently each time, so a rate is an estimate. The 95% Wilson interval shows the range the true rate is likely in; it stays sensible with few answers, where a simple ± would go below 0% or above 100%.</p>
        <Formula>{FX.wilson}</Formula>
        <p>Example:</p>
        <Formula>{FX.wilsonExample}</Formula>
        <h3>Up, down or within noise</h3>
        <p>Two periods are compared with an interval on the difference, not by eye. Only a difference whose whole interval sits on one side of zero is called a change.</p>
        <Formula>{FX.change}</Formula>
        <Formula>{FX.changeExample}</Formula>
        <h3>Citation stability</h3>
        <p>How much the set of cited sites changes from day to day. A low score means the sources are still open to newcomers; a high score means engines keep citing the same sites. It is only used to rank suggestions.</p>
        <Formula>{FX.stability}</Formula>
      </>
    ),
    pages: (
      <>
        <h2>Page by page</h2>
        <p>Pages with several views, such as {n("nav.audit")} and {n("nav.search")}, show them as tabs under the page title.</p>
        <Table head={["Page", "Answers the question", "What to do next"]} rows={[
          [page("overview", "nav.overview"), "Where do we stand? Visibility, share of voice, owned citations, readiness, top actions.", "Open the weakest area."],
          [page("ai/visibility", "nav.visibility"), "Which questions mention us, on which engine, over time.", "Open a question to read its answers."],
          [page("ai/share-of-voice", "nav.sov"), "Who gets named instead of us.", <>Add missing rivals under {n("nav.competitors")}.</>],
          [page("ai/citations", "nav.citations"), "Which sources engines rely on, by source type and page type; which cite rivals but not us.", <>Target those sources from the {n("nav.actionPlan")}.</>],
          [page("ai/fan-out", "nav.fanout"), "What engines actually search for. Only engines with web search report this.", "Use added words in your pages and questions."],
          [page("ai/answers", "nav.answers"), "Every raw answer, with citations and searches.", "Correct misreadings; import manual samples."],
          [page("opportunities", "nav.actionPlan"), "What to do next, ranked, and whether it worked.", "Accept, start, mark done."],
          [page("audit", "nav.audit"), "Can engines fetch, find, understand and quote the site.", "Fix the first failing layer."],
          [page("search", "nav.search"), "Google clicks, impressions, sessions, queries and landing pages.", "Compare with AI visibility."],
          [page("reports", "nav.reports"), "One shareable page per period.", "Download HTML or Markdown."],
        ]} />
        <p>When a page title shows <strong>{n("access.viewOnly")}</strong>, you have view access to that project: every button that changes data is hidden. See {topic("access", "Users and access")}.</p>
        <Tip>Filters (models, tags, time range) sit at the top of each AI visibility page. They are kept in the page address, so you can bookmark or share a filtered view.</Tip>
      </>
    ),
    opportunities: (
      <>
        <h2>Action plan</h2>
        <p>The {page("opportunities", "nav.actionPlan")} is the single to-do list. Suggestions come from four sources and are grouped into <strong>Fix first</strong>, <strong>Worth doing</strong> and <strong>Nice to have</strong>:</p>
        <Table head={["Source", "Example", "Group"]} rows={[
          ["Site audit", "WAF blocks AI crawlers; no structured data", "Critical → Fix first, warning → Worth doing, info → Nice to have. Observational rules are never Fix first."],
          ["AI citations", "Rivals are cited for a question and you are not", "Worth doing, or Nice to have when the source is hard to enter"],
          ["Search", "A query sits just outside the top three; low click-through", "Nice to have"],
          ["Metric change", "Visibility dropped beyond the noise", "Fix first"],
        ]} />
        <h3>Tabs and lifecycle</h3>
        <ol>
          <li><strong>Suggested</strong>: accept an item, or dismiss it if it does not apply. Dismissed items can be restored from their tab.</li>
          <li><strong>In progress</strong>: accepted items are recorded with a baseline, for example “3 pages affected”. Click Start, then Mark done.</li>
          <li><strong>Verified</strong> when the next period's check passes. A verified action that later fails becomes <strong>Regressed</strong> and returns to In progress.</li>
        </ol>
        <h3>How “done” is checked</h3>
        <ul>
          <li>Audit actions pass when the issue is gone from the latest audit.</li>
          <li>Citation actions pass when visibility on that question rises beyond the noise.</li>
          <li>A visibility drop passes when visibility is back within the noise of the baseline.</li>
          <li>Search actions are checked by you: mark them done when finished.</li>
        </ul>
        <h3>When each suggestion appears</h3>
        <p><strong>AI citations.</strong> A question gets a citation suggestion when engines cite competitor sites for it and never your own:</p>
        <Formula>{FX.gap}</Formula>
        <p><strong>Search.</strong> From Google Search Console, over 28 days ending three days ago (Google's data arrives late), compared with the 28 days before:</p>
        <Formula>{FX.striking}</Formula>
        <p><strong>Metric change.</strong> Visibility over the last 30 days compared with the 30 days before, with the “Up, down or within noise” rule from {topic("numbers", "How the numbers work")}.</p>
        <h3>How an action is checked</h3>
        <Formula>{FX.verify}</Formula>
        <p>Branded search queries (your own name) are left out: people searching for you already found you.</p>
        <Tip>Each item shows “How to fix” and “Done when”. The CLI has the same list: <code>craftsail-growth opportunities --slug &lt;project&gt;</code>.</Tip>
      </>
    ),
    audit: (
      <>
        <h2>Site audit</h2>
        <p>The audit runs 47 rules over the crawled pages and groups them in four layers, in the order to fix them:</p>
        <Table head={["Layer", "Question", "Typical findings"]} rows={[
          ["Access", "Can crawlers fetch the content?", "robots.txt blocks AI bots, a CDN rejects AI user agents, noindex, pages empty without JavaScript"],
          ["Discover", "Can they find every URL?", "No sitemap, sitemap not in robots.txt, no llms.txt, broken links"],
          ["Understand", "Can they tell what each page is?", "No structured data, schema that contradicts the page, long titles"],
          ["Cite", "Is there something worth quoting?", "No definition sentence, few numbers, no comparison, no quotable passage"],
        ]} />
        <p>When a layer fails, findings in later layers are marked <strong>blocked</strong>: fixing them now would not change what engines see.</p>
        <p>Each rule shows its <strong>evidence</strong>: standard, vendor docs, experiment, observational or rule of thumb. Observational rules and rules of thumb are advice and are never critical.</p>
        <h3>How a layer's status is decided</h3>
        <Formula>{FX.layers}</Formula>
        <p>The readiness number on {n("nav.overview")} (for example “1 / 4”) counts the layers that pass with no findings at all; a layer with only warnings does not count.</p>
        <h3>How a page is scored</h3>
        <p>The {n("nav.tabs.pages")} tab gives each page a score out of 100 from six parts. The thresholds describe pages that engines did cite in published datasets; they are signals, not a guarantee. Words are counted in the HTML as fetched, without running JavaScript.</p>
        <Formula>{FX.pageScore}</Formula>
        <Formula>{FX.pageParts}</Formula>
        <ul>
          <li>The <strong>{n("nav.tabs.readiness")}</strong> tab shows the layers; <strong>{n("nav.tabs.issues")}</strong> lists every rule with the pages affected; <strong>{n("nav.tabs.pages")}</strong> scores each page.</li>
          <li><strong>Export for AI</strong> downloads the audit as Markdown to hand to a coding assistant.</li>
          <li>Crawl again after changing the site; the audit only sees the last crawl.</li>
        </ul>
      </>
    ),
    manual: (
      <>
        <h2>Manual sampling</h2>
        <p>Some of the products people actually use have no API: ChatGPT web with search, Claude web, Google AI Overviews, Baidu AI Search, Doubao app, Metaso, Nano AI. You can still measure them by hand.</p>
        <ol>
          <li>On {page("ai/answers", "nav.answers")}, click <strong>Export sampling sheet</strong>.</li>
          <li>For each question, open the product in a private window, start a new conversation, ask it and paste the full answer into the sheet, even when you are not mentioned.</li>
          <li>Paste the filled sheet back and click <strong>Import filled sheet</strong>.</li>
        </ol>
        <Warn>Do not sample from your everyday account: engines personalize answers. Use a private window or a profile used only for sampling. Manual answers are counted as Web and never mixed with API answers.</Warn>
        <p>The CLI does the same: <code>craftsail-growth sample-sheet --slug &lt;project&gt; --intent buyer --limit 20</code> and <code>craftsail-growth sample-import</code>.</p>
      </>
    ),
    providers: (
      <>
        <h2>Model providers</h2>
        <p>{page("settings/providers", "nav.providers")} lists every engine with its status. Pick one, paste the key, <strong>Test connection</strong>, then save. Engines without a key are skipped.</p>
        <Table head={["Kind", "Engines", "What it measures"]} rows={[
          ["Searches the web", "Perplexity; Doubao with the Ark content plugin enabled", "What the engine finds and cites today. Closest to the consumer product."],
          ["Answers from memory", "DeepSeek, Kimi, GLM, MiniMax, OpenAI, Claude, Gemini, Grok (as configured)", "What the model already knows about you. Changes slowly, with new model versions."],
        ]} />
        <ul>
          <li><strong>Advanced</strong> lets you change the model or point to a relay endpoint. A relay needs a model name it recognizes.</li>
          <li>Keys are stored in <code>config/default.toml</code> on this server and never leave it except to call that provider.</li>
          <li>{n("nav.schedule")} shows estimated and reported tokens for every run.</li>
        </ul>
        <h3>What sampling costs</h3>
        <p>Every enabled question is asked on every engine that has a key, as many times a day as you set. The token figure is an estimate for planning; the engine's own report is shown next to it when the engine returns one.</p>
        <Formula>{FX.calls}</Formula>
        <Formula>{FX.callsExample}</Formula>
      </>
    ),
    google: (
      <>
        <h2>Google Search &amp; GA4</h2>
        <p>Optional. It adds Search Console clicks, impressions and queries and GA4 sessions to {n("nav.search")}, and search items to the action plan. AI visibility and the audit work without it.</p>
        <h3>Step 1: connect a Google account</h3>
        <p>This server needs its own <strong>OAuth client</strong> once. It is the app's identity at Google: when you click Connect Google, Google shows the consent page for this app and sends the result back to the redirect URI. A self-hosted install cannot share one, because each server has its own address.</p>
        <ol>
          <li>Enable the Search Console API, the Google Analytics Data API and the Analytics Admin API in Google Cloud.</li>
          <li>Create an OAuth client of type Web application and add the redirect URI shown on the page, exactly.</li>
          <li>Paste the client ID and secret, save, and click Connect Google.</li>
        </ol>
        <p>Prefer no browser sign-in? Use a <strong>service account</strong> under Advanced instead, and add its email to the Search Console property and as a Viewer on the GA4 property.</p>
        <h3>Step 2: pick properties</h3>
        <p>Choose the Search Console property and the GA4 property for this project, then <strong>Sync now</strong>. Each card shows its status and the last imported day.</p>
        <h3>How the search numbers are computed</h3>
        <p>Totals come from Google's daily totals. Query and page rows leave out anonymized queries, so they always add up to less; the difference is shown, never filled in.</p>
        <Formula>{FX.search}</Formula>
        <Warn>While the consent screen is in Testing, Google expires the sign-in after about 7 days and the status turns to “Needs sign-in”. Publish the app, or reconnect when that happens. Imported data is kept.</Warn>
      </>
    ),
    schedule: (
      <>
        <h2>Schedule &amp; runs</h2>
        <p>A <strong>period</strong> runs seven steps: crawl, audit, sample, sync Google, verify actions, update the action plan, write the report. If there are no questions yet, it drafts brand facts and questions first.</p>
        <ul>
          <li><strong>Run a period every</strong> day, week, two weeks or 30 days. The server checks every 30 minutes; it must be running for schedules to fire.</li>
          <li><strong>Runs per question and engine per day</strong> (1–10, default 3). More runs give tighter intervals and cost more.</li>
          <li>The buttons run one step on its own: Sample now, Sync Google, Crawl site, Audit site, Verify actions.</li>
          <li><strong>Sampling runs</strong> lists every batch with planned, succeeded and failed calls and tokens. <strong>Retry failures</strong> re-asks only the failed calls.</li>
          <li>One job runs per project at a time. A job shown as <em>interrupted</em> was stopped by a server restart; start it again.</li>
        </ul>
      </>
    ),
    access: (
      <>
        <h2>Users and access</h2>
        <p>Admins add people under {page("settings/users", "nav.users")} and choose, for each project, whether that person can view it, edit it, or not see it at all. There are two roles:</p>
        <ul>
          <li><strong>Admin</strong>: every project, plus the workspace: projects, model providers, Google and users.</li>
          <li><strong>Member</strong>: only the projects shared with them, each with <strong>{n("users.access.view")}</strong> or <strong>{n("users.access.edit")}</strong>.</li>
        </ul>
        <Table head={["", n("users.access.view"), n("users.access.edit"), n("users.roles.admin")]} rows={[
          ["Read every page; download reports and exports", "✓", "✓", "✓"],
          ["Change brand, competitors, questions and schedule", "", "✓", "✓"],
          ["Run crawls, audits and sampling; accept and finish actions; correct answers", "", "✓", "✓"],
          ["Create projects; model providers; Google; users", "", "", "✓"],
        ]} />
        <h3>Invite someone</h3>
        <ol>
          <li>Open {page("settings/users", "nav.users")} and fill in <strong>{n("users.add")}</strong>: a username (an e-mail address works) and a password of at least 12 characters.</li>
          <li>Select the new user and set each project to {n("users.access.none")}, {n("users.access.view")} or {n("users.access.edit")}. Changes apply immediately.</li>
          <li>Send them the address of this dashboard, the username and the password. They can change the password from the account menu.</li>
        </ol>
        <h3>Good to know</h3>
        <ul>
          <li>A project that is not shared with someone does not appear for them at all, not even its name.</li>
          <li>Pages of a project you can only view show <strong>{n("access.viewOnly")}</strong> next to the title and hide every button that changes data.</li>
          <li>Change your own password under <strong>{n("access.changePassword")}</strong> in the account menu at the bottom of the sidebar. It signs you out on your other devices.</li>
          <li>Resetting someone's password, disabling them or deleting them signs them out everywhere at once. Deleting a user keeps all project data.</li>
          <li>There is always at least one enabled admin; the last one cannot be demoted, disabled or deleted.</li>
          <li>Upgrading from the single login: that account becomes the first admin, and everyone signs in once more.</li>
          <li>Locked out? On the server run <code>craftsail-growth user passwd &lt;name&gt;</code> and type the new password (it shows as you type; to hide it, pipe it in: <code>printf '%s\n' "$PASS" | craftsail-growth user passwd &lt;name&gt;</code>). <code>craftsail-growth user add &lt;name&gt; --admin</code> adds another admin.</li>
          <li>If an admin changes your access while you are signed in, reload the page to see the new project list.</li>
          <li>The API token in the server configuration acts as an admin for scripts and the CLI.</li>
        </ul>
      </>
    ),
    troubleshooting: (
      <>
        <h2>Troubleshooting</h2>
        <Table head={["You see", "Likely cause", "What to do"]} rows={[
          ["Visibility 0% on every question", "Questions do not name the category, so engines ask what you mean. Common when the project was created without an engine key.", <>Rewrite them ({topic("prompts", "guide")}), or connect a model and click Redraft with AI on {n("nav.questions")}.</>],
          ["“Not measured” or —", "Nothing to count yet, for example no competitors named.", "Add competitors; run more samples."],
          ["“small sample”", "Fewer than 30 answers in the window.", "Raise runs per day or widen the time range."],
          ["Many failed runs", "Wrong key, quota, or a relay that does not know the model.", "Test the provider; check the model name under Advanced; Retry failures."],
          ["No answers at all", "No engine connected, or sampling never ran.", "Connect a provider, then Sample now."],
          ["Access layer failing: AI_UA_BLOCKED", "A CDN or WAF rejects AI crawler user agents while browsers work.", "Allow GPTBot, ClaudeBot, PerplexityBot and similar in the CDN bot settings."],
          ["Pages with almost no words", "The site renders in JavaScript; crawlers see an empty shell.", "Server-side render or pre-render key pages."],
          ["Google “Needs sign-in”", "The refresh token expired (Testing mode) or was revoked.", <>Reconnect on {n("nav.google")}.</>],
          ["Search totals ≠ sum of queries", "Google hides anonymized queries from row reports.", "Expected. Totals come from Google's daily totals."],
          ["A job stays “running”", "It is still working; periods with many questions take a while.", <>Watch {n("nav.schedule")}; restart the server only if the log stopped.</>],
          ["A project is missing from my list", "It is not shared with you.", <>Ask an admin to give you View or Edit on it under {n("nav.users")}.</>],
          [<>No Save or Run buttons; the title says {n("access.viewOnly")}</>, "You have view access to this project.", "Ask an admin for Edit."],
          ["Signed out after an admin changed something", "Your password was reset, or your account was disabled.", "Sign in with the new password, or ask the admin."],
          ["Messages or a report in English", "Server messages, job logs and report content are English in every language.", "Expected for now."],
        ]} />
      </>
    ),
    limits: (
      <>
        <h2>What this is not</h2>
        <ul>
          <li><strong>Not a promise of citations.</strong> Nobody can guarantee an engine will cite a page. The tool measures, suggests and verifies.</li>
          <li><strong>Not a ranking.</strong> Rates are estimates from repeated samples, shown with their uncertainty.</li>
          <li><strong>Not keyword research or backlinks.</strong> There is no paid SEO data source.</li>
          <li><strong>Not a publisher.</strong> It drafts fix snippets such as llms.txt and JSON-LD for you to review; it does not change your site.</li>
          <li><strong>Self-hosted, with simple roles.</strong> Admins and members with View or Edit per project; no teams or single sign-on. Your keys and data stay on this server.</li>
        </ul>
      </>
    ),
  }),
  buttons: ({ n, page }) => ({
    indexing: <>{n("tips.indexing")}</>,
    serve: <>Use it after setting up a project or changing a lot at once. Progress and the log are under Jobs on {page("settings/schedule", "nav.schedule")}; only one job runs per project at a time. With a schedule on, this runs by itself.</>,
    sample: <>One round only. For tighter intervals, raise “Runs per question and engine per day” and let the schedule run. Each call costs tokens; see the cost formula under {n("nav.providers")}.</>,
    syncGoogle: <>Needs Google connected and properties chosen on {page("settings/google", "nav.google")}. Google's data lags about three days.</>,
    crawl: <>Only fetches pages; it does not change the audit. Run the audit afterwards, or use {n("audit.actions.crawl")} on {page("audit", "nav.audit")}.</>,
    audit: <>Fast, because nothing is fetched. If you changed the site, crawl first or the audit still sees the old pages.</>,
    verify: <>An action passes when its rule is satisfied and moves to Verified; a verified action that fails later becomes Regressed. See the check rules under {n("nav.actionPlan")}.</>,
    scheduleSave: <>The server checks every 30 minutes whether a period is due, so it must be running. Off stops automatic periods; the buttons still work.</>,
    retry: <>Shown only for runs with failures. Fix the cause first (key, quota, model name), or the same calls fail again.</>,
    stop: <>Answers and pages saved before the stop are kept. A job marked interrupted was stopped by a server restart and can be started again.</>,
    auditCrawl: <>Takes longer than {n("audit.actions.again")} because every page is fetched again.</>,
    auditAgain: <>Useful after a rule change or to refresh results; it cannot see changes on the site until the next crawl.</>,
    exportAI: <>The file lists each failing rule with its evidence, the pages affected and how to fix it. Paste it into a coding assistant together with your site's code.</>,
    exportSheet: <>Open each product in a private window, ask the question in a new conversation and paste the full answer, even when you are not mentioned.</>,
    importSheet: <>Imported answers are counted as Web and never mixed with API answers. Importing the same sheet again on the same day updates those answers instead of adding copies.</>,
    correct: <>Use it when the tool missed a mention (for example a nickname) or flagged one wrongly. Add the nickname as an alias under {n("nav.brand")} so it is found next time.</>,
    addQuestion: <>Name the category the way a buyer would; see {n("nav.questions")} guidance in “Write good questions”. The group decides how the question is counted.</>,
    saveQuestions: <>Turn a question off instead of deleting it to keep its history. Questions that name your brand are marked branded and count as recognition.</>,
    redraft: <>Runs in the background and can take a few minutes; you can leave the page. It cannot run while another job runs on the project. Review both lists afterwards.</>,
    createProject: <>Add a model key first, or the questions fall back to generic templates. Without a website, tick {n("projects.noSite")} and give the brand name.</>,
    saveBrand: <>Empty fields were not found on the site; fill in what you know. Only admins can change the brand name, because it is also the project name.</>,
    addCompetitor: <>Add three to six real rivals with the names people use for them. Unconfirmed competitors found in answers can be confirmed here.</>,
    saveCompetitors: <>Past answers are not re-counted; changes apply from the next sampling.</>,
    accept: <>The baseline (for example “3 pages affected”) is recorded at this moment, so the check compares against where you started.</>,
    dismiss: <>Dismissed items move to the Dismissed tab and are not suggested again while dismissed.</>,
    progress: <>Start marks the action In progress; Mark done hands it to the next period's check. Search actions are checked by you.</>,
    restore: <>It comes back as an accepted action under In progress, ready to Start.</>,
    buildReport: <>Uses the same numbers and rules as the dashboard. Report text is in English whatever language the dashboard shows.</>,
    downloadReport: <>HTML opens in any browser and can be mailed as is; Markdown pastes into documents and wikis.</>,
    searchSync: <>Totals come from Google's daily totals, rows from query reports; the two differ because Google hides anonymized queries.</>,
    saveKeyword: <>Saving marks the query for this project so you can find it again among many rows.</>,
    googleConnect: <>Needs this server's OAuth client once (under Advanced). Google shows its consent page and sends you back here.</>,
    googleSyncNow: <>Each card shows the last imported day. The first import can take a while for large properties.</>,
    saveProperties: <>Pick the Search Console property and GA4 property that match this project's site; each project has its own.</>,
    googleDisconnect: <>Stops future imports for every project on this server. Reconnect with {n("google.connect")}.</>,
    testConnection: <>A failure shows the provider's message: a wrong key, no quota, or a model name the endpoint does not know.</>,
    saveConnect: <>Keys stay on this server and are sent only to that provider.</>,
    advanced: <>A relay endpoint must accept the model name you give it. Leave empty to use the provider's default.</>,
    addUser: <>Give the person the password yourself; they can change it from the account menu. New users see no project until you give access.</>,
    userAccess: <>Admins always have Edit on every project, so this table is only shown for members.</>,
    setPassword: <>At least 12 characters. The user must sign in again on every device.</>,
    disableUser: <>You cannot disable yourself or the last enabled admin.</>,
    deleteUser: <>Cannot be undone. You cannot delete yourself or the last enabled admin.</>,
    changePassword: <>Enter the current password first. This browser stays signed in.</>,
  }),
};
