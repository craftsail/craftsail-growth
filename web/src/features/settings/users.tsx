// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useMemo, useState } from "react";
import { createUser, deleteUser, listProjects, listUsers, patchUser, putUserAccess, resetUserPassword, type Access, type Project, type Role, type UserRow } from "../../api";
import { useAccess } from "../../app/access";
import { useI18n, type Key } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";

type Level = Access | "";
const LEVELS: { id: Level; label: Key }[] = [
  { id: "", label: "users.access.none" },
  { id: "view", label: "users.access.view" },
  { id: "edit", label: "users.access.edit" },
];

// Users is the admin page for who can sign in and which projects each
// member can view or edit. Admins see and edit every project.
export function Users() {
  const { t } = useI18n();
  const { user: me } = useAccess();
  const [users, setUsers] = useState<UserRow[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [sel, setSel] = useState<number | null>(null);
  const [err, setErr] = useState("");
  const [note, setNote] = useState("");

  function load() {
    Promise.all([listUsers(), listProjects()])
      .then(([u, p]) => { setUsers(u.items || []); setProjects(p.items || []); })
      .catch((e: Error) => setErr(e.message));
  }
  useEffect(load, []);

  const current = users.find((u) => u.id === sel) || users[0];
  // run returns whether fn succeeded, so forms can keep input on failure.
  async function run(fn: () => Promise<unknown>, done?: string): Promise<boolean> {
    setErr(""); setNote("");
    try {
      await fn();
      if (done) setNote(done);
      load();
      return true;
    } catch (e) {
      setErr((e as Error).message);
      return false;
    }
  }

  return (
    <section>
      <p className="page-description">{t("users.description")}</p>
      {err && <div className="alert alert-error mb-4">{err}</div>}
      {note && <div className="alert alert-success mb-4">{note}</div>}
      <div className="grid gap-6 lg:grid-cols-[280px_minmax(0,1fr)]">
        <div className="space-y-4">
          <AddUser onAdd={(name, pass, role) => run(async () => { const r = await createUser(name, pass, role); setSel(r.id); }, t("users.added", { name }))} />
          <nav className="card p-2" aria-label={t("nav.users")}>
            {users.map((u) => (
              <button key={u.id} type="button" onClick={() => setSel(u.id)}
                className={"subnav-link justify-between" + (current?.id === u.id ? " subnav-link-active" : "")}>
                <span className="truncate">{u.username}</span>
                <span className="flex shrink-0 gap-1">
                  {u.role === "admin" && <span className="badge badge-primary">{t("users.roles.admin")}</span>}
                  {u.disabled && <span className="badge badge-gray">{t("users.disabled")}</span>}
                </span>
              </button>
            ))}
          </nav>
        </div>
        {current && (
          <UserDetail
            key={current.id}
            u={current}
            self={me?.id === current.id}
            projects={projects}
            onRole={(role) => run(() => patchUser(current.id, { role }))}
            onDisabled={(disabled) => run(() => patchUser(current.id, { disabled }))}
            onReset={(p) => run(() => resetUserPassword(current.id, p), t("users.passwordReset"))}
            onDelete={() => { if (window.confirm(t("users.deleteConfirm", { name: current.username }))) run(() => deleteUser(current.id)).then(() => setSel(null)); }}
            onAccess={(slug, access) => run(() => putUserAccess(current.id, [{ slug, access }]))}
          />
        )}
      </div>
    </section>
  );
}

function AddUser({ onAdd }: { onAdd: (name: string, pass: string, role: Role) => Promise<boolean> }) {
  const { t } = useI18n();
  const [name, setName] = useState("");
  const [pass, setPass] = useState("");
  const [role, setRole] = useState<Role>("member");
  return (
    <form className="card space-y-2 p-4" onSubmit={async (e) => { e.preventDefault(); if (await onAdd(name.trim(), pass, role)) { setName(""); setPass(""); } }}>
      <div className="flex items-center gap-1 text-sm font-semibold text-gray-900">{t("users.add")}<HelpTip id="addUser" /></div>
      <input className="input" placeholder={t("users.username")} value={name} onChange={(e) => setName(e.target.value)} autoComplete="off" aria-label={t("users.username")} />
      <input className="input" type="password" placeholder={t("users.tempPassword")} value={pass} onChange={(e) => setPass(e.target.value)} autoComplete="new-password" aria-label={t("users.tempPassword")} />
      <select className="input" value={role} onChange={(e) => setRole(e.target.value as Role)} aria-label={t("users.role")}>
        <option value="member">{t("users.roles.member")}</option>
        <option value="admin">{t("users.roles.admin")}</option>
      </select>
      <button type="submit" className="btn btn-primary w-full" disabled={!name.trim() || pass.length < 12}>{t("users.add")}</button>
      <p className="text-xs text-gray-500">{t("users.addHint")}</p>
    </form>
  );
}

function UserDetail(props: {
  u: UserRow;
  self: boolean;
  projects: Project[];
  onRole: (r: Role) => void;
  onDisabled: (d: boolean) => void;
  onReset: (p: string) => void;
  onDelete: () => void;
  onAccess: (slug: string, a: Level) => void;
}) {
  const { t } = useI18n();
  const { u, self, projects } = props;
  const [pass, setPass] = useState("");
  const grant = useMemo(() => Object.fromEntries(u.projects.map((p) => [p.slug, p.access])) as Record<string, Access>, [u]);
  return (
    <div className="card space-y-6 p-6">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="m-0 text-lg font-semibold text-gray-900">{u.username}</h2>
        {self && <span className="badge badge-gray">{t("users.you")}</span>}
        <select className="input w-40" value={u.role} disabled={self} onChange={(e) => props.onRole(e.target.value as Role)} aria-label={t("users.role")}>
          <option value="member">{t("users.roles.member")}</option>
          <option value="admin">{t("users.roles.admin")}</option>
        </select>
        <label className="inline-flex items-center gap-2 text-sm text-gray-700">
          <input type="checkbox" checked={u.disabled} disabled={self} onChange={(e) => props.onDisabled(e.target.checked)} /> {t("users.disable")}
        </label>
        <HelpTip id="disableUser" />
        {!self && <button type="button" className="btn btn-ghost ml-auto text-red-700" onClick={props.onDelete}>{t("users.delete")}</button>}
        {!self && <HelpTip id="deleteUser" />}
      </div>

      <div>
        <h3 className="mb-2 flex items-center gap-1 text-sm font-semibold text-gray-900">{t("users.projectAccess")}<HelpTip id="userAccess" /></h3>
        {u.role === "admin" ? (
          <p className="text-sm text-gray-500">{t("users.adminAll")}</p>
        ) : (
          <div className="overflow-hidden rounded-xl border border-gray-200">
            <table className="w-full text-sm">
              <tbody>
                {projects.map((p) => (
                  <tr key={p.slug} className="border-b border-gray-100 last:border-0">
                    <td className="px-4 py-2.5">
                      <div className="font-medium text-gray-900">{p.name}</div>
                      <div className="text-xs text-gray-500">{p.no_site ? t("common.noWebsite") : (p.site || p.slug)}</div>
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <div className="inline-flex rounded-lg bg-gray-100 p-0.5" role="radiogroup" aria-label={p.name}>
                        {LEVELS.map((lv) => {
                          const on = (grant[p.slug] || "") === lv.id;
                          return (
                            <button key={lv.id || "none"} type="button" role="radio" aria-checked={on}
                              className={"rounded-md px-3 py-1 text-xs " + (on ? "bg-white font-medium text-primary-700 shadow-sm" : "text-gray-600 hover:text-gray-900")}
                              onClick={() => { if (!on) props.onAccess(p.slug, lv.id); }}>
                              {t(lv.label)}
                            </button>
                          );
                        })}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <p className="mt-2 text-xs text-gray-500">{t("users.accessHint")}</p>
      </div>

      {!self && (
        <form className="flex flex-wrap items-end gap-2 border-t border-gray-100 pt-5" onSubmit={(e) => { e.preventDefault(); props.onReset(pass); setPass(""); }}>
          <label className="block text-sm"><span className="mb-1 block font-medium text-gray-700">{t("users.resetPassword")}</span>
            <input className="input w-72" type="password" value={pass} onChange={(e) => setPass(e.target.value)} autoComplete="new-password" /></label>
          <button type="submit" className="btn btn-secondary" disabled={pass.length < 12}>{t("users.setPassword")}</button>
          <HelpTip id="setPassword" />
        </form>
      )}
    </div>
  );
}
