// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ReactNode } from "react";

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="rounded-2xl border border-dashed border-gray-300 p-8 text-center">
      <p className="text-sm font-medium text-gray-700">{title}</p>
      {children && <div className="mt-2 text-sm text-gray-500">{children}</div>}
    </div>
  );
}
