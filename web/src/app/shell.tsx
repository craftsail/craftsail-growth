// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useRef, useState } from "react";
import { Link, Navigate, NavLink, Outlet, useLocation, useNavigate, useParams } from "react-router-dom";
import { IconCheck, IconChevronRight, IconInfoCircle, IconLayoutSidebar, IconSelector, IconSettings } from "@tabler/icons-react";
import type { Project, SessionUser } from "../api";
import { AccessProvider, useAccess } from "./access";
import { helpTopicFor, itemFor, titleFor } from "./nav";
import { useI18n } from "../i18n";
import { Sidebar } from "../components/layout/Sidebar";

// Layout: a gray canvas with the sidebar on the left and the page in a white
// rounded panel on the right. The panel has a slim top bar (sidebar toggle and
// breadcrumb), then the page title and the page itself.
export function AppShell({
  projects,
  onProject,
  onCreated,
  onLogout,
  user,
}: {
  user: SessionUser | null;
  projects: Project[];
  onProject?: (p: Project) => void;
  onCreated?: (p: Project) => void;
  onLogout?: () => void;
}) {
  const { slug } = useParams();
  const loc = useLocation();
  const current = projects.find((p) => p.slug === slug) || projects[0];
  const [collapsed, setCollapsed] = useState(localStorage.getItem("sidebar-collapsed") === "1");
  const [open, setOpen] = useState(false);
  function toggleSidebar() {
    if (window.matchMedia("(min-width: 1024px)").matches) {
      const next = !collapsed;
      setCollapsed(next);
      localStorage.setItem("sidebar-collapsed", next ? "1" : "0");
    } else {
      setOpen((v) => !v);
    }
  }

  const { t } = useI18n();
  const page = t(titleFor(loc.pathname));
  const item = itemFor(loc.pathname);
  const tabs = item?.tabs;
  const leaf = loc.pathname.split("/").slice(3).join("/");
  const tab = tabs?.find((x) => x.to === leaf);

  // A project that is not in the list (not shared, or left over in the
  // address from another user) would show one project and load another.
  if (slug && !projects.some((p) => p.slug === slug)) {
    return <Navigate to={`/p/${projects[0].slug}/overview`} replace />;
  }

  return (
    <AccessProvider user={user} project={current}>
    <div className="flex min-h-screen bg-gray-50">
      {open && (
        <button type="button" className="fixed inset-0 z-30 bg-black/30 lg:hidden" aria-label={t("shell.closeMenu")} onClick={() => setOpen(false)} />
      )}
      <Sidebar
        collapsed={collapsed}
        open={open}
        user={user?.username || ""}
        onLogout={onLogout}
        onNavigate={() => setOpen(false)}
      />
      <div className="min-w-0 flex-1 p-2 lg:py-3 lg:pr-3 lg:pl-0">
        <div className="flex min-h-[calc(100vh-1rem)] flex-col rounded-2xl border border-gray-200 bg-white shadow-card lg:min-h-[calc(100vh-1.5rem)]">
          <div className="flex h-14 shrink-0 items-center gap-3 border-b border-gray-100 px-4 lg:px-5">
            <button type="button" className="btn btn-ghost btn-icon p-1.5" aria-label={t("shell.toggleSidebar")} title={t("shell.toggleSidebar")} onClick={toggleSidebar}>
              <IconLayoutSidebar size={18} stroke={1.7} />
            </button>
            <span className="h-5 w-px bg-gray-200" aria-hidden="true" />
            <nav className="flex min-w-0 items-center gap-2 text-sm" aria-label={t("shell.breadcrumb")}>
              <ProjectMenu projects={projects} current={current} />
              <IconChevronRight size={14} stroke={1.8} className="shrink-0 text-gray-400" aria-hidden="true" />
              <span className={"truncate " + (tab && tab.to !== item?.to ? "text-gray-500" : "text-gray-900")}>{page}</span>
              {tab && tab.to !== item?.to && (
                <>
                  <IconChevronRight size={14} stroke={1.8} className="shrink-0 text-gray-400" aria-hidden="true" />
                  <span className="truncate text-gray-900">{t(tab.label)}</span>
                </>
              )}
            </nav>
          </div>
          <main className={"min-w-0 flex-1 px-5 py-7 lg:px-8" + (tabs ? " has-tabs" : "")}>
            <div className={(tabs ? "mb-4" : "mb-6") + " flex items-center gap-2"}>
              <h1 className="page-title">{page}</h1>
              {!["help", "settings/password"].includes(leaf) && <ViewOnlyBadge />}
              {helpTopicFor(loc.pathname) && (
                <Link to={`/p/${current?.slug}/help#${helpTopicFor(loc.pathname)}`} className="text-gray-400 hover:text-primary-600" title={t("shell.howPageWorks")} aria-label={t("shell.howPageWorks")}>
                  <IconInfoCircle size={22} stroke={1.7} />
                </Link>
              )}
            </div>
            {tabs && (
              <nav className="mb-6 flex gap-6 border-b border-gray-200" aria-label={page}>
                {tabs.map((x) => (
                  <NavLink key={x.to} to={`/p/${current?.slug}/${x.to}`} end
                    className={({ isActive }) => "-mb-px border-b-2 px-0.5 pb-2.5 text-sm font-medium no-underline transition-colors " + (isActive ? "border-primary-600 text-primary-700" : "border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-800")}>
                    {t(x.label)}
                  </NavLink>
                ))}
              </nav>
            )}
            <Outlet context={{ project: current, projects, onProject, onCreated }} />
          </main>
        </div>
      </div>
    </div>
    </AccessProvider>
  );
}

// ViewOnlyBadge tells a viewer why pages have no Save or Run buttons.
function ViewOnlyBadge() {
  const { t } = useI18n();
  const { canEdit } = useAccess();
  if (canEdit) return null;
  return <span className="badge badge-gray ml-1 self-center">{t("access.viewOnly")}</span>;
}

// ProjectMenu is the first breadcrumb segment: the current project, which
// opens a list to switch to another project on the same page.
function ProjectMenu({ projects, current }: { projects: Project[]; current?: Project }) {
  const loc = useLocation();
  const nav = useNavigate();
  const { t } = useI18n();
  const { isAdmin } = useAccess();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const close = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, [open]);
  function pick(slug: string) {
    setOpen(false);
    const rest = loc.pathname.replace(/^\/p\/[^/]+/, "") || "/overview";
    nav(`/p/${slug}${rest}${loc.search}`);
  }
  return (
    <div className="relative" ref={ref}>
      <button type="button" className="flex max-w-56 items-center gap-1 rounded-lg px-2 py-1 text-gray-600 hover:bg-gray-100 hover:text-gray-900" onClick={() => setOpen((v) => !v)} aria-expanded={open} aria-haspopup="listbox" title={t("shell.switchProject")}>
        <span className="truncate font-medium">{current?.name || t("nav.projects")}</span>
        <IconSelector size={14} stroke={1.8} className="shrink-0 text-gray-400" />
      </button>
      {open && (
        <div className="absolute left-0 z-50 mt-1 w-72 rounded-xl border border-gray-200 bg-white py-1 shadow-lg" role="listbox">
          <div className="px-3 pb-1 pt-2 text-xs font-semibold uppercase tracking-wider text-gray-400">{t("nav.projects")}</div>
          {projects.map((p) => (
            <button key={p.slug} type="button" role="option" aria-selected={p.slug === current?.slug} className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-gray-50" onClick={() => pick(p.slug)}>
              <span className="min-w-0 flex-1">
                <span className="block truncate font-medium text-gray-900">{p.name}</span>
                <span className="block truncate text-xs text-gray-500">{p.no_site ? t("common.noWebsite") : (p.site || p.slug)}</span>
              </span>
              {p.slug === current?.slug && <IconCheck size={16} className="shrink-0 text-primary-600" />}
            </button>
          ))}
          {isAdmin && <div className="my-1 border-t border-gray-100" />}
          {isAdmin && <Link to={`/p/${current?.slug}/settings/projects`} className="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 no-underline hover:bg-gray-50" onClick={() => setOpen(false)}>
            <IconSettings size={16} className="text-gray-400" /> {t("shell.manageProjects")}
          </Link>}
        </div>
      )}
    </div>
  );
}
