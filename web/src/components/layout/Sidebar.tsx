// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useRef, useState } from "react";
import { NavLink, useLocation, useParams } from "react-router-dom";
import {
  IconArrowsSplit2, IconBook, IconBrandGoogle, IconCalendarTime, IconChartPie, IconCheck, IconCpu, IconEye,
  IconFolders, IconHome, IconId, IconKey, IconLanguage, IconListCheck, IconListDetails, IconLogout, IconMessages,
  IconQuote, IconReport, IconSelector, IconStethoscope, IconUserCog, IconUsersGroup, IconWorldSearch, type Icon as TablerIcon,
} from "@tabler/icons-react";
import { NAV, itemFor, type NavIcon } from "../../app/nav";
import { LogoMark, Wordmark } from "../brand/Logo";
import { LOCALES, useI18n } from "../../i18n";
import { useAccess } from "../../app/access";

const ICONS: Record<NavIcon, TablerIcon> = {
  home: IconHome, eye: IconEye, pie: IconChartPie, quote: IconQuote, split: IconArrowsSplit2, messages: IconMessages,
  listcheck: IconListCheck, stethoscope: IconStethoscope, search: IconWorldSearch, report: IconReport,
  folders: IconFolders, id: IconId, users: IconUsersGroup, list: IconListDetails, calendar: IconCalendarTime,
  cpu: IconCpu, google: IconBrandGoogle, usercog: IconUserCog,
};

function Icon({ name, size = 18 }: { name: NavIcon; size?: number }) {
  const C = ICONS[name];
  return <C size={size} stroke={1.7} className="shrink-0" aria-hidden="true" />;
}

const linkClass = (active: boolean, collapsed: boolean) =>
  "sidebar-link py-2 " + (collapsed ? "justify-center px-0 " : "") + (active ? "sidebar-link-active" : "");

// Sidebar: flat titled sections, every page one click away. Sub-views of a
// page are tabs inside that page, so no item here expands or collapses.
export function Sidebar({
  collapsed,
  open,
  user,
  onLogout,
  onNavigate,
}: {
  collapsed: boolean;
  open: boolean;
  user: string;
  onLogout?: () => void;
  onNavigate: () => void;
}) {
  const { slug = "" } = useParams();
  const loc = useLocation();
  const { t } = useI18n();
  const { isAdmin } = useAccess();
  const active = itemFor(loc.pathname);
  // Workspace items are for admins; drop sections left empty.
  const sections = NAV.map((sec) => ({ ...sec, items: sec.items.filter((it) => !it.admin || isAdmin) })).filter((sec) => sec.items.length > 0);
  const width = collapsed ? 72 : 256;
  return (
    <aside
      className={
        "fixed inset-y-0 left-0 z-40 flex shrink-0 flex-col bg-gray-50 transition-[width,transform] duration-300 lg:sticky lg:top-0 lg:h-screen lg:translate-x-0 " +
        (open ? "translate-x-0 shadow-xl" : "-translate-x-full")
      }
      style={{ width, minWidth: width }}
    >
      <div className={"flex h-16 shrink-0 items-center " + (collapsed ? "justify-center px-3" : "px-5")}>
        <NavLink to={`/p/${slug}/overview`} onClick={onNavigate} className="flex min-w-0 items-center gap-2.5 no-underline" title="craftsail growth">
          <LogoMark size={34} />
          {!collapsed && <Wordmark />}
        </NavLink>
      </div>

      <nav className="flex-1 overflow-y-auto px-3 pb-3" aria-label={t("shell.mainNav")}>
        {sections.map((sec, i) => (
          <div key={sec.id} className={i === 0 ? "" : "mt-4"}>
            {sec.label && (collapsed
              ? <div className="mx-3 mb-2 border-t border-gray-200" />
              : <div className="mb-1 px-3.5 text-[11px] font-semibold uppercase tracking-wider text-gray-400">{t(sec.label)}</div>)}
            {sec.items.map((it) => (
              <NavLink
                key={it.id}
                to={`/p/${slug}/${it.to}`}
                end
                title={t(it.label)}
                onClick={onNavigate}
                className={() => linkClass(active?.id === it.id, collapsed) + " mb-0.5"}
              >
                <Icon name={it.icon} />
                {!collapsed && <span className="truncate">{t(it.label)}</span>}
              </NavLink>
            ))}
          </div>
        ))}
      </nav>

      <div className="shrink-0 px-3 pb-3">
        <NavLink
          to={`/p/${slug}/help`}
          onClick={onNavigate}
          title={t("nav.help")}
          className={({ isActive }) => linkClass(isActive, collapsed) + " mb-2"}
        >
          <IconBook size={18} stroke={1.7} className="shrink-0" aria-hidden="true" />
          {!collapsed && <span>{t("nav.help")}</span>}
        </NavLink>
        <div className="border-t border-gray-200 pt-3">
          <UserMenu user={user} collapsed={collapsed} onLogout={onLogout} />
        </div>
      </div>
    </aside>
  );
}

function UserMenu({ user, collapsed, onLogout }: { user: string; collapsed: boolean; onLogout?: () => void }) {
  const { t, locale, setLocale } = useI18n();
  const { slug = "" } = useParams();
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
  const name = user || t("shell.signedIn");
  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        className={"flex w-full items-center gap-3 rounded-xl p-1.5 text-left hover:bg-gray-100 " + (collapsed ? "justify-center" : "")}
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        title={name}
      >
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-primary-500 to-primary-600 text-sm font-semibold text-white">
          {name.slice(0, 1).toUpperCase()}
        </span>
        {!collapsed && (
          <>
            <span className="min-w-0 flex-1">
              <span className="block truncate text-sm font-medium text-gray-900">{name}</span>
              <span className="block truncate text-xs text-gray-500">{t("shell.localAccount")}</span>
            </span>
            <IconSelector size={16} stroke={1.8} className="shrink-0 text-gray-400" aria-hidden="true" />
          </>
        )}
      </button>
      {open && (
        <div className={"absolute bottom-full z-50 mb-2 w-56 rounded-xl border border-gray-200 bg-white py-1 shadow-lg " + (collapsed ? "left-0" : "left-0 right-0 w-auto")}>
          <div className="border-b border-gray-100 px-4 py-2 text-xs text-gray-500">{t("shell.signedInAs", { name })}</div>
          <div className="flex items-center gap-2 px-4 pb-1 pt-2 text-xs font-medium text-gray-500"><IconLanguage size={14} /> {t("lang.label")}</div>
          {LOCALES.map((l) => (
            <button key={l.id} type="button" className="flex w-full items-center justify-between px-4 py-1.5 text-sm text-gray-700 hover:bg-gray-100" onClick={() => { setLocale(l.id); setOpen(false); }}>
              {l.label}
              {locale === l.id && <IconCheck size={15} className="text-primary-600" />}
            </button>
          ))}
          <NavLink to={`/p/${slug}/settings/password`} className="mt-1 flex w-full items-center gap-2 border-t border-gray-100 px-4 py-2 text-sm text-gray-700 no-underline hover:bg-gray-100" onClick={() => setOpen(false)}>
            <IconKey size={16} stroke={1.8} /> {t("access.changePassword")}
          </NavLink>
          {onLogout && (
            <button type="button" className="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100" onClick={onLogout}>
              <IconLogout size={16} stroke={1.8} /> {t("shell.logout")}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
