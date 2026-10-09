// SPDX-License-Identifier: AGPL-3.0-or-later

import { usageBodies, usageButtons } from "../usage";
import { Formula, FX, Table, Warn, type HelpDoc } from "../kit";

export const zh: HelpDoc = {
  groups: { start: "入门", read: "读懂结果", act: "采取行动", connect: "接入", help: "帮助" },
  body: (k) => { const { page, n } = k; return ({
    ...usageBodies(k),
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
  }); },
  buttons: (k) => { const { n, page } = k; return ({
    indexing: <>{n("tips.indexing")}</>,
    serve: <>在建好项目或一次改动较多之后使用。进度和日志在{page("settings/schedule", "nav.schedule")}的任务列表里；每个项目同一时间只运行一个任务。开启定期运行后会自动执行。</>,
    sample: <>只跑一轮。想让区间更窄，就提高“每个问题每个引擎每天采样次数”并让定期运行去完成。每次调用都消耗 token，成本公式见{n("nav.providers")}。</>,
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
    saveBrand: <>空着的字段表示网站上没找到，把你知道的补上。品牌名同时也是项目名，只有管理员能修改。</>,
    addCompetitor: <>添加三到六个真实对手，写上大家常用的叫法。回答中发现的未确认竞品可以在这里确认。</>,
    saveCompetitors: <>不会重新统计过去的回答，从下次采样起生效。</>,
    accept: <>此刻会记录基线（例如“影响 3 个页面”），验收时与起点比较。</>,
    dismiss: <>已忽略的条目会移到“已忽略”标签页，忽略期间不会再被建议。</>,
    restore: <>恢复后会作为已接受的事项出现在“进行中”，可以直接开始。</>,
    downloadReport: <>HTML 可以用任何浏览器打开，也能直接发邮件；Markdown 适合粘贴到文档和知识库。</>,
    saveKeyword: <>收藏会为这个项目标记该查询词，方便在大量数据中再次找到。</>,
    googleConnect: <>需要先在“高级”里配置一次本服务器的 OAuth 客户端。Google 会显示授权页，然后把你带回这里。</>,
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
    ...usageButtons(k),
  }); },
};
