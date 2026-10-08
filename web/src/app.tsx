// SPDX-License-Identifier: AGPL-3.0-or-later

import { Link, Navigate, Route, Routes } from "react-router-dom";
import { useEffect, useState, type ReactNode } from "react";
import { SIGNED_OUT, getSession, listProjects, logout, type Project, type SessionUser } from "./api";
import { useAccess } from "./app/access";
import { Logo } from "./components/brand/Logo";
import { Users } from "./features/settings/users";
import { PasswordPage } from "./features/settings/password";
import { AppShell } from "./app/shell";
import { Login } from "./features/auth/login";
import { ChangeDefaultPassword } from "./features/auth/change-default";
import { Onboard } from "./features/auth/onboard";
import { Help } from "./features/help/help";
import { Competitors } from "./features/geo/competitors";
import { Reports } from "./features/reports/reports";
import { Answers } from "./features/measure/answers";
import { Schedule } from "./features/settings/schedule";
import { Projects } from "./features/settings/projects";
import { LEGACY_REDIRECTS } from "./app/nav";
import { Opportunities } from "./features/actions/opportunities";
import { Overview } from "./features/overview/overview";
import { Facts } from "./features/geo/facts";
import { MeasureCitations, MeasureFanout, MeasureShare, MeasureVisibility } from "./features/measure/pages";
import { MeasurePrompt } from "./features/measure/prompt";
import { Questions } from "./features/geo/questions";
import { ModelProviders } from "./features/settings/providers";
import { GoogleData } from "./features/settings/google";
import { Readiness } from "./features/audit/readiness";
import { AuditIssues } from "./features/audit/issues";
import { AuditPages } from "./features/audit/pages";
import { GAExplorer } from "./features/search/ga-explore";
import { GscPages } from "./features/search/gsc-pages";
import { Keywords } from "./features/search/keywords";
import { Webstats } from "./features/search/webstats";
import { useI18n } from "./i18n";

export function App() {
  const { t } = useI18n();
  const [authed, setAuthed] = useState<boolean | null>(null);
  const [needSetup, setNeedSetup] = useState(false);
  const [user, setUser] = useState<SessionUser | null>(null);
  const [projects, setProjects] = useState<Project[] | null>(null);
  const [err, setErr] = useState("");

  function loadSession() {
    getSession()
      .then((s) => {
        setNeedSetup(!!s.setup);
        setUser(s.user || null);
        setAuthed(!!s.ok);
      })
      .catch(() => setAuthed(false));
  }
  useEffect(loadSession, []);
  useEffect(() => {
    const out = () => { setUser(null); setAuthed(false); };
    window.addEventListener(SIGNED_OUT, out);
    return () => window.removeEventListener(SIGNED_OUT, out);
  }, []);
  const signOut = async () => { await logout(); setUser(null); setAuthed(false); };

  const mustChange = !!user?.must_change_password;
  useEffect(() => {
    if (!authed || mustChange) {
      setProjects(null);
      return;
    }
    listProjects()
      .then((d) => setProjects(d.items || []))
      .catch((e: Error) => setErr(e.message));
  }, [authed, mustChange]);

  if (authed === false) {
    return <Login setup={needSetup} onOk={() => { setErr(""); setNeedSetup(false); loadSession(); }} />;
  }
  if (authed && mustChange) {
    return <ChangeDefaultPassword onOk={loadSession} onLogout={signOut} />;
  }
  if (authed === null || (authed && !projects && !err)) {
    return <div className="p-8 text-sm text-gray-500">{t("common.loading")}</div>;
  }
  if (err) {
    return (
      <div className="p-8">
        <p className="alert alert-error">{err}</p>
        <p className="page-description mb-5">{t("auth.apiDown")}</p>
      </div>
    );
  }
  if (!projects) {
    return <div className="p-8 text-sm text-gray-500">{t("common.loading")}</div>;
  }
  if (projects.length === 0 && user?.role !== "admin") {
    return <NoProjects onLogout={signOut} />;
  }
  if (projects.length === 0) {
    return (
      <Routes>
        <Route path="/settings" element={<div className="mx-auto max-w-6xl p-6"><Link className="btn btn-secondary mb-4" to="/">{t("auth.backToFirst")}</Link><h1 className="page-title mb-6">{t("nav.providers")}</h1><ModelProviders /></div>} />
        <Route path="/help" element={<div className="p-6"><Help /></div>} />
        <Route path="*" element={<Onboard onCreated={(p) => setProjects([p])} />} />
      </Routes>
    );
  }

  return (
    <Routes>
      <Route path="/" element={<Navigate to={`/p/${projects[0].slug}/overview`} replace />} />
      <Route path="/onboard" element={user?.role === "admin" ? <Onboard onCreated={(p) => setProjects([p, ...projects])} /> : <Navigate to="/" replace />} />
      <Route path="/settings" element={<Navigate to={`/p/${projects[0].slug}/settings`} replace />} />
      <Route path="/help" element={<Navigate to={`/p/${projects[0].slug}/help`} replace />} />
      <Route path="/p/:slug" element={<AppShell user={user} projects={projects} onProject={(p) => {
        setProjects((prev) => (prev || []).map((x) => (x.slug === p.slug ? { ...x, ...p } : x)));
      }} onCreated={(p) => setProjects((prev) => [p, ...(prev || []).filter((x) => x.slug !== p.slug)])} onLogout={signOut} />}>
        <Route index element={<Navigate to="overview" replace />} />
        <Route path="overview" element={<Overview />} />
        <Route path="ai/visibility" element={<MeasureVisibility />} />
        <Route path="ai/share-of-voice" element={<MeasureShare />} />
        <Route path="ai/citations" element={<MeasureCitations />} />
        <Route path="ai/fan-out" element={<MeasureFanout />} />
        <Route path="ai/prompts/:qid" element={<MeasurePrompt />} />
        <Route path="ai/answers" element={<Answers />} />
        <Route path="opportunities" element={<Opportunities />} />
        <Route path="audit" element={<Readiness />} />
        <Route path="audit/issues" element={<AuditIssues />} />
        <Route path="audit/pages" element={<AuditPages />} />
        <Route path="search" element={<Webstats />} />
        <Route path="search/keywords" element={<Keywords />} />
        <Route path="search/pages" element={<GscPages />} />
        <Route path="search/channels" element={<GAExplorer report="channel" />} />
        <Route path="search/landings" element={<GAExplorer report="landing" />} />
        <Route path="reports" element={<Reports />} />
        <Route path="settings/projects" element={<AdminOnly><Projects /></AdminOnly>} />
        <Route path="settings/users" element={<AdminOnly><Users /></AdminOnly>} />
        <Route path="settings/password" element={<PasswordPage />} />
        <Route path="settings/brand" element={<Facts />} />
        <Route path="settings/competitors" element={<Competitors />} />
        <Route path="settings/questions" element={<Questions />} />
        <Route path="settings/schedule" element={<Schedule />} />
        <Route path="settings/providers" element={<AdminOnly><ModelProviders /></AdminOnly>} />
        <Route path="settings/google" element={<AdminOnly><GoogleData /></AdminOnly>} />
        <Route path="help" element={<Help />} />
        {Object.entries(LEGACY_REDIRECTS).map(([from, to]) => (
          <Route key={from} path={from} element={<Navigate to={`../${to}`} replace />} />
        ))}
        <Route path="*" element={<Navigate to="overview" replace />} />
      </Route>
    </Routes>
  );
}



// AdminOnly keeps members off workspace pages typed in by hand; the server
// refuses their requests anyway.
function AdminOnly({ children }: { children: ReactNode }) {
  const { isAdmin } = useAccess();
  return isAdmin ? <>{children}</> : <Navigate to="../overview" replace />;
}

// NoProjects is what a member sees before any project is shared with them.
function NoProjects({ onLogout }: { onLogout: () => void }) {
  const { t } = useI18n();
  return (
    <div className="mesh-gradient flex min-h-screen items-start justify-center bg-gray-50 px-4">
      <div className="card mt-[8vh] w-full max-w-lg p-8 shadow-glass">
        <Logo />
        <h1 className="mt-6 text-2xl font-bold tracking-tight text-gray-900">{t("access.noProjectsTitle")}</h1>
        <p className="mt-1 text-sm leading-6 text-gray-500">{t("access.noProjects")}</p>
        <button type="button" className="btn btn-secondary mt-6" onClick={onLogout}>{t("shell.logout")}</button>
      </div>
    </div>
  );
}
