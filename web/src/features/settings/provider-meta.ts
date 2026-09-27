// SPDX-License-Identifier: AGPL-3.0-or-later

import openai from "../../assets/providers/openai.svg";
import claude from "../../assets/providers/claude.svg";
import gemini from "../../assets/providers/gemini.svg";
import grok from "../../assets/providers/grok.svg";
import perplexity from "../../assets/providers/perplexity.svg";
import deepseek from "../../assets/providers/deepseek.svg";
import doubao from "../../assets/providers/doubao.svg";
import kimi from "../../assets/providers/kimi.svg";
import glm from "../../assets/providers/glm.svg";
import minimax from "../../assets/providers/minimax.svg";

// Logos from LobeHub Icons (MIT), see assets/providers/LICENSE.md.
export const PROVIDER_LOGO: Record<string, string> = {
  openai, claude, gemini, grok, perplexity, deepseek, doubao, kimi, glm, minimax,
};

// Display order: the most used engines first.
export const PROVIDER_ORDER = ["openai", "claude", "gemini", "perplexity", "grok", "deepseek", "doubao", "kimi", "glm", "minimax"];

// Short display names; the backend names carry notes like "(Ark API)".
export const PROVIDER_LABEL: Record<string, string> = {
  openai: "OpenAI (ChatGPT)", claude: "Claude", gemini: "Gemini", perplexity: "Perplexity", grok: "Grok",
  deepseek: "DeepSeek", doubao: "Doubao", kimi: "Kimi", glm: "Zhipu GLM", minimax: "MiniMax",
};

// Display metadata for model providers: brand color and where to get a key.
export const PROVIDER_COLOR: Record<string, string> = {
  glm: "#1aae39", doubao: "#0075de", deepseek: "#4d6bfe", kimi: "#31302e",
  minimax: "#dd5b00", gemini: "#4285f4", openai: "#10a37f", claude: "#d97757",
  grok: "#111827", perplexity: "#20808d",
};

export const PROVIDER_KEY_URL: Record<string, string> = {
  glm: "https://open.bigmodel.cn/usercenter/proj-apikey/apikeys",
  doubao: "https://console.volcengine.com/ark/region:ark+cn-beijing/apikey",
  deepseek: "https://platform.deepseek.com/api_keys",
  kimi: "https://platform.moonshot.cn/console/api-keys",
  minimax: "https://platform.minimaxi.com/user-center/basic-information/interface-key",
  gemini: "https://aistudio.google.com/apikey",
  openai: "https://platform.openai.com/api-keys",
  claude: "https://console.anthropic.com/settings/keys",
  grok: "https://console.x.ai/",
  perplexity: "https://www.perplexity.ai/settings/api",
};
