// SPDX-License-Identifier: AGPL-3.0-or-later

import { usageBodies, usageButtons } from "../usage";
import { Formula, FX, Table, Warn, type HelpDoc } from "../kit";

export const en: HelpDoc = {
  groups: { start: "Getting started", read: "Reading results", act: "Taking action", connect: "Connecting", help: "Help" },
  body: (k) => { const { page, n } = k; return ({
    ...usageBodies(k),
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
  }); },
  buttons: (k) => { const { n, page } = k; return ({
    indexing: <>{n("tips.indexing")}</>,
    serve: <>Use it after setting up a project or changing a lot at once. Progress and the log are under Jobs on {page("settings/schedule", "nav.schedule")}; only one job runs per project at a time. With a schedule on, this runs by itself.</>,
    sample: <>One round only. For tighter intervals, raise “Runs per question and engine per day” and let the schedule run. Each call costs tokens; see the cost formula under {n("nav.providers")}.</>,
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
    saveBrand: <>Empty fields were not found on the site; fill in what you know. Only admins can change the brand name, because it is also the project name.</>,
    addCompetitor: <>Add three to six real rivals with the names people use for them. Unconfirmed competitors found in answers can be confirmed here.</>,
    saveCompetitors: <>Past answers are not re-counted; changes apply from the next sampling.</>,
    accept: <>The baseline (for example “3 pages affected”) is recorded at this moment, so the check compares against where you started.</>,
    dismiss: <>Dismissed items move to the Dismissed tab and are not suggested again while dismissed.</>,
    restore: <>It comes back as an accepted action under In progress, ready to Start.</>,
    downloadReport: <>HTML opens in any browser and can be mailed as is; Markdown pastes into documents and wikis.</>,
    saveKeyword: <>Saving marks the query for this project so you can find it again among many rows.</>,
    googleConnect: <>Needs this server's OAuth client once (under Advanced). Google shows its consent page and sends you back here.</>,
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
    ...usageButtons(k),
  }); },
};
