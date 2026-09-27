// SPDX-License-Identifier: AGPL-3.0-or-later

import { createContext, useContext, type ReactNode } from "react";
import type { Project, SessionUser } from "../api";

type AccessValue = { user: SessionUser | null; isAdmin: boolean; canEdit: boolean };

const Ctx = createContext<AccessValue>({ user: null, isAdmin: false, canEdit: false });

// AccessProvider answers "may this person change the current project?".
// The server enforces the same rule; this only hides what would fail.
export function AccessProvider({ user, project, children }: { user: SessionUser | null; project?: Project; children: ReactNode }) {
  const isAdmin = user?.role === "admin";
  const canEdit = isAdmin || project?.access === "edit";
  return <Ctx.Provider value={{ user, isAdmin, canEdit }}>{children}</Ctx.Provider>;
}

export function useAccess() {
  return useContext(Ctx);
}
