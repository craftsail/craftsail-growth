// SPDX-License-Identifier: AGPL-3.0-or-later

type Envelope<T> = { code: number; msg: string; data: T };

export type Project = {
  id: number;
  access?: Access;
  slug: string;
  name: string;
  site: string;
  market?: string;
  no_site: boolean;
  platforms?: string[];
  targets?: { mention_rate?: number };
  gsc_site?: string;
  ga4_property?: string;
  brand?: { aliases?: string[] };
  monitor_every_days?: number | null;
  monitor_next_run?: string | null;
  monitor_runs_per_day?: number;
};

export const SIGNED_OUT = "craftsail:signed-out";

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
  });
  const env = (await res.json()) as Envelope<T>;
  // A session ended by the server (password reset, disabled, deleted) sends
  // the app back to sign-in. A wrong password on /api/login is not that.
  if (res.status === 401 && path !== "/api/login") {
    window.dispatchEvent(new Event(SIGNED_OUT));
  }
  if (!res.ok || env.code !== 0) {
    throw new Error(env.msg || res.statusText);
  }
  return env.data;
}

export type Role = "admin" | "member";
export type Access = "view" | "edit";
export type SessionUser = { id: number; username: string; role: Role; must_change_password?: boolean };

export function getSession() {
  return request<{ ok: boolean; user?: SessionUser; setup?: boolean }>("/api/session");
}

export type UserRow = {
  id: number;
  username: string;
  role: Role;
  disabled: boolean;
  last_login_at: number | null;
  projects: { slug: string; name: string; access: Access }[];
};

export function listUsers() {
  return request<{ items: UserRow[] }>("/api/users");
}
export function createUser(username: string, password: string, role: Role) {
  return request<{ id: number }>("/api/users", { method: "POST", body: JSON.stringify({ username, password, role }) });
}
export function patchUser(id: number, patch: { role?: Role; disabled?: boolean }) {
  return request<{ ok: boolean }>(`/api/users/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
}
export function deleteUser(id: number) {
  return request<{ ok: boolean }>(`/api/users/${id}`, { method: "DELETE" });
}
export function resetUserPassword(id: number, password: string) {
  return request<{ ok: boolean }>(`/api/users/${id}/password`, { method: "POST", body: JSON.stringify({ password }) });
}
export function putUserAccess(id: number, items: { slug: string; access: Access | "" }[]) {
  return request<{ ok: boolean }>(`/api/users/${id}/access`, { method: "PUT", body: JSON.stringify({ items }) });
}
export function changeOwnPassword(current: string, password: string) {
  return request<{ ok: boolean }>("/api/me/password", { method: "POST", body: JSON.stringify({ current, password }) });
}

export function setupAccount(username: string, password: string) {
  return request<{ ok: boolean }>("/api/setup", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}

export function login(username: string, password: string) {
  return request<{ ok: boolean }>("/api/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}

export function logout() {
  return request<{ ok: boolean }>("/api/logout", { method: "POST" });
}

export function listProjects() {
  return request<{ items: Project[] }>("/api/projects");
}

export function getProject(slug: string) {
  return request<Project>(`/api/projects/${slug}`);
}

export function createProject(body: {
  url?: string;
  name?: string;
  slug?: string;
  no_site?: boolean;
}) {
  return request<Project>("/api/projects", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function patchProject(slug: string, body: {
  url?: string;
  name?: string;
  no_site?: boolean;
  materials?: string;
  max_pages?: number;
  gsc_site?: string;
  ga4_property?: string;
}) {
  return request<Project>(`/api/projects/${slug}`, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export type AuditLayer = {
  key: string;
  name: string;
  question: string;
  status: "ok" | "warn" | "fail";
  issues?: string[];
  items?: { code: string; severity: string; title: string; count: number }[];
  blocked_by?: string;
  blocked_by_key?: string;
};

export type AuditPage = {
  url: string;
  title: string;
  score: number;
  grade: string;
  word_count: number;
  issue_codes?: string[];
  issues?: string[];
  dimensions?: Record<string, number>;
  blocks?: Record<string, boolean>;
  jsonld_types?: string[];
};

export type AuditReport = {
  avg_score: number;
  page_count: number;
  no_site: boolean;
  grade_distribution: Record<string, number>;
  layers: AuditLayer[];
  block_gap: { block: string; missing_pages: number; total: number }[];
  pages: AuditPage[];
  formula?: string;
  crawl_view?: CrawlView;
};

type CrawlRow = {
  url: string;
  status: number;
  score: number;
  words: number;
  h1: number;
  images_missing_alt: number;
  internal_links: number;
  millis: number;
  issues?: string[];
};

export type CrawlView = {
  health: number;
  pages: number;
  critical: number;
  content_avg: number;
  orphans: number;
  rows: CrawlRow[];
};

export function getAudit(slug: string) {
  return request<AuditReport>(`/api/projects/${slug}/audit`);
}

export function runAudit(slug: string) {
  return request<AuditReport>(`/api/projects/${slug}/audit`, { method: "POST" });
}

export type Question = {
  id?: number;
  qid: string;
  group: string;
  tags?: string[];
  system_tags?: string[];
  market?: string;
  text: string;
  intent?: string;
  enabled?: boolean;
};

export type Competitor = {
  id?: number;
  name: string;
  site?: string;
  aliases?: string[];
  market?: string;
  confirmed?: boolean | null;
};

export type KeyNumber = { fact: string; value: string; source: string };
export type BrandFacts = {
  aliases: string[];
  products: string[];
  industry: string;
  target_users: string;
  definition: string;
  disambiguation: string[];
  key_numbers: KeyNumber[];
  suitable: string[];
  unsuitable: string[];
};

export function getBrand(slug: string) {
  return request<{ name: string; site: string; brand: BrandFacts; facts_markdown: string }>(`/api/projects/${slug}/brand`);
}

export function saveBrand(slug: string, name: string, brand: BrandFacts) {
  return request<{ facts_markdown: string }>(`/api/projects/${slug}/brand`, { method: "PUT", body: JSON.stringify({ name, brand }) });
}

export function getQuestions(slug: string) {
  return request<{ items: Question[]; brand?: string; aliases?: string[]; site?: string }>(`/api/projects/${slug}/questions`);
}

export function saveQuestions(slug: string, items: Question[]) {
  return request<{ ok: boolean }>(`/api/projects/${slug}/questions`, {
    method: "PUT",
    body: JSON.stringify({ items }),
  });
}

export function getCompetitors(slug: string) {
  return request<{ items: Competitor[] }>(`/api/projects/${slug}/competitors`);
}

export function saveCompetitors(slug: string, items: Competitor[]) {
  return request<{ ok: boolean }>(`/api/projects/${slug}/competitors`, {
    method: "PUT",
    body: JSON.stringify({ items }),
  });
}

export type SampleRow = {
  id: number;
  platform: string;
  platform_name: string;
  qid: string;
  question_text: string;
  mentioned: boolean;
  rank?: number | null;
  needs_review: boolean;
  sampled_on?: string;
  ok: boolean;
  answer: string;
  sample_mode: string;
  brand_in_question: boolean;
  cited?: { url: string; title: string }[];
  web_queries?: string[];
  error?: string;
  negative?: boolean;
  manual_override?: boolean;
  competitors_mentioned?: string[];
  env?: string;
  run_id?: number | null;
};

export type KeyItem = {
  code: string;
  name: string;
  market?: string;
  ready: boolean;
  manual?: boolean;
  key_env?: string;
  model_env?: string;
  model?: string;
  key_tail?: string;
  note?: string;
  search?: boolean;
  official_base?: string;
  base?: string;
  base_env?: string;
  custom_base?: boolean;
  custom_model?: boolean;
  proxy_set?: boolean;
};

export function getKeys() {
  return request<{ items: KeyItem[] }>("/api/keys");
}

export function saveKeys(updates: Record<string, string>) {
  return request<{ items: KeyItem[] }>("/api/keys", {
    method: "PUT",
    body: JSON.stringify({ updates }),
  });
}

export function verifyKey(body: { code: string; key?: string; base?: string }) {
  return request<{ ok: boolean }>("/api/keys/verify", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function getSamples(slug: string, platform = "") {
  const q = platform ? `?platform=${encodeURIComponent(platform)}` : "";
  return request<{ items: SampleRow[] }>(`/api/projects/${slug}/samples${q}`);
}

export type KeywordRow = {
  query: string;
  page?: string;
  clicks: number;
  impressions: number;
  ctr: number;
  position: number;
  band: string;
};

export type GscPageRow = {
  url: string;
  clicks: number;
  impressions: number;
  ctr: number;
  position: number;
  band: string;
};

type SearchPeriod = {
  clicks: number;
  impressions: number;
  ctr: number;
  position: number;
  clicks_delta?: number | null;
  impressions_delta?: number | null;
  ctr_delta?: number | null;
  position_delta?: number | null;
  has_previous: boolean;
};

export function getKeywords(slug: string) {
  return request<{ items: KeywordRow[]; period: SearchPeriod }>(`/api/projects/${slug}/keywords`);
}

export function getGscPages(slug: string) {
  return request<{ items: GscPageRow[] }>(`/api/projects/${slug}/gsc-pages`);
}

export function getSavedKeywords(slug: string) {
  return request<{ items: { query: string; notes: string }[] }>(`/api/projects/${slug}/saved-keywords`);
}

export function saveKeyword(slug: string, query: string, notes = "") {
  return request<{ ok: boolean }>(`/api/projects/${slug}/saved-keywords`, {
    method: "POST",
    body: JSON.stringify({ query, notes }),
  });
}


export function getSampleSheet(slug: string) {
  return request<{ markdown: string }>(`/api/projects/${slug}/sample-sheet`);
}

export function importSamples(slug: string, text: string) {
  return request<{ count: number }>(`/api/projects/${slug}/samples/import`, {
    method: "POST",
    body: JSON.stringify({ text }),
  });
}

type Task = {
  code: string;
  priority: string;
  package: string;
  title: string;
  why: string;
  action: string;
  owner: string;
  effort: string;
  window: string;
  risk: string;
  status: string;
  market?: string;
  acceptance?: { type?: string; check?: string; desc?: string };
  affected?: string[];
  baseline_count?: number;
};

export function patchTask(slug: string, code: string, status: string, note = "") {
  return request<Task>(`/api/projects/${slug}/tasks/${code}`, {
    method: "PATCH",
    body: JSON.stringify({ status, note }),
  });
}

export type JobRow = {
  id: number;
  action: string;
  label?: string;
  status: string;
  log?: string;
  error?: string;
  started_at?: number | null;
  finished_at?: number | null;
};

export function startJob(slug: string, action: string, params?: Record<string, unknown>) {
  return request<{ job: JobRow }>(`/api/projects/${slug}/jobs`, {
    method: "POST",
    body: JSON.stringify({ action, params: params || {} }),
  });
}

export function listJobs(slug: string) {
  return request<{ jobs: JobRow[]; running: number | null }>(`/api/jobs?slug=${encodeURIComponent(slug)}`);
}

export function getJob(id: number, offset = 0) {
  return request<{ job: JobRow; log: string; offset: number }>(`/api/jobs/${id}?offset=${offset}`);
}

export function stopJob(id: number) {
  return request<{ ok: boolean }>(`/api/jobs/${id}/stop`, { method: "POST" });
}

// waitJob polls a background job until it leaves "running". Slow work such
// as crawling or drafting with a model runs as a job, so the page stays
// responsive and the work survives leaving the page.
export async function waitJob(id: number, everyMs = 2000): Promise<JobRow> {
  for (;;) {
    const r = await getJob(id);
    if (r.job.status !== "running" && r.job.status !== "queued") return { ...r.job, log: r.job.log || r.log };
    await new Promise((resolve) => setTimeout(resolve, everyMs));
  }
}

// runningJob returns the project's running job when it is one of actions.
export async function runningJob(slug: string, actions: string[]): Promise<JobRow | null> {
  const r = await listJobs(slug);
  const j = r.jobs?.find((x) => x.id === r.running);
  return j && actions.includes(j.action) ? j : null;
}



export function buildReport(slug: string) {
  return request<{ markdown: string; html: string; on: string }>(`/api/projects/${slug}/report`, { method: "POST" });
}

export function getReport(slug: string) {
  return request<{ markdown: string; html: string }>(`/api/projects/${slug}/report`);
}



export function patchMonitor(slug: string, every_days: number, runs_per_day?: number) {
  return request<Project>(`/api/projects/${slug}/monitor`, { method: "PATCH", body: JSON.stringify({ every_days, runs_per_day }) });
}

export type WebQueryRow = {
  query?: string;
  clicks?: number;
  impressions?: number;
  position?: number;
};

export type WebInsight = {
  from?: string;
  to?: string;
  gsc_clicks?: number | null;
  gsc_impressions?: number | null;
  ga_sessions?: number | null;
  ga_key_events?: number;
  ai_sessions?: number;
  top_queries?: WebQueryRow[];
  gap_queries?: WebQueryRow[];
  top_landings?: { landing: string; sessions: number }[];
};

export type GscSitemapRow = {
  path?: string;
  submitted?: number;
  errors?: number;
  warnings?: number;
  pending?: boolean;
};

export type GscIndexRow = {
  url?: string;
  verdict?: string;
  coverage_state?: string;
  indexing_state?: string;
  last_crawl?: string;
};

export type SourceView = {
  source: string;
  state: string;
  label: string;
  detail?: string;
  property?: string;
  through?: string;
  calendar?: string;
  can_reconnect?: boolean;
  can_sync?: boolean;
  backfill?: boolean;
};

export type WindowRow = {
  source: string;
  window_days?: number;
  finalized_through?: string;
  clicks?: number;
  impressions?: number;
  sessions?: number;
  previous_clicks?: number;
  previous_impressions?: number;
  previous_sessions?: number;
  covered_days?: number;
};

export type OfficialRow = {
  day: string;
  source: string;
  clicks?: number;
  impressions?: number;
  position?: number;
  sessions?: number;
  engaged?: number | null;
};

export type WebstatsSnapshot = {
  insight?: WebInsight | null;
  sitemaps?: GscSitemapRow[];
  index?: GscIndexRow[];
  sources?: SourceView[];
  windows?: WindowRow[];
  official?: OfficialRow[];
  deltas?: { source: string; metric: string; kind: string; dir: string; value: number }[];
};

export type GSCChoice = { site_url: string; level: string; selectable: boolean; suggested: boolean };
export type GAChoice = { id: string; name: string; suggested: boolean };
export type GoogleStatus = {
  oauth_configured: boolean;
  connected: boolean;
  needs_reconnect: boolean;
  email?: string;
  redirect_uri: string;
  gsc_site?: string;
  ga4_property?: string;
  gsc_choices?: GSCChoice[];
  ga_choices?: GAChoice[];
  list_error?: string;
};

export function getGoogleStatus(slug: string, pick = false) {
  return request<GoogleStatus>(`/api/projects/${slug}/google${pick ? "?pick=1" : ""}`);
}

export function saveGoogleChoice(slug: string, body: { gsc_site?: string; ga4_property?: string }) {
  return request<Project>(`/api/projects/${slug}/google`, { method: "POST", body: JSON.stringify(body) });
}

export function disconnectGoogle() {
  return request<{ ok: boolean }>("/api/google/disconnect", { method: "POST", body: "{}" });
}

export function getWebstats(slug: string) {
  return request<WebstatsSnapshot>(`/api/projects/${slug}/webstats`);
}

export function runWebstats(slug: string) {
  return request<{ gsc_rows?: number; ga_rows?: number; from?: string; to?: string; index_note?: string }>(
    `/api/projects/${slug}/webstats`,
    { method: "POST" },
  );
}

// ---- Opportunities and actions ----

export type OpportunitySource = "audit" | "citation" | "search" | "metric";

export type OpportunityItem = {
  key: string;
  source: OpportunitySource;
  kind: string;
  priority: "P0" | "P1" | "P2";
  title: string;
  why: string;
  fix: string;
  evidence?: string;
  refs?: string[];
  qid?: string;
  urls?: string[];
  acceptance: { type?: string; check?: string; desc?: string };
  detail?: Record<string, unknown>;
  status?: string;
  task_code?: string;
};

export function listOpportunities(slug: string, filter: { source?: string; status?: string } = {}) {
  const q = new URLSearchParams();
  if (filter.source) q.set("source", filter.source);
  if (filter.status) q.set("status", filter.status);
  const qs = q.toString();
  return request<{ items: OpportunityItem[]; hints?: { code: string; text: string }[] }>(`/api/projects/${slug}/opportunities${qs ? "?" + qs : ""}`);
}

export function acceptOpportunity(slug: string, key: string) {
  return request<{ code: string; status: string }>(`/api/projects/${slug}/opportunities/accept`, { method: "POST", body: JSON.stringify({ key }) });
}

export function dismissOpportunity(slug: string, key: string) {
  return request<{ code: string; status: string }>(`/api/projects/${slug}/opportunities/dismiss`, { method: "POST", body: JSON.stringify({ key }) });
}

// ---- Audit issues ----

export type AuditIssueRow = {
  id: number;
  url: string;
  code: string;
  severity: "critical" | "warning" | "info";
  layer: "access" | "discover" | "understand" | "cite";
  blocked: boolean;
  detail?: Record<string, unknown>;
  title: string;
  why: string;
  fix: string;
  surface: "seo" | "geo" | "both";
  scope: "site" | "page" | "content";
  evidence: string;
  refs: string[];
};

export type Reference = { key: string; cite: string; url: string; kind: string };

export function getAuditIssues(slug: string) {
  return request<{ items: AuditIssueRow[]; references: Record<string, Reference> }>(`/api/projects/${slug}/audit/issues`);
}

// ---- Sampling runs ----

export type SampleRunRow = {
  id: number;
  trigger: string;
  status: string;
  outcome: string;
  planned: number;
  succeeded: number;
  failed: number;
  est_tokens: number;
  used_tokens: number;
  started_at: number;
  finished_at: number | null;
  parent_id: number | null;
};

export function listRuns(slug: string) {
  return request<{ items: SampleRunRow[] }>(`/api/projects/${slug}/runs`);
}

export function retryRun(slug: string, id: number) {
  return request<{ id: number }>(`/api/projects/${slug}/runs/${id}/retry`, { method: "POST" });
}

export function overrideSample(slug: string, id: number, body: { mentioned?: boolean; negative?: boolean }) {
  return request<SampleRow>(`/api/projects/${slug}/samples/${id}`, { method: "PATCH", body: JSON.stringify(body) });
}
