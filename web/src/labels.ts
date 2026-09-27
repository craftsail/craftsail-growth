// SPDX-License-Identifier: AGPL-3.0-or-later

// Prompt groups, score parts and content blocks are stored with Chinese keys
// in the database and used by the backend. The UI shows them through these
// translation keys. This file and the Chinese catalog are the only places
// under src/ that may contain Chinese.
import type { Key } from "./i18n";

export const PROMPT_GROUPS: { key: string; label: Key; intent: "buyer" | "educate" | "probe" }[] = [
  { key: "推荐", label: "groups.recommendation", intent: "buyer" },
  { key: "比较", label: "groups.comparison", intent: "buyer" },
  { key: "替代", label: "groups.alternatives", intent: "buyer" },
  { key: "价格", label: "groups.pricing", intent: "buyer" },
  { key: "风险", label: "groups.risks", intent: "educate" },
  { key: "场景", label: "groups.usecase", intent: "educate" },
  { key: "品牌验证", label: "groups.brandcheck", intent: "probe" },
];

export function groupLabelKey(key: string | undefined): Key | undefined {
  return PROMPT_GROUPS.find((g) => g.key === key)?.label;
}

export const DEFAULT_GROUP = PROMPT_GROUPS[0].key;

export const SCORE_PARTS: { key: string; label: Key; max: number }[] = [
  { key: "可抓取性", label: "score.crawl", max: 15 },
  { key: "内容长度", label: "score.length", max: 15 },
  { key: "结构规范", label: "score.structure", max: 20 },
  { key: "可抽取块", label: "score.blocks", max: 25 },
  { key: "权威信号", label: "score.authority", max: 15 },
  { key: "对题性", label: "score.relevance", max: 10 },
];

export const BLOCKS: { key: string; label: Key }[] = [
  { key: "定义", label: "blocks.definition" },
  { key: "数字事实", label: "blocks.numbers" },
  { key: "对比", label: "blocks.comparison" },
  { key: "操作步骤", label: "blocks.steps" },
];
