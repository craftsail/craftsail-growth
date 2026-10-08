// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Key } from "../i18n";
import type { TopicId } from "../features/help/kit";

// Every button that has a "?" tip. label is the button's own text, the tip
// text is tips.<id>, and topic is the help topic whose "Buttons on this page"
// section explains it in detail (help anchor #<topic>/<id>).
export const TIPS = {
  serve: { topic: "schedule", label: "schedule.actions.serve" },
  sample: { topic: "schedule", label: "schedule.actions.sample" },
  indexing: { topic: "schedule", label: "schedule.actions.indexing" },
  syncGoogle: { topic: "schedule", label: "schedule.actions.webstats" },
  crawl: { topic: "schedule", label: "schedule.actions.crawl" },
  audit: { topic: "schedule", label: "schedule.actions.audit" },
  verify: { topic: "schedule", label: "schedule.actions.verify" },
  scheduleSave: { topic: "schedule", label: "common.save" },
  retry: { topic: "schedule", label: "schedule.retry" },
  stop: { topic: "schedule", label: "schedule.stop" },
  auditCrawl: { topic: "audit", label: "audit.actions.crawl" },
  auditAgain: { topic: "audit", label: "audit.actions.again" },
  exportAI: { topic: "audit", label: "audit.actions.export" },
  exportSheet: { topic: "manual", label: "measure.answers.exportSheet" },
  importSheet: { topic: "manual", label: "measure.answers.importSheet" },
  correct: { topic: "manual", label: "measure.answers.markMentioned" },
  addQuestion: { topic: "prompts", label: "questions.add" },
  saveQuestions: { topic: "prompts", label: "common.save" },
  redraft: { topic: "prompts", label: "questions.redraft" },
  createProject: { topic: "setup", label: "projects.create" },
  saveBrand: { topic: "setup", label: "brand.save" },
  addCompetitor: { topic: "setup", label: "competitors.add" },
  saveCompetitors: { topic: "setup", label: "common.save" },
  accept: { topic: "opportunities", label: "plan.actions.accept" },
  dismiss: { topic: "opportunities", label: "plan.actions.dismiss" },
  progress: { topic: "opportunities", label: "plan.actions.markDone" },
  restore: { topic: "opportunities", label: "plan.actions.restore" },
  buildReport: { topic: "pages", label: "reports.build" },
  downloadReport: { topic: "pages", label: "reports.html" },
  searchSync: { topic: "google", label: "search.sync" },
  saveKeyword: { topic: "google", label: "search.save" },
  googleConnect: { topic: "google", label: "google.connect" },
  googleSyncNow: { topic: "google", label: "google.syncNow" },
  saveProperties: { topic: "google", label: "google.saveProperties" },
  googleDisconnect: { topic: "google", label: "google.disconnect" },
  testConnection: { topic: "providers", label: "providers.testConnection" },
  saveConnect: { topic: "providers", label: "providers.saveConnect" },
  advanced: { topic: "providers", label: "providers.advanced" },
  addUser: { topic: "access", label: "users.add" },
  userAccess: { topic: "access", label: "users.projectAccess" },
  setPassword: { topic: "access", label: "users.setPassword" },
  disableUser: { topic: "access", label: "users.disable" },
  deleteUser: { topic: "access", label: "users.delete" },
  changePassword: { topic: "access", label: "access.changePassword" },
} satisfies Record<string, { topic: TopicId; label: Key }>;

export type TipId = keyof typeof TIPS;
export const TIP_IDS = Object.keys(TIPS) as TipId[];
