// SPDX-License-Identifier: AGPL-3.0-or-later

import { Formula, FX, Table, Tip, Warn, type HelpDoc } from "../kit";

export const zh: HelpDoc = {
  groups: { start: "入门", read: "读懂结果", act: "采取行动", connect: "接入", help: "帮助" },
  topics: {
    start: { group: "start", label: "从这里开始", keys: "概览 闭环 是什么 快速上手 第一次" },
    setup: { group: "start", label: "建立项目", keys: "项目 创建 清单 密钥 品牌 竞品 问题 排程 引导" },
    prompts: { group: "start", label: "写好问题", keys: "问题 提示词 分组 品牌词 非品牌 品类 泛泛 零" },
    terms: { group: "read", label: "术语表", keys: "术语 可见性 认知度 声量份额 引用 扩展搜索 周期 采样 访问方式" },
    numbers: { group: "read", label: "数字怎么算", keys: "公式 区间 置信 wilson newcombe 样本少 未测 api 网页 失败 稳定度 声量" },
    pages: { group: "read", label: "逐页说明", keys: "概览 可见性 声量 引用 扩展搜索 回答 搜索 报告 导航 菜单" },
    opportunities: { group: "act", label: "行动计划", keys: "机会 采纳 忽略 优先级 优先处理 已验收 回退 状态" },
    audit: { group: "act", label: "站点体检", keys: "审计 就绪度 层级 访问 发现 理解 引用 被阻塞 严重度 依据 问题 导出" },
    manual: { group: "act", label: "人工采样", keys: "表格 导入 导出 chatgpt 网页版 谷歌 ai 概览 百度 没有 api" },
    providers: { group: "connect", label: "模型服务商", keys: "api 密钥 引擎 中转 端点 模型 联网搜索 成本" },
    google: { group: "connect", label: "Google 搜索与 GA4", keys: "oauth 客户端 服务账号 search console ga4 媒体资源 同步 回调" },
    access: { group: "connect", label: "用户与权限", keys: "用户 角色 管理员 成员 权限 查看 编辑 密码 共享 邀请 团队 登录" },
    schedule: { group: "connect", label: "计划与运行", keys: "周期 每天 每日运行次数 token 成本 重试 任务 中断" },
    troubleshooting: { group: "help", label: "常见问题", keys: "问题 错误 0% 未测量 失败 卡住 重新连接 空白 spa 被拦截" },
    limits: { group: "help", label: "它不做什么", keys: "限制 承诺 保证 关键词 外链" },
  },
  body: ({ page, topic, n }) => ({
    start: (
      <>
        <h2>从这里开始</h2>
        <p>craftsail-growth 只回答一个问题：<strong>当人们向 AI 引擎询问你所在的品类时，回答里有没有提到你、引用你，并且情况是否在变好？</strong></p>
        <p>它按周期循环运行：</p>
        <ol>
          <li><strong>测量。</strong>把你的问题向每个引擎问若干次，记录每一条回答：提到了谁、先后顺序、引用了哪些来源。</li>
          <li><strong>诊断。</strong>抓取并体检你的网站，分四层检查：引擎能不能抓到、找得到、看得懂、引用得了。</li>
          <li><strong>行动。</strong>体检发现、引用缺口、搜索数据和指标下滑，汇成一份排好序的行动计划。你接受准备做的条目。</li>
          <li><strong>验证。</strong>下一个周期按规则检查每条已接受的行动，通过则标为已验收，之后又退回则标为已回退。</li>
        </ol>
        <p>侧边栏也按这个闭环排列：<strong>{n("nav.sections.measure")}</strong>负责测量，<strong>{n("nav.sections.improve")}</strong>负责诊断和行动，<strong>{n("nav.sections.project")}</strong>是当前项目的设置，<strong>{n("nav.sections.workspace")}</strong>是所有项目共用的配置。</p>
        <h3>最初的 15 分钟</h3>
        <ol>
          <li>先在{page("settings/providers", "nav.providers")}里至少添加一个引擎密钥。请在创建项目<strong>之前</strong>完成，因为起草品牌信息和问题也要用到它。</li>
          <li>在{page("settings/projects", "nav.projects")}里创建项目，并勾选“立即运行第一个周期”。</li>
          <li>运行期间阅读{topic("prompts", "写好问题")}，然后到{page("settings/questions", "nav.questions")}检查自动起草的问题。</li>
          <li>运行结束后打开{page("overview", "nav.overview")}，再看{page("opportunities", "nav.actionPlan")}。</li>
        </ol>
        <p>之后，管理员可以在{page("settings/users", "nav.users")}里邀请同事或客户，见{topic("access", "用户与权限")}。</p>
        <Tip>第一个周期只能告诉你现状，价值来自持续重复：在{page("settings/schedule", "nav.schedule")}里开启定时运行。</Tip>
      </>
    ),
    setup: (
      <>
        <h2>建立项目</h2>
        <p>一个项目对应一个品牌及其网站。问题、报告等其他内容都属于某个项目。在每个页面顶部面包屑开头的项目名处切换项目。</p>
        <Table head={["步骤", "位置", "为什么重要"]} rows={[
          ["1. 接入引擎", page("settings/providers", "nav.providers"), "没有密钥就不会采样，问题也只能退回到通用模板。"],
          ["2. 创建项目", page("settings/projects", "nav.projects"), "填写网站地址。没有自有网站时，勾选“没有自己的网站”并填写品牌名。"],
          ["3. 检查品牌信息", page("settings/brand", "nav.brand"), "一张简短的表单：名称和别名、一句话定义、品类、受众、带来源的关键数字、适合与不适合的场景。内容从网站起草；空着的字段表示网站上没找到。生成的 llms.txt 和 JSON-LD 都来自这里。"],
          ["4. 检查竞品", page("settings/competitors", "nav.competitors"), "三到六个真实对手，附上别名。声量份额和排名都按这份名单计算。"],
          ["5. 修改问题", page("settings/questions", "nav.questions"), <>最重要的一步。见{topic("prompts", "写好问题")}。</>],
          ["6. 接入 Google（可选）", page("settings/google", "nav.google"), "增加搜索点击、查询词和会话数据，以及行动计划里的搜索类条目。"],
          ["7. 排程", page("settings/schedule", "nav.schedule"), "每周或每天运行一个周期，趋势和验证才有意义。"],
          ["8. 邀请他人（可选）", page("settings/users", "nav.users"), "添加同事或客户，并按项目设置查看或编辑权限。"],
        ]} />
        <Warn>先创建了项目、后添加密钥？那么它的问题是通用模板。添加密钥后打开{page("settings/questions", "nav.questions")}，点击<strong>用 AI 重新起草</strong>。它会用根据网站写出的草稿替换问题和竞品，完成后请把两者都检查一遍。</Warn>
      </>
    ),
    prompts: (
      <>
        <h2>写好问题</h2>
        <p>问题（也叫提示词）就是工具向每个引擎提问的内容。它们决定了测量什么；问题写得差，其他所有数字都没有意义。在{page("settings/questions", "nav.questions")}里编辑。</p>
        <h3>像买家一样说出品类</h3>
        <Table head={["差", "更好"]} rows={[
          ["这个品类里最好的工具有哪些？", "把 Word 文档转成 PDF，最好用的命令行工具有哪些？"],
          ["小团队该用什么？", "小型 SaaS 团队应该选哪个文档自动化 API？"],
          ["第一天该从哪里开始？", "不装 Microsoft Office，怎么用 Python 生成 Excel 报表？"],
        ]} />
        <p>问题里不说出品类，引擎就会反问你指的是什么，而不是给出推荐。这样不管你怎么改网站，可见性都会停在 0%。</p>
        <h3>分组</h3>
        <p>每个问题属于一个分组。推荐、比较、替代和价格属于<strong>买家</strong>问题。风险和使用场景用来解释品类。<strong>品牌验证</strong>会故意点出你的品牌名。</p>
        <h3>品牌词与非品牌词</h3>
        <p>包含品牌名、别名或域名的问题属于<strong>品牌词问题</strong>，“系统标签”一列会标出来。品牌词问题几乎必然会提到你，所以单独统计为<strong>认知度</strong>，不计入可见性。问题应以非品牌的买家问题为主。</p>
        <h3>实用规则</h3>
        <ul>
          <li>用买家的语言写问题。每个启用的问题都会在每个已接入的引擎上提问。</li>
          <li>从 10 到 20 个问题开始。每个问题每次运行在每个引擎上消耗一次调用。</li>
          <li>用自己的标签（比如产品线）来筛选 AI 可见性的各个页面。</li>
          <li>不想要的问题请停用而不是删除，这样能保留历史数据。</li>
          <li><strong>用 AI 重新起草</strong>会让已接入的模型根据你的网站重新写一份问题列表，并替换当前的问题和竞品。</li>
        </ul>
      </>
    ),
    terms: (
      <>
        <h2>术语表</h2>
        <Table head={["术语", "含义"]} rows={[
          ["引擎 / 模型服务商", "工具去提问的 AI 产品，比如 DeepSeek、OpenAI 或 Perplexity。"],
          ["问题（提示词）", "问题列表中的一条，会在每个引擎上提问。"],
          ["运行 / 回答", "一个引擎对一个问题回答一次。引擎的回答会变化，所以每个问题每天会问多次。"],
          ["周期", "完整跑一遍：抓取、体检、采样、同步 Google、验证、更新行动计划、生成报告。"],
          ["访问方式（API / 网页）", "通过引擎 API 得到的回答，与在网页或 App 里手工收集的回答分开统计，因为两者引用的来源不同。"],
          ["可见性", "在非品牌问题的成功回答中，提到你品牌的比例。"],
          ["认知度", "同样的比例，但只看品牌词问题。它说明引擎是否认识你，而不是是否推荐你。"],
          ["声量份额", "你的提及次数 ÷（你的提及次数 + 竞品提及次数）。"],
          ["引用", "引擎在回答中附上的来源网址。"],
          ["搜索扩展（Query fan-out）", "引擎为回答问题实际执行的网页搜索，与问题原文对比。"],
          ["建议", "行动计划中的一条，来自体检、引用、搜索数据或指标下滑。"],
          ["行动", "你接受的建议。它带有基线，以及一条用来检查是否奏效的规则。"],
          ["就绪度层级", "访问、发现、理解、引用：页面需要满足的四件事，也是修复顺序。"],
        ]} />
      </>
    ),
    numbers: (
      <>
        <h2>数字怎么算</h2>
        <p><strong>可见性</strong>是在非品牌问题的成功回答中提到品牌的比例。点出品牌名的问题单独统计为<strong>认知度</strong>。</p>
        <p>每次运行的回答都不一样，所以每个比例都是估计值。旁边会显示 <strong>95% 区间</strong>（例如“CI 0–22%”）和回答数（n）。请看区间而不只是数字：14 个回答中 0% 的意思是“大概率低于 22%”，而不是“从不”。</p>
        <ul>
          <li><strong>样本少</strong>：回答不足 30 个。只能当作粗略信号。</li>
          <li><strong>未测</strong>或 <strong>—</strong>：还没有可统计的内容，比如没有任何竞品被提到。它不会显示成 0%。</li>
          <li><strong>失败的运行</strong>（超时、额度不足）不计入比例，单独统计。可以在{page("settings/schedule", "nav.schedule")}里重试。</li>
          <li><strong>API 和网页</strong>的回答从不相加。两者都有时，筛选栏里会出现切换开关。</li>
          <li>周期之间的<strong>上升或下降</strong>只有在变化大于噪声时才会标出（95% Newcombe 区间不包含零）。</li>
        </ul>
        <h3>公式</h3>
        <p>公式统一用英文记号书写，下面逐一说明含义。</p>
        <p><strong>可见性。</strong>在成功返回的回答中，有多少提到了你。问题里本身带有你品牌名的不计入，因为它们几乎必然会提到你。</p>
        <Formula>{FX.visibility}</Formula>
        <p><strong>认知度。</strong>同样的统计，但只看点出了你品牌名的问题。它说明引擎是否认识你，而不是是否推荐你。</p>
        <Formula>{FX.recognition}</Formula>
        <p><strong>第一名与前三名。</strong>被最先提到比最后提到更有价值，所以也按首次出现的顺序统计。</p>
        <Formula>{FX.top}</Formula>
        <p><strong>声量份额。</strong>在所有品牌提及（你加上竞品）中你所占的比例。一次提及都没有时显示为未测。</p>
        <Formula>{FX.sov}</Formula>
        <p><strong>引用你域名的回答</strong>与<strong>引用份额。</strong>不同引擎之间不比较引用数量：有的引擎每条回答附带的来源比别的多得多。</p>
        <Formula>{FX.ownCite}</Formula>
        <h3>每个比例旁边的区间</h3>
        <p>引擎每次的回答都不一样，所以比例只是估计值。95% Wilson 区间给出真实比例大概率所在的范围；回答很少时它依然合理，而简单的 ± 会算出低于 0% 或高于 100% 的结果。</p>
        <Formula>{FX.wilson}</Formula>
        <p>例子：</p>
        <Formula>{FX.wilsonExample}</Formula>
        <h3>上升、下降还是噪声</h3>
        <p>两个周期的比较用差值的区间来判断，而不是凭肉眼。只有整个区间都落在零的同一侧，才算变化。</p>
        <Formula>{FX.change}</Formula>
        <Formula>{FX.changeExample}</Formula>
        <h3>引用稳定度</h3>
        <p>被引用的网站每天变化有多大。分数低说明来源还对新网站开放；分数高说明引擎总在引用同一批网站。它只用于给建议排序。</p>
        <Formula>{FX.stability}</Formula>
      </>
    ),
    pages: (
      <>
        <h2>逐页说明</h2>
        <p>{n("nav.audit")}、{n("nav.search")}这类有多个视图的页面，会在标题下方以标签页展示。</p>
        <Table head={["页面", "回答什么问题", "下一步"]} rows={[
          [page("overview", "nav.overview"), "我们现在处于什么位置？可见性、声量份额、自有引用、就绪度、最重要的行动。", "打开最弱的那一项。"],
          [page("ai/visibility", "nav.visibility"), "哪些问题提到了我们，在哪个引擎上，随时间怎么变化。", "打开一个问题阅读它的回答。"],
          [page("ai/share-of-voice", "nav.sov"), "谁代替我们被提到了。", <>在{n("nav.competitors")}里补上缺失的对手。</>],
          [page("ai/citations", "nav.citations"), "引擎依赖哪些来源（按来源类型和页面类型）；哪些来源引用了对手却没有引用我们。", <>在{n("nav.actionPlan")}里针对这些来源行动。</>],
          [page("ai/fan-out", "nav.fanout"), "引擎实际搜了什么。只有带联网搜索的引擎会报告。", "把新增的词用到页面和问题里。"],
          [page("ai/answers", "nav.answers"), "每一条原始回答，附引用和搜索。", "纠正误判；导入人工采样。"],
          [page("opportunities", "nav.actionPlan"), "接下来做什么（已排序），以及是否奏效。", "接受、开始、标记完成。"],
          [page("audit", "nav.audit"), "引擎能否抓取、找到、理解并引用网站。", "修复第一个未通过的层级。"],
          [page("search", "nav.search"), "Google 点击、展示、会话、查询词和着陆页。", "与 AI 可见性对照。"],
          [page("reports", "nav.reports"), "每个周期一份可分享的页面。", "下载 HTML 或 Markdown。"],
        ]} />
        <p>如果页面标题旁显示<strong>{n("access.viewOnly")}</strong>，说明你对该项目只有查看权限，所有会修改数据的按钮都会隐藏。见{topic("access", "用户与权限")}。</p>
        <Tip>筛选（模型、标签、时间范围）位于每个 AI 可见性页面顶部。它们保存在页面地址里，所以可以收藏或分享筛选后的视图。</Tip>
      </>
    ),
    opportunities: (
      <>
        <h2>行动计划</h2>
        <p>{page("opportunities", "nav.actionPlan")}是唯一的待办清单。建议来自四个来源，分为<strong>优先处理</strong>、<strong>值得做</strong>和<strong>有空再做</strong>三组：</p>
        <Table head={["来源", "例子", "分组"]} rows={[
          ["站点体检", "WAF 拦截了 AI 爬虫；没有结构化数据", "严重 → 优先处理，警告 → 值得做，提示 → 有空再做。观察性规则不会进入优先处理。"],
          ["AI 引用", "某个问题下对手被引用而你没有", "值得做；来源难以进入时为有空再做"],
          ["搜索", "某个查询词排在前三名之外一点；点击率低", "有空再做"],
          ["指标变化", "可见性下降幅度超过噪声", "优先处理"],
        ]} />
        <h3>标签页与生命周期</h3>
        <ol>
          <li><strong>建议</strong>：接受一条，或在不适用时忽略。已忽略的条目可以在对应标签页里恢复。</li>
          <li><strong>进行中</strong>：接受时会记录基线，比如“影响 3 个页面”。先点开始，完成后点标记完成。</li>
          <li>下一个周期检查通过后变为<strong>已验收</strong>。已验收的行动之后又不达标时变为<strong>已回退</strong>，并回到进行中。</li>
        </ol>
        <h3>怎么检查“完成”</h3>
        <ul>
          <li>体检类行动：最新一次体检中该问题消失即通过。</li>
          <li>引用类行动：该问题的可见性上升且超过噪声即通过。</li>
          <li>可见性下滑：可见性回到基线的噪声范围内即通过。</li>
          <li>搜索类行动由你自己判断：做完后标记完成。</li>
        </ul>
        <h3>每类建议什么时候出现</h3>
        <p><strong>AI 引用。</strong>某个问题下引擎引用了竞品网站、却从未引用你的网站时，就会出现引用类建议：</p>
        <Formula>{FX.gap}</Formula>
        <p><strong>搜索。</strong>来自 Google Search Console，统计截至三天前的 28 天（Google 数据有延迟），并与再往前的 28 天比较：</p>
        <Formula>{FX.striking}</Formula>
        <p><strong>指标变化。</strong>最近 30 天的可见性与之前 30 天比较，使用{topic("numbers", "数字怎么算")}里“上升、下降还是噪声”的规则。</p>
        <h3>行动怎么验收</h3>
        <Formula>{FX.verify}</Formula>
        <p>品牌词搜索（搜你自己的名字）不会列出：搜你的人已经找到你了。</p>
        <Tip>每条都会显示“怎么修”和“完成标准”。CLI 里有同样的清单：<code>craftsail-growth opportunities --slug &lt;project&gt;</code>。</Tip>
      </>
    ),
    audit: (
      <>
        <h2>站点体检</h2>
        <p>体检会对抓取到的页面运行 47 条规则，并按修复顺序分成四层：</p>
        <Table head={["层级", "问题", "典型发现"]} rows={[
          ["访问", "爬虫能抓到内容吗？", "robots.txt 屏蔽 AI 爬虫、CDN 拒绝 AI 的 User-Agent、noindex、不执行 JavaScript 时页面为空"],
          ["发现", "能找到所有网址吗？", "没有 sitemap、robots.txt 未声明 sitemap、没有 llms.txt、死链"],
          ["理解", "能看出每个页面是什么吗？", "没有结构化数据、schema 与页面内容矛盾、标题过长"],
          ["引用", "有值得引用的内容吗？", "没有定义句、数字少、没有对比、没有可直接引用的段落"],
        ]} />
        <p>某一层未通过时，后面各层的发现会标为<strong>被阻塞</strong>：现在修它们并不会改变引擎看到的内容。</p>
        <p>每条规则都标有<strong>依据</strong>：标准、官方文档、实验、观察性或经验规则。观察性规则和经验规则只是建议，不会被判为严重。</p>
        <h3>层级状态怎么判定</h3>
        <Formula>{FX.layers}</Formula>
        <p>{n("nav.overview")}里的就绪度数字（例如“1 / 4”）只统计完全没有问题的层级；只有警告的层级不计入。</p>
        <h3>页面怎么打分</h3>
        <p>{n("nav.tabs.pages")}标签页按六个部分给每个页面打 0–100 分。这些阈值描述的是公开数据集中被引擎引用过的页面，是信号而不是保证。字数按抓取到的 HTML 统计，不执行 JavaScript。</p>
        <Formula>{FX.pageScore}</Formula>
        <Formula>{FX.pageParts}</Formula>
        <ul>
          <li><strong>{n("nav.tabs.readiness")}</strong>标签页显示各层级；<strong>{n("nav.tabs.issues")}</strong>列出每条规则及受影响的页面；<strong>{n("nav.tabs.pages")}</strong>给每个页面打分。</li>
          <li><strong>导出给 AI</strong>会把体检结果下载为 Markdown，方便交给编程助手处理。</li>
          <li>修改网站后请重新抓取；体检只看最近一次抓取。</li>
        </ul>
      </>
    ),
    manual: (
      <>
        <h2>人工采样</h2>
        <p>很多人实际在用的产品没有 API：带搜索的 ChatGPT 网页版、Claude 网页版、Google AI 概览、百度 AI 搜索、豆包 App、秘塔、纳米 AI。你仍然可以手工测量它们。</p>
        <ol>
          <li>在{page("ai/answers", "nav.answers")}页面点击<strong>导出采样表</strong>。</li>
          <li>对每个问题，在无痕窗口中打开产品，新建对话，提问，并把完整回答粘贴进表格，即使没有提到你也要贴。</li>
          <li>把填好的表格粘贴回来，点击<strong>导入填好的采样表</strong>。</li>
        </ol>
        <Warn>不要用日常账号采样：引擎会对回答做个性化。请使用无痕窗口或专门用于采样的浏览器配置。人工回答计为“网页”，不会与 API 回答混在一起。</Warn>
        <p>CLI 也能做到：<code>craftsail-growth sample-sheet --slug &lt;project&gt; --intent buyer --limit 20</code> 和 <code>craftsail-growth sample-import</code>。</p>
      </>
    ),
    providers: (
      <>
        <h2>模型服务商</h2>
        <p>{page("settings/providers", "nav.providers")}列出了每个引擎及其状态。选择一个，粘贴密钥，点<strong>测试连接</strong>，然后保存。没有密钥的引擎会被跳过。</p>
        <Table head={["类型", "引擎", "测量的是什么"]} rows={[
          ["联网搜索", "Perplexity；开启了方舟内容插件的豆包", "引擎今天能搜到并引用什么。最接近用户实际使用的产品。"],
          ["凭记忆回答", "DeepSeek、Kimi、GLM、MiniMax、OpenAI、Claude、Gemini、Grok（视配置而定）", "模型已经知道的关于你的内容。变化较慢，随模型版本更新。"],
        ]} />
        <ul>
          <li><strong>高级</strong>里可以更换模型，或指向中转端点。中转服务需要能识别的模型名。</li>
          <li>密钥保存在本服务器的 <code>config/default.toml</code> 中，除了调用对应服务商外不会离开服务器。</li>
          <li>{n("nav.schedule")}会显示每次运行的预估和实际 token。</li>
        </ul>
        <h3>采样的成本</h3>
        <p>每个启用的问题都会在每个有密钥的引擎上提问，每天的次数由你设置。token 数是用于规划的估计值；引擎返回实际用量时，会显示在旁边。</p>
        <Formula>{FX.calls}</Formula>
        <Formula>{FX.callsExample}</Formula>
      </>
    ),
    google: (
      <>
        <h2>Google 搜索与 GA4</h2>
        <p>可选。它会把 Search Console 的点击、展示和查询词以及 GA4 会话数据加入{n("nav.search")}，并在行动计划中加入搜索类条目。AI 可见性和体检不依赖它。</p>
        <h3>第 1 步：连接 Google 账号</h3>
        <p>本服务器需要配置一次自己的 <strong>OAuth 客户端</strong>。它是这个应用在 Google 那里的身份：点击“连接 Google”时，Google 会显示该应用的授权页，并把结果发回回调地址。自托管实例无法共用一个客户端，因为每台服务器的地址不同。</p>
        <ol>
          <li>在 Google Cloud 中启用 Search Console API、Google Analytics Data API 和 Analytics Admin API。</li>
          <li>创建类型为“Web 应用”的 OAuth 客户端，并原样添加页面上显示的回调地址。</li>
          <li>粘贴客户端 ID 和密钥，保存，然后点击“连接 Google”。</li>
        </ol>
        <p>不想在浏览器中登录？可以在“高级”里改用<strong>服务账号</strong>，把它的邮箱添加到 Search Console 资源中，并在 GA4 媒体资源上设为查看者。</p>
        <h3>第 2 步：选择资源</h3>
        <p>为这个项目选择 Search Console 资源和 GA4 媒体资源，然后点<strong>立即同步</strong>。每张卡片都会显示状态和最后导入的日期。</p>
        <h3>搜索数字怎么计算</h3>
        <p>总数来自 Google 的每日汇总。查询词和页面明细不含匿名查询，所以加起来总是更少；差额会显示出来，不会被补齐。</p>
        <Formula>{FX.search}</Formula>
        <Warn>授权页面处于“测试”状态时，Google 大约 7 天后会让登录过期，状态变为“需要重新登录”。请发布应用，或到时重新连接。已导入的数据会保留。</Warn>
      </>
    ),
    schedule: (
      <>
        <h2>计划与运行</h2>
        <p>一个<strong>周期</strong>包含七步：抓取、体检、采样、同步 Google、验收事项、更新行动计划、生成报告。如果还没有问题，它会先起草品牌信息和问题。</p>
        <ul>
          <li><strong>运行周期</strong>可选每天、每周、每两周或每 30 天。服务器每 30 分钟检查一次，必须保持运行，排程才会触发。</li>
          <li><strong>每个问题每个引擎每天的运行次数</strong>（1–10，默认 3）。次数越多区间越窄，成本也越高。</li>
          <li>各个按钮可以单独运行某一步：立即采样、同步 Google、抓取网站、体检网站、验收事项。</li>
          <li><strong>采样记录</strong>列出每一批的计划、成功、失败调用数和 token。<strong>重试失败项</strong>只会重问失败的调用。</li>
          <li>每个项目同一时间只运行一个任务。显示为<em>被中断</em>的任务是被服务器重启打断的，重新启动即可。</li>
        </ul>
      </>
    ),
    access: (
      <>
        <h2>用户与权限</h2>
        <p>管理员在{page("settings/users", "nav.users")}里添加用户，并为每个项目选择这个人可以查看、可以编辑，还是完全看不到。角色只有两种：</p>
        <ul>
          <li><strong>管理员</strong>：所有项目，以及工作区：项目、模型服务商、Google 和用户。</li>
          <li><strong>成员</strong>：只能看到共享给他的项目，每个项目的权限是<strong>{n("users.access.view")}</strong>或<strong>{n("users.access.edit")}</strong>。</li>
        </ul>
        <Table head={["", n("users.access.view"), n("users.access.edit"), n("users.roles.admin")]} rows={[
          ["阅读所有页面，下载报告和导出文件", "✓", "✓", "✓"],
          ["修改品牌、竞品、问题库和计划设置", "", "✓", "✓"],
          ["运行抓取、体检和采样；接受和完成行动；纠正回答", "", "✓", "✓"],
          ["创建项目；模型服务商；Google；用户管理", "", "", "✓"],
        ]} />
        <h3>邀请一个人</h3>
        <ol>
          <li>打开{page("settings/users", "nav.users")}，在<strong>{n("users.add")}</strong>里填写用户名（可以用邮箱）和至少 12 个字符的密码。</li>
          <li>选中新用户，把每个项目设为{n("users.access.none")}、{n("users.access.view")}或{n("users.access.edit")}，立即生效。</li>
          <li>把本系统的地址、用户名和密码发给对方。对方可以在账号菜单里修改密码。</li>
        </ol>
        <h3>需要知道的</h3>
        <ul>
          <li>没有共享给某人的项目，对他完全不可见，连项目名称都看不到。</li>
          <li>只有查看权限的项目，页面标题旁会显示<strong>{n("access.viewOnly")}</strong>，所有会修改数据的按钮都会隐藏。</li>
          <li>在侧边栏底部的账号菜单里点<strong>{n("access.changePassword")}</strong>可以修改自己的密码，修改后其他设备上的登录会失效。</li>
          <li>重置某人的密码、停用或删除他，都会让他在所有设备上立即退出。删除用户不会删除任何项目数据。</li>
          <li>系统始终至少保留一个启用的管理员；最后一个管理员不能被降级、停用或删除。</li>
          <li>从原来的单账号升级：原账号会成为第一个管理员，所有人需要重新登录一次。</li>
          <li>无法登录？在服务器上运行 <code>craftsail-growth user passwd &lt;用户名&gt;</code> 并输入新密码（输入时会显示出来；想隐藏可以用管道：<code>printf '%s\n' "$PASS" | craftsail-growth user passwd &lt;用户名&gt;</code>）；<code>craftsail-growth user add &lt;用户名&gt; --admin</code> 可以再添加一个管理员。</li>
          <li>登录期间如果管理员改了你的权限，刷新页面即可看到新的项目列表。</li>
          <li>服务器配置里的 API token 相当于管理员，供脚本和命令行使用。</li>
        </ul>
      </>
    ),
    troubleshooting: (
      <>
        <h2>常见问题</h2>
        <Table head={["你看到", "可能原因", "怎么做"]} rows={[
          ["所有问题的可见性都是 0%", "问题没有说出品类，引擎会反问你指的是什么。在没有引擎密钥时创建的项目经常这样。", <>重写问题（{topic("prompts", "指南")}），或接入模型后在{n("nav.questions")}里点“用 AI 重新起草”。</>],
          ["“未测”或 —", "还没有可统计的内容，比如没有提到任何竞品。", "添加竞品；多采样几次。"],
          ["“样本少”", "时间范围内回答不足 30 个。", "提高每日运行次数，或放宽时间范围。"],
          ["大量运行失败", "密钥错误、额度不足，或中转服务不认识该模型。", "测试服务商连接；在“高级”里检查模型名；重试失败项。"],
          ["完全没有回答", "没有接入引擎，或从未运行过采样。", "接入服务商，然后点立即采样。"],
          ["访问层未通过：AI_UA_BLOCKED", "CDN 或 WAF 拒绝了 AI 爬虫的 User-Agent，而浏览器访问正常。", "在 CDN 的爬虫设置中放行 GPTBot、ClaudeBot、PerplexityBot 等。"],
          ["页面几乎没有文字", "网站靠 JavaScript 渲染，爬虫只看到空壳。", "对关键页面做服务端渲染或预渲染。"],
          ["Google 显示“需要重新登录”", "刷新令牌过期（测试模式）或被撤销。", <>在{n("nav.google")}里重新连接。</>],
          ["搜索总数 ≠ 查询词之和", "Google 在明细报表中隐藏了匿名查询。", "属于正常现象。总数来自 Google 的每日汇总。"],
          ["任务一直显示“运行中”", "仍在执行；问题多的周期需要一段时间。", <>在{n("nav.schedule")}里观察；只有日志停止更新时才重启服务器。</>],
          ["列表里找不到某个项目", "该项目没有共享给你。", <>请管理员在{n("nav.users")}里给你该项目的查看或编辑权限。</>],
          [<>没有保存、运行按钮，标题旁显示{n("access.viewOnly")}</>, "你对该项目只有查看权限。", "请管理员给你编辑权限。"],
          ["管理员改了设置后被退出登录", "你的密码被重置，或账号被停用。", "用新密码登录，或联系管理员。"],
          ["部分提示或报告是英文", "服务器消息、任务日志和报告正文目前只有英文。", "目前属于正常现象。"],
        ]} />
      </>
    ),
    limits: (
      <>
        <h2>它不做什么</h2>
        <ul>
          <li><strong>不承诺引用。</strong>没有人能保证引擎一定会引用某个页面。本工具负责测量、建议和验证。</li>
          <li><strong>不是排名。</strong>比例是重复采样得出的估计值，并附带不确定性。</li>
          <li><strong>不做关键词研究或外链。</strong>没有接入付费 SEO 数据源。</li>
          <li><strong>不替你发布。</strong>它会起草 llms.txt、JSON-LD 等修复片段供你审阅，但不会改动你的网站。</li>
          <li><strong>自托管，权限简单。</strong>只有管理员和成员两种角色，成员按项目分配查看或编辑权限；没有团队和单点登录。你的密钥和数据都留在本服务器上。</li>
        </ul>
      </>
    ),
  }),
  buttons: ({ n, page }) => ({
    indexing: <>{n("tips.indexing")}</>,
    serve: <>在建好项目或一次改动较多之后使用。进度和日志在{page("settings/schedule", "nav.schedule")}的任务列表里；每个项目同一时间只运行一个任务。开启定期运行后会自动执行。</>,
    sample: <>只跑一轮。想让区间更窄，就提高“每个问题每个引擎每天采样次数”并让定期运行去完成。每次调用都消耗 token，成本公式见{n("nav.providers")}。</>,
    syncGoogle: <>需要先在{page("settings/google", "nav.google")}里连接 Google 并选好资源。Google 的数据大约延迟三天。</>,
    crawl: <>只抓取页面，不会更新体检结果。之后再运行体检，或直接在{page("audit", "nav.audit")}里用“{n("audit.actions.crawl")}”。</>,
    audit: <>不抓取页面，所以很快。如果改过网站，要先抓取，否则体检看到的还是旧页面。</>,
    verify: <>事项满足规则就变为已验收；已验收的事项之后又不达标会变为已回退。验收规则见{n("nav.actionPlan")}。</>,
    scheduleSave: <>服务器每 30 分钟检查一次是否该运行，所以它必须保持运行。选“关闭”会停止自动运行，按钮仍然可用。</>,
    retry: <>只在有失败的批次上显示。先解决原因（密钥、额度、模型名），否则同样的调用还会失败。</>,
    stop: <>停止前已保存的回答和页面会保留。显示为被中断的任务是被服务器重启打断的，可以重新启动。</>,
    auditCrawl: <>比“{n("audit.actions.again")}”慢，因为要重新抓取每个页面。</>,
    auditAgain: <>适合在规则更新后刷新结果；在下次抓取之前，它看不到网站上的改动。</>,
    exportAI: <>文件列出每条未通过的规则、依据、受影响的页面和修复方法。把它和网站代码一起交给编程助手。</>,
    exportSheet: <>在无痕窗口打开每个产品，新建对话提问，并粘贴完整回答，即使没有提到你也要贴。</>,
    importSheet: <>导入的回答计为“网页”，不会和 API 回答混在一起。同一天再次导入同一张表会更新这些回答，而不是重复添加。</>,
    correct: <>当工具漏掉了提及（例如昵称）或误判时使用。把昵称作为别名加到{n("nav.brand")}里，下次就能识别。</>,
    addQuestion: <>像买家一样说出品类，见“写好问题”。分组决定这个问题如何计入统计。</>,
    saveQuestions: <>不想要的问题请停用而不是删除，以保留历史。点出你品牌名的问题会被标为品牌词，计入认知度。</>,
    redraft: <>在后台运行，需要几分钟，可以先离开页面。项目有其他任务在运行时无法开始。完成后请检查两份列表。</>,
    createProject: <>先添加模型密钥，否则问题只能用通用模板。没有网站时勾选“{n("projects.noSite")}”并填写品牌名。</>,
    saveBrand: <>空着的字段表示网站上没找到，把你知道的补上。品牌名同时也是项目名，只有管理员能修改。</>,
    addCompetitor: <>添加三到六个真实对手，写上大家常用的叫法。回答中发现的未确认竞品可以在这里确认。</>,
    saveCompetitors: <>不会重新统计过去的回答，从下次采样起生效。</>,
    accept: <>此刻会记录基线（例如“影响 3 个页面”），验收时与起点比较。</>,
    dismiss: <>已忽略的条目会移到“已忽略”标签页，忽略期间不会再被建议。</>,
    progress: <>“开始”把事项标为进行中；“标记完成”交给下一个周期验收。搜索类事项由你自己判断。</>,
    restore: <>恢复后会作为已接受的事项出现在“进行中”，可以直接开始。</>,
    buildReport: <>口径与看板完全一致。无论看板用什么语言，报告正文都是英文。</>,
    downloadReport: <>HTML 可以用任何浏览器打开，也能直接发邮件；Markdown 适合粘贴到文档和知识库。</>,
    searchSync: <>总数来自 Google 的每日汇总，明细来自查询报表；两者不一致是因为 Google 隐藏了匿名查询。</>,
    saveKeyword: <>收藏会为这个项目标记该查询词，方便在大量数据中再次找到。</>,
    googleConnect: <>需要先在“高级”里配置一次本服务器的 OAuth 客户端。Google 会显示授权页，然后把你带回这里。</>,
    googleSyncNow: <>每张卡片都会显示最后导入的日期。资源较大时，第一次导入可能要一段时间。</>,
    saveProperties: <>选择与本项目网站对应的 Search Console 资源和 GA4 媒体资源；每个项目各自设置。</>,
    googleDisconnect: <>会停止本服务器上所有项目今后的导入。用“{n("google.connect")}”可重新连接。</>,
    testConnection: <>失败时会显示服务商返回的信息：密钥错误、额度不足，或接口不认识该模型名。</>,
    saveConnect: <>密钥保存在本服务器上，只会发送给对应的服务商。</>,
    advanced: <>中转接口必须能识别你填写的模型名。留空则使用服务商默认值。</>,
    addUser: <>密码由你告诉对方，对方可以在账号菜单里修改。新用户在你分配权限之前看不到任何项目。</>,
    userAccess: <>管理员对所有项目始终拥有编辑权限，所以这个表格只对成员显示。</>,
    setPassword: <>至少 12 个字符。该用户需要在所有设备上重新登录。</>,
    disableUser: <>不能停用你自己，也不能停用最后一个启用的管理员。</>,
    deleteUser: <>无法撤销。不能删除你自己，也不能删除最后一个启用的管理员。</>,
    changePassword: <>需要先输入当前密码。当前浏览器会保持登录。</>,
  }),
};
