// SPDX-License-Identifier: AGPL-3.0-or-later

// Audit rule texts. Codes and English text live in internal/service/audit/issues.go.
export const zh = {
  "AI_UA_BLOCKED": {
    "title": "WAF 或 CDN 拒绝 AI 爬虫",
    "why": "浏览器能打开首页，但真实的 AI 爬虫 UA 收到 403/406，尽管 robots.txt 允许它们。",
    "fix": "在防爬虫规则里放行这些 UA，然后重新抓取。"
  },
  "BROKEN_INTERNAL_LINK": {
    "title": "站内链接指向失效地址",
    "why": "爬虫顺着链接发现页面；失效链接浪费请求，还会让目标页被隐藏。",
    "fix": "把链接改到可访问的地址，或删除它。"
  },
  "CANONICAL_MISMATCH": {
    "title": "canonical 指向了别的地址",
    "why": "信号会合并到目标地址，所以展示的不会是这个页面。",
    "fix": "确认这是有意合并；否则把 canonical 指向本页。"
  },
  "CLIENT_ERROR": {
    "title": "页面返回 4xx",
    "why": "4xx 页面不会被收录；如果还在链接或 sitemap 里，会浪费抓取。",
    "fix": "恢复页面，或从链接和 sitemap 中移除并重定向到最接近的可用页面。"
  },
  "DUPLICATE_BODY": {
    "title": "多个页面正文几乎相同",
    "why": "搜索引擎只收录重复内容中的一个版本，你想要的那个可能落选。",
    "fix": "保留一个规范版本，其余用 301 或 canonical 指向它。"
  },
  "DUPLICATE_META_DESCRIPTION": {
    "title": "多个页面描述相同",
    "why": "Google 建议每个页面使用独立的描述。",
    "fix": "为每个页面写专属描述，或删掉重复的。"
  },
  "DUPLICATE_TITLE": {
    "title": "多个页面标题相同",
    "why": "标题相同让页面难以区分，往往说明存在重复地址。",
    "fix": "每个页面写不同的标题，或用 canonical、301 合并重复页。"
  },
  "FEW_EXTERNAL_LINKS": {
    "title": "引用的来源太少",
    "why": "在 GEO-bench 实验中，标注来源是效果最强的改写方式之一。",
    "fix": "为你的论断链接出处。"
  },
  "FEW_H2": {
    "title": "小节太少",
    "why": "被引用最多的四分之一页面平均有 10.6 个标题。仅为相关性。",
    "fix": "把长内容拆成若干小节，每节回答一个问题。"
  },
  "HREFLANG_LOW": {
    "title": "多语言站点几乎没有 hreflang",
    "why": "hreflang 告诉引擎该展示哪个语言版本；没有它，各版本会互相竞争。",
    "fix": "在各语言版本之间添加相互指向的 hreflang。"
  },
  "IMAGES_MISSING_ALT": {
    "title": "图片缺少 alt 文本",
    "why": "Google 用 alt 文本理解图片，无障碍也需要它。",
    "fix": "为有信息量的图片写 alt 描述；装饰性图片用 alt=\"\"。"
  },
  "LANG_IMBALANCE": {
    "title": "各语言内容量差距很大",
    "why": "站点同时发布中英文内容，但其中一种语言的内容页少得多。",
    "fix": "先补齐薄弱的一侧，从与你的问题相关的页面开始。"
  },
  "LLMS_TXT_BROKEN_LINKS": {
    "title": "llms.txt 链接到失效或被屏蔽的页面",
    "why": "一份指向 404 或禁止访问路径的地图，比没有更糟。",
    "fix": "修复或删除失效条目。"
  },
  "LOW_CONTENT_PAGE": {
    "title": "功能页文字很少",
    "why": "登录、购物车、联系页本来就短；列出来只是为了不把它们误判为空壳页。",
    "fix": "除非这个页面需要排名，否则无需处理。"
  },
  "LOW_LIST_DENSITY": {
    "title": "几乎没有列表",
    "why": "被引用最多的四分之一页面列表密度为 0.43，最少的四分之一只有 0.05。仅为相关性。",
    "fix": "把枚举内容改成 ul/ol 列表。"
  },
  "LOW_RELEVANCE": {
    "title": "标题没用上问题里的用词",
    "why": "与查询的相关度是影响力最强的单一预测因子（r = 0.432）。",
    "fix": "在 title、H1 和 H2 里使用买家会用的词。"
  },
  "MISSING_H1": {
    "title": "内容页没有 H1",
    "why": "主标题是 Google 生成标题链接的信号之一，也向读者说明主题。",
    "fix": "添加一个说明页面主题的 H1。"
  },
  "MISSING_META_DESCRIPTION": {
    "title": "缺少 meta 描述",
    "why": "Google 可能用描述作为摘要；没有时会从正文截取。",
    "fix": "写一两句话的摘要。"
  },
  "MISSING_TITLE": {
    "title": "缺少 <title>",
    "why": "title 元素是 Google 生成标题链接的主要来源。",
    "fix": "添加独立、描述性的标题。"
  },
  "MULTIPLE_H1": {
    "title": "有多个 H1",
    "why": "通常是模板问题，例如把 logo 标成了 H1。Google 不会因此降权，列出只为清晰。",
    "fix": "只为主标题保留一个 H1。"
  },
  "NOINDEX": {
    "title": "页面设置了 noindex",
    "why": "noindex 告诉搜索引擎不要展示该页，也会让它进不了基于索引的 AI 功能。",
    "fix": "需要被发现的页面去掉 noindex。"
  },
  "NON_200_STATUS": {
    "title": "页面返回非 200 的成功或跳转码",
    "why": "爬虫对 202 和 3xx 的处理与 200 不同。",
    "fix": "直接用 200 返回规范地址。"
  },
  "NO_AUTHOR_ENTITY": {
    "title": "文章标记缺少作者",
    "why": "Google 的内容指南关心内容由谁创作；Article 标记可以说明。",
    "fix": "在 Article 标记里加上作者，并在页面显示署名。"
  },
  "NO_CANONICAL": {
    "title": "没有 canonical",
    "why": "存在重复时，没有它搜索引擎会自己挑选规范地址。",
    "fix": "添加指向自身的 rel=canonical。"
  },
  "NO_COMPARISON": {
    "title": "没有对比内容",
    "why": "在吸收数据集中，对比型页面的平均影响力高 55.3%。",
    "fix": "增加一个按同一标准对比各选项的表格。"
  },
  "NO_DATE": {
    "title": "没有可见日期",
    "why": "中文引擎研究中，时效类查询被引内容的半衰期约 39 天；日期让读者和引擎判断新鲜度。",
    "fix": "显示发布和更新日期，并加入 Article 标记。"
  },
  "NO_DEFINITION": {
    "title": "没有定义句",
    "why": "在吸收数据集中，含定义的页面平均影响力高 57.3%。",
    "fix": "开头用一句话说明主题是什么。"
  },
  "NO_HOWTO": {
    "title": "没有操作步骤",
    "why": "在吸收数据集中，操作指南类内容的平均影响力高 41.2%。",
    "fix": "在需要读者动手的地方加上编号步骤。"
  },
  "NO_JSONLD": {
    "title": "没有结构化数据",
    "why": "结构化数据帮助引擎给页面归类。Google 表示 AI 功能并不要求它。",
    "fix": "首页加 Organization，合适的页面加 Article 或 Product。"
  },
  "NO_LLMS_TXT": {
    "title": "没有 /llms.txt",
    "why": "llms.txt 是 2024 年的提案，不是标准，也没有引擎说明会读取它。成本低，收益不确定。",
    "fix": "可选：用品牌事实生成并发布 /llms.txt。"
  },
  "NO_NUMBERS": {
    "title": "具体数字太少",
    "why": "在 GEO-bench 实验中，加入统计数据是效果最强的改写之一；含数字的页面在吸收数据集中影响力高 61.6%。",
    "fix": "加入带单位、日期和出处的数字。"
  },
  "NO_QUOTABLE_PASSAGE": {
    "title": "没有可独立引用的段落",
    "why": "检索按段落而不是按页面选材。规则：至少 60 词且含数字、定义或步骤的小节。该规则为本项目自定。",
    "fix": "改写两三个关键小节，让第一句直接回答小节标题。"
  },
  "NO_SITEMAP": {
    "title": "没有 sitemap.xml",
    "why": "sitemap 告诉爬虫哪些地址重要；大型或内链较弱的站点受益最大。",
    "fix": "发布只列规范地址的 /sitemap.xml。"
  },
  "ORPHAN_PAGE": {
    "title": "没有页面链接到这里",
    "why": "只能通过 sitemap 到达的页面更难被发现，也缺少站内上下文。",
    "fix": "从相关页面添加指向它的链接。"
  },
  "PAGE_UNREACHABLE": {
    "title": "页面无法抓取",
    "why": "网络错误或超时意味着爬虫拿不到内容。",
    "fix": "检查这个地址的 DNS、TLS 和服务器日志。"
  },
  "REDIRECT_CHAIN": {
    "title": "到达页面前跳转了两次以上",
    "why": "每次跳转都增加延迟，爬虫也会放弃过长的跳转链。",
    "fix": "直接跳到最终地址，并更新站内链接。"
  },
  "ROBOTS_BLOCKS_AI": {
    "title": "robots.txt 屏蔽了 AI 爬虫",
    "why": "按 RFC 9309，匹配的 Disallow 组会阻止该爬虫抓取对应路径。",
    "fix": "添加明确放行 GPTBot、OAI-SearchBot、ClaudeBot、PerplexityBot、Google-Extended 以及你关注的地区爬虫的 User-agent 组。"
  },
  "SCHEMA_CONTENT_MISMATCH": {
    "title": "结构化数据与可见内容不符",
    "why": "Google 的规范要求标记描述的是用户看得到的内容。",
    "fix": "删除该标记，或补上对应的可见内容。"
  },
  "SERVER_ERROR": {
    "title": "服务器错误（5xx）",
    "why": "Google 说明反复出现 5xx 会降低抓取频率，甚至让地址掉出索引。",
    "fix": "修复服务器错误；页面已下线则返回 404/410。"
  },
  "SHORT_CONTENT": {
    "title": "内容页偏短",
    "why": "在引用数据集中，前四分之一页面平均 1,943 词，后四分之一只有 170 词。这是相关性，不是目标。",
    "fix": "补充读者需要的实质内容（数字、对比、步骤），不要注水。"
  },
  "SITEMAP_LOW_VALUE_URLS": {
    "title": "sitemap 里有参数、搜索或翻页地址",
    "why": "sitemap 应只列出你希望被收录的规范地址。",
    "fix": "把这些地址从 sitemap 中移除。"
  },
  "SITEMAP_NOT_DECLARED": {
    "title": "robots.txt 没有声明 sitemap",
    "why": "有了 Sitemap: 这一行，任何爬虫不用告知也能找到 sitemap。",
    "fix": "在 robots.txt 里加上 `Sitemap: https://example.com/sitemap.xml`。"
  },
  "SLOW_RESPONSE": {
    "title": "服务器响应慢",
    "why": "HTML 超过 3 秒才返回；爬虫在慢站点上抓取的页面更少。3 秒阈值为本项目规则。",
    "fix": "在边缘缓存 HTML，或排查慢的路由。"
  },
  "SPA_SHELL": {
    "title": "静态 HTML 里没有内容",
    "why": "Google 在延迟队列里渲染 JavaScript，很多 AI 爬虫只读静态 HTML，所以纯前端渲染的页面看起来是空的。",
    "fix": "内容页使用服务端渲染或预渲染。"
  },
  "TITLE_TOO_LONG": {
    "title": "标题过长",
    "why": "Google 没有长度上限，但会按显示宽度截断。此处阈值约为 60 个拉丁字符或 32 个中日韩字符。",
    "fix": "把关键词放在最前面。"
  },
  "XROBOTS_NOINDEX": {
    "title": "X-Robots-Tag 响应头含 noindex",
    "why": "该响应头与 meta 标签效果相同，但在源码里看不到，常由 CDN 注入。",
    "fix": "在服务器或 CDN 上删除这条响应头规则。"
  }
};
