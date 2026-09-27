// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { getKeys, saveKeys, type KeyItem } from "../../api";

export function useKeys() {
  const [items, setItems] = useState<KeyItem[]>([]);
  const [draft, setDraft] = useState<Record<string, string>>({});
  const [err, setErr] = useState("");
  const [ok, setOk] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    getKeys().then((d) => setItems(d.items || [])).catch((e: Error) => setErr(e.message));
  }, []);
  async function save(updates: Record<string, string>, msg: string) {
    setBusy(true); setErr(""); setOk("");
    try {
      const r = await saveKeys(updates);
      setItems(r.items || []);
      const next = { ...draft };
      for (const k of Object.keys(updates)) delete next[k];
      setDraft(next);
      setOk(msg);
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return { items, draft, setDraft, err, setErr, ok, setOk, busy, save };
}
