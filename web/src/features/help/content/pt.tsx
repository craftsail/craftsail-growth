// SPDX-License-Identifier: AGPL-3.0-or-later

import { usageBodies, usageButtons } from "../usage";
import { Formula, FX, Table, Warn, type HelpDoc } from "../kit";

export const pt: HelpDoc = {
  groups: { start: "Primeiros passos", read: "Entender os resultados", act: "Agir", connect: "Conexões", help: "Ajuda" },
  body: (k) => { const { page, n } = k; return ({
    ...usageBodies(k),
    terms: (
      <>
        <h2>Termos principais</h2>
        <Table head={["Termo", "Significado"]} rows={[
          ["Mecanismo / provedor de modelo", "Um produto de IA que a ferramenta consulta, como DeepSeek, OpenAI ou Perplexity."],
          ["Pergunta (prompt)", "Um item da sua lista, feito em todos os mecanismos."],
          ["Execução / resposta", "Um mecanismo respondendo uma pergunta uma vez. As respostas variam, então cada pergunta é feita várias vezes por dia."],
          ["Período", "Uma passada completa: rastrear, auditar, amostrar, sincronizar o Google, verificar, atualizar o plano de ação, gerar o relatório."],
          ["Acesso (API / Web)", "Respostas da API de um mecanismo e respostas coletadas à mão no produto web ou app ficam separadas; elas citam fontes diferentes."],
          ["Visibilidade", "Parcela das respostas bem-sucedidas a perguntas sem marca que mencionam sua marca."],
          ["Reconhecimento", "A mesma taxa em perguntas com marca. Mostra se o mecanismo conhece você, não se o recomenda."],
          ["Participação de voz", "Suas menções divididas pelas suas menções mais as dos concorrentes."],
          ["Citação", "Uma URL de fonte que o mecanismo anexou à resposta."],
          ["Expansão de buscas (query fan-out)", "As buscas na web que um mecanismo fez para responder, comparadas com as palavras da própria pergunta."],
          ["Sugestão", "Um item do plano de ação vindo da auditoria, das citações, dos dados de busca ou de uma queda de métrica."],
          ["Ação", "Uma sugestão que você aceitou. Tem uma linha de base e uma regra que verifica se funcionou."],
          ["Camada de prontidão", "Acesso, Descoberta, Compreensão, Citação: as quatro coisas de que uma página precisa, na ordem de correção."],
        ]} />
      </>
    ),
    numbers: (
      <>
        <h2>Como os números funcionam</h2>
        <p><strong>Visibilidade</strong> é a parcela das respostas bem-sucedidas a perguntas sem marca que mencionam a marca. Perguntas que citam a marca são contadas à parte como <strong>reconhecimento</strong>.</p>
        <p>As respostas mudam a cada execução, então toda taxa é uma estimativa. Ao lado dela você vê um <strong>intervalo de 95%</strong> (por exemplo “IC 0–22%”) e o número de respostas (n). Leia o intervalo, não só o número: 0% em 14 respostas significa “provavelmente abaixo de 22%”, não “nunca”.</p>
        <ul>
          <li><strong>Amostra pequena</strong>: menos de 30 respostas. Trate a taxa como um sinal aproximado.</li>
          <li><strong>Não medido</strong> ou <strong>—</strong>: ainda não há o que contar, por exemplo nenhum concorrente foi citado. Nunca aparece como 0%.</li>
          <li><strong>Execuções com falha</strong> (tempo esgotado, cota) ficam fora das taxas e são contadas à parte. Tente de novo em {page("settings/schedule", "nav.schedule")}.</li>
          <li>Respostas de <strong>API e Web</strong> nunca são somadas. Quando existem as duas, aparece uma chave na barra de filtros.</li>
          <li><strong>Alta ou queda</strong> entre períodos só é indicada quando a mudança é maior que o ruído (um intervalo de Newcombe de 95% que não inclui zero).</li>
        </ul>
        <h3>Fórmulas</h3>
        <p>As fórmulas usam notação em inglês; o significado de cada uma está explicado abaixo.</p>
        <p><strong>Visibilidade.</strong> Das respostas que voltaram, quantas citam você. Perguntas que já contêm o seu nome ficam de fora, porque citam você quase por definição.</p>
        <Formula>{FX.visibility}</Formula>
        <p><strong>Reconhecimento.</strong> A mesma conta, só nas perguntas que citam o seu nome. Mostra se o mecanismo conhece você, não se o recomenda.</p>
        <Formula>{FX.recognition}</Formula>
        <p><strong>Top 1 e top 3.</strong> Ser citado primeiro vale mais do que ser citado por último, então a ordem da primeira aparição também conta.</p>
        <Formula>{FX.top}</Formula>
        <p><strong>Participação de voz.</strong> A sua parte de todas as menções de marca, as suas mais as dos concorrentes. Sem nenhuma menção, não é medida.</p>
        <Formula>{FX.sov}</Formula>
        <p><strong>Respostas que citam seu domínio</strong> e <strong>participação de citações.</strong> Contagens de citações nunca são comparadas entre mecanismos: alguns anexam muito mais fontes por resposta do que outros.</p>
        <Formula>{FX.ownCite}</Formula>
        <h3>O intervalo ao lado de cada taxa</h3>
        <p>Um mecanismo responde de um jeito diferente a cada vez, então uma taxa é uma estimativa. O intervalo de Wilson de 95% mostra a faixa em que a taxa real provavelmente está; ele continua razoável com poucas respostas, quando um ± simples passaria de 0% ou 100%.</p>
        <Formula>{FX.wilson}</Formula>
        <p>Exemplo:</p>
        <Formula>{FX.wilsonExample}</Formula>
        <h3>Alta, queda ou ruído</h3>
        <p>Dois períodos são comparados com um intervalo sobre a diferença, não a olho. Só uma diferença cujo intervalo inteiro fica de um lado do zero é chamada de mudança.</p>
        <Formula>{FX.change}</Formula>
        <Formula>{FX.changeExample}</Formula>
        <h3>Estabilidade das citações</h3>
        <p>O quanto o conjunto de sites citados muda de um dia para o outro. Nota baixa significa que as fontes ainda estão abertas a novos sites; nota alta significa que os mecanismos citam sempre os mesmos. Serve só para ordenar sugestões.</p>
        <Formula>{FX.stability}</Formula>
      </>
    ),
    audit: (
      <>
        <h2>Auditoria do site</h2>
        <p>A auditoria aplica 47 regras às páginas rastreadas e as agrupa em quatro camadas, na ordem de correção:</p>
        <Table head={["Camada", "Pergunta", "Achados típicos"]} rows={[
          ["Acesso", "Os rastreadores conseguem buscar o conteúdo?", "robots.txt bloqueia bots de IA, uma CDN rejeita user agents de IA, noindex, páginas vazias sem JavaScript"],
          ["Descoberta", "Conseguem encontrar todas as URLs?", "Sem sitemap, sitemap fora do robots.txt, sem llms.txt, links quebrados"],
          ["Compreensão", "Conseguem saber o que é cada página?", "Sem dados estruturados, schema que contradiz a página, títulos longos"],
          ["Citação", "Há algo que valha citar?", "Sem frase de definição, poucos números, sem comparação, sem trecho citável"],
        ]} />
        <p>Quando uma camada falha, os achados das camadas seguintes ficam <strong>bloqueados</strong>: corrigi-los agora não mudaria o que os mecanismos veem.</p>
        <p>Cada regra mostra sua <strong>evidência</strong>: padrão, documentação oficial, experimento, observacional ou regra prática. Regras observacionais e regras práticas são conselhos e nunca são críticas.</p>
        <h3>Como o status de uma camada é decidido</h3>
        <Formula>{FX.layers}</Formula>
        <p>O número de prontidão em {n("nav.overview")} (por exemplo “1 / 4”) conta as camadas aprovadas sem nenhum achado; uma camada só com avisos não conta.</p>
        <h3>Como uma página recebe nota</h3>
        <p>A aba {n("nav.tabs.pages")} dá a cada página uma nota de 0 a 100 a partir de seis partes. Os limites descrevem páginas que os mecanismos citaram em conjuntos de dados publicados; são sinais, não garantia. As palavras são contadas no HTML recebido, sem executar JavaScript.</p>
        <Formula>{FX.pageScore}</Formula>
        <Formula>{FX.pageParts}</Formula>
        <ul>
          <li>A aba <strong>{n("nav.tabs.readiness")}</strong> mostra as camadas; <strong>{n("nav.tabs.issues")}</strong> lista cada regra com as páginas afetadas; <strong>{n("nav.tabs.pages")}</strong> dá uma nota a cada página.</li>
          <li><strong>Exportar para IA</strong> baixa a auditoria em Markdown para entregar a um assistente de programação.</li>
          <li>Rastreie de novo depois de mudar o site; a auditoria só vê o último rastreamento.</li>
        </ul>
      </>
    ),
    manual: (
      <>
        <h2>Amostragem manual</h2>
        <p>Alguns produtos que as pessoas realmente usam não têm API: ChatGPT web com busca, Claude web, Google AI Overviews, Baidu AI Search, o app Doubao, Metaso, Nano AI. Ainda assim você pode medi-los à mão.</p>
        <ol>
          <li>Em {page("ai/answers", "nav.answers")}, clique em <strong>Exportar planilha de amostragem</strong>.</li>
          <li>Para cada pergunta, abra o produto numa janela anônima, inicie uma conversa nova, faça a pergunta e cole a resposta completa na planilha, mesmo quando você não for mencionado.</li>
          <li>Cole a planilha preenchida de volta e clique em <strong>Importar planilha preenchida</strong>.</li>
        </ol>
        <Warn>Não amostre com a sua conta do dia a dia: os mecanismos personalizam as respostas. Use uma janela anônima ou um perfil usado só para amostragem. Respostas manuais contam como Web e nunca se misturam com respostas de API.</Warn>
        <p>A CLI faz o mesmo: <code>craftsail-growth sample-sheet --slug &lt;project&gt; --intent buyer --limit 20</code> e <code>craftsail-growth sample-import</code>.</p>
      </>
    ),
    providers: (
      <>
        <h2>Provedores de modelos</h2>
        <p>{page("settings/providers", "nav.providers")} lista cada mecanismo com seu status. Escolha um, cole a chave, clique em <strong>Testar conexão</strong> e salve. Mecanismos sem chave são ignorados.</p>
        <Table head={["Tipo", "Mecanismos", "O que mede"]} rows={[
          ["Busca na web", "Perplexity; Doubao com o plugin de conteúdo Ark ativado", "O que o mecanismo encontra e cita hoje. O mais próximo do produto que o consumidor usa."],
          ["Responde de memória", "DeepSeek, Kimi, GLM, MiniMax, OpenAI, Claude, Gemini, Grok (conforme configurado)", "O que o modelo já sabe sobre você. Muda devagar, com novas versões do modelo."],
        ]} />
        <ul>
          <li><strong>Avançado</strong> permite trocar o modelo ou apontar para um endpoint de relay. Um relay precisa de um nome de modelo que ele reconheça.</li>
          <li>As chaves ficam em <code>config/default.toml</code> neste servidor e só saem dele para chamar o próprio provedor.</li>
          <li>{n("nav.schedule")} mostra os tokens estimados e informados de cada execução.</li>
        </ul>
        <h3>Quanto custa a amostragem</h3>
        <p>Cada pergunta ativa é feita em cada mecanismo com chave, tantas vezes por dia quanto você definir. O número de tokens é uma estimativa para planejamento; quando o mecanismo informa o uso real, ele aparece ao lado.</p>
        <Formula>{FX.calls}</Formula>
        <Formula>{FX.callsExample}</Formula>
      </>
    ),
    schedule: (
      <>
        <h2>Agenda e execuções</h2>
        <p>Um <strong>período</strong> executa sete passos: rastrear, auditar, amostrar, sincronizar o Google, verificar ações, atualizar o plano de ação e gerar o relatório. Se ainda não houver perguntas, ele primeiro rascunha os fatos da marca e as perguntas.</p>
        <ul>
          <li><strong>Executar um período a cada</strong> dia, semana, duas semanas ou 30 dias. O servidor verifica a cada 30 minutos; ele precisa estar rodando para a agenda disparar.</li>
          <li><strong>Execuções por pergunta e mecanismo por dia</strong> (1–10, padrão 3). Mais execuções dão intervalos mais estreitos e custam mais.</li>
          <li>Os botões executam um passo isolado: Amostrar agora, Sincronizar Google, Rastrear site, Auditar site, Verificar ações.</li>
          <li><strong>Execuções de amostragem</strong> lista cada lote com chamadas planejadas, bem-sucedidas e com falha, e os tokens. <strong>Tentar de novo as falhas</strong> repete só as chamadas que falharam.</li>
          <li>Roda uma tarefa por projeto de cada vez. Uma tarefa mostrada como <em>Interrompida</em> foi parada por um reinício do servidor; inicie-a de novo.</li>
        </ul>
      </>
    ),
    access: (
      <>
        <h2>Usuários e acesso</h2>
        <p>Administradores adicionam pessoas em {page("settings/users", "nav.users")} e escolhem, para cada projeto, se a pessoa pode vê-lo, editá-lo ou nem enxergá-lo. Há duas funções:</p>
        <ul>
          <li><strong>Administrador</strong>: todos os projetos, além do espaço de trabalho: projetos, provedores de modelos, Google e usuários.</li>
          <li><strong>Membro</strong>: só os projetos compartilhados com ele, cada um com <strong>{n("users.access.view")}</strong> ou <strong>{n("users.access.edit")}</strong>.</li>
        </ul>
        <Table head={["", n("users.access.view"), n("users.access.edit"), n("users.roles.admin")]} rows={[
          ["Ler todas as páginas; baixar relatórios e exportações", "✓", "✓", "✓"],
          ["Alterar marca, concorrentes, perguntas e agenda", "", "✓", "✓"],
          ["Executar rastreamentos, auditorias e amostragem; aceitar e concluir ações; corrigir respostas", "", "✓", "✓"],
          ["Criar projetos; provedores de modelos; Google; usuários", "", "", "✓"],
        ]} />
        <h3>Convidar alguém</h3>
        <ol>
          <li>Abra {page("settings/users", "nav.users")} e preencha <strong>{n("users.add")}</strong>: um nome de usuário (pode ser um e-mail) e uma senha de pelo menos 12 caracteres.</li>
          <li>Selecione o novo usuário e defina cada projeto como {n("users.access.none")}, {n("users.access.view")} ou {n("users.access.edit")}. As mudanças valem na hora.</li>
          <li>Envie para a pessoa o endereço deste painel, o nome de usuário e a senha. Ela pode alterar a senha no menu da conta.</li>
        </ol>
        <h3>Bom saber</h3>
        <ul>
          <li>Um projeto não compartilhado com alguém não aparece para essa pessoa, nem mesmo o nome.</li>
          <li>As páginas de um projeto que você só pode ver mostram <strong>{n("access.viewOnly")}</strong> ao lado do título e escondem todos os botões que alteram dados.</li>
          <li>Altere a sua senha em <strong>{n("access.changePassword")}</strong>, no menu da conta no fim da barra lateral. Isso encerra a sessão nos seus outros dispositivos.</li>
          <li>Redefinir a senha de alguém, desativar ou excluir a pessoa encerra todas as sessões dela na hora. Excluir um usuário mantém todos os dados dos projetos.</li>
          <li>Sempre existe pelo menos um administrador ativo; o último não pode ser rebaixado, desativado nem excluído.</li>
          <li>Na atualização a partir do login único: essa conta vira o primeiro administrador, e todos entram de novo uma vez.</li>
          <li>Sem acesso? No servidor, rode <code>craftsail-growth user passwd &lt;nome&gt;</code> e digite a nova senha (ela aparece enquanto você digita; para escondê-la, use um pipe: <code>printf '%s\n' "$PASS" | craftsail-growth user passwd &lt;nome&gt;</code>). <code>craftsail-growth user add &lt;nome&gt; --admin</code> adiciona outro administrador.</li>
          <li>Se um administrador mudar seu acesso enquanto você está conectado, recarregue a página para ver a nova lista de projetos.</li>
          <li>O token de API da configuração do servidor funciona como administrador para scripts e a CLI.</li>
        </ul>
      </>
    ),
    limits: (
      <>
        <h2>O que isto não é</h2>
        <ul>
          <li><strong>Não é promessa de citações.</strong> Ninguém garante que um mecanismo vai citar uma página. A ferramenta mede, sugere e verifica.</li>
          <li><strong>Não é um ranking.</strong> As taxas são estimativas de amostras repetidas, mostradas com sua incerteza.</li>
          <li><strong>Não é pesquisa de palavras-chave nem backlinks.</strong> Não há fonte paga de dados de SEO.</li>
          <li><strong>Não publica nada.</strong> Rascunha trechos de correção como llms.txt e JSON-LD para você revisar; não altera o seu site.</li>
          <li><strong>Instalação própria, com funções simples.</strong> Administradores e membros com Ver ou Editar por projeto; sem equipes nem login único (SSO). Suas chaves e dados ficam neste servidor.</li>
        </ul>
      </>
    ),
  }); },
  buttons: (k) => { const { n, page } = k; return ({
    indexing: <>{n("tips.indexing")}</>,
    serve: <>Use depois de configurar um projeto ou mudar muita coisa de uma vez. O progresso e o log ficam em Tarefas, em {page("settings/schedule", "nav.schedule")}; roda uma tarefa por projeto de cada vez. Com a agenda ligada, roda sozinho.</>,
    sample: <>Só uma rodada. Para intervalos mais estreitos, aumente “Execuções por pergunta e mecanismo por dia” e deixe a agenda rodar. Cada chamada gasta tokens; veja a fórmula de custo em {n("nav.providers")}.</>,
    crawl: <>Só busca as páginas; não muda a auditoria. Rode a auditoria depois ou use “{n("audit.actions.crawl")}” em {page("audit", "nav.audit")}.</>,
    audit: <>É rápido porque nada é buscado. Se você mudou o site, rastreie antes, ou a auditoria ainda vê as páginas antigas.</>,
    verify: <>Uma ação passa quando sua regra é atendida e vai para Verificadas; uma ação verificada que falha depois fica como Regrediu. Veja as regras em {n("nav.actionPlan")}.</>,
    scheduleSave: <>O servidor confere a cada 30 minutos se há um período a rodar, então precisa estar ligado. Desligado para os períodos automáticos; os botões continuam funcionando.</>,
    retry: <>Aparece só em execuções com falhas. Resolva a causa antes (chave, cota, nome do modelo), senão as mesmas chamadas falham de novo.</>,
    stop: <>Respostas e páginas salvas antes da parada ficam. Uma tarefa marcada como interrompida foi parada por um reinício do servidor e pode ser iniciada de novo.</>,
    auditCrawl: <>Demora mais que “{n("audit.actions.again")}” porque cada página é buscada de novo.</>,
    auditAgain: <>Útil depois de uma mudança nas regras ou para atualizar resultados; não vê mudanças no site até o próximo rastreamento.</>,
    exportAI: <>O arquivo lista cada regra que falha, com a evidência, as páginas afetadas e como corrigir. Cole-o num assistente de programação junto com o código do site.</>,
    exportSheet: <>Abra cada produto numa janela anônima, faça a pergunta numa conversa nova e cole a resposta completa, mesmo quando você não for citado.</>,
    importSheet: <>Respostas importadas contam como Web e nunca se misturam com as de API. Importar a mesma planilha de novo no mesmo dia atualiza essas respostas em vez de duplicá-las.</>,
    correct: <>Use quando a ferramenta deixou passar uma menção (por exemplo, um apelido) ou marcou uma errada. Adicione o apelido como nome alternativo em {n("nav.brand")} para ser reconhecido na próxima vez.</>,
    addQuestion: <>Nomeie a categoria como um comprador faria; veja “Escrever boas perguntas”. O grupo decide como a pergunta é contada.</>,
    saveQuestions: <>Desative uma pergunta em vez de excluí-la para manter o histórico. Perguntas que citam a sua marca ficam marcadas como com marca e contam como reconhecimento.</>,
    saveBrand: <>Campos vazios não foram encontrados no site; preencha o que você sabe. Só administradores mudam o nome da marca, porque ele também é o nome do projeto.</>,
    addCompetitor: <>Adicione de três a seis concorrentes reais, com os nomes que as pessoas usam. Concorrentes não confirmados encontrados nas respostas podem ser confirmados aqui.</>,
    saveCompetitors: <>Respostas antigas não são recontadas; vale a partir da próxima amostragem.</>,
    accept: <>A linha de base (por exemplo “3 páginas afetadas”) é registrada neste momento, então a verificação compara com o ponto de partida.</>,
    dismiss: <>Itens descartados vão para a aba Descartadas e não são sugeridos de novo enquanto estiverem descartados.</>,
    restore: <>Ela volta como ação aceita em Em andamento, pronta para Começar.</>,
    downloadReport: <>O HTML abre em qualquer navegador e pode ser enviado por e-mail como está; o Markdown serve para colar em documentos e wikis.</>,
    saveKeyword: <>Salvar marca a consulta neste projeto para você achá-la de novo entre muitas linhas.</>,
    googleConnect: <>Precisa do cliente OAuth deste servidor uma vez (em Avançado). O Google mostra a tela de consentimento e traz você de volta.</>,
    saveProperties: <>Escolha a propriedade do Search Console e a do GA4 que correspondem ao site deste projeto; cada projeto tem as suas.</>,
    googleDisconnect: <>Interrompe as próximas importações de todos os projetos deste servidor. Reconecte com “{n("google.connect")}”.</>,
    testConnection: <>Uma falha mostra a mensagem do provedor: chave errada, sem cota ou um nome de modelo que o endpoint não conhece.</>,
    saveConnect: <>As chaves ficam neste servidor e só são enviadas a esse provedor.</>,
    advanced: <>Um endpoint de relay precisa aceitar o nome de modelo informado. Deixe vazio para usar o padrão do provedor.</>,
    addUser: <>Passe a senha você mesmo para a pessoa; ela pode alterá-la no menu da conta. Usuários novos não veem nenhum projeto até você dar acesso.</>,
    userAccess: <>Administradores sempre têm Editar em todos os projetos, então esta tabela só aparece para membros.</>,
    setPassword: <>Pelo menos 12 caracteres. O usuário precisa entrar de novo em todos os dispositivos.</>,
    disableUser: <>Você não pode desativar a si mesmo nem o último administrador ativo.</>,
    deleteUser: <>Não dá para desfazer. Você não pode excluir a si mesmo nem o último administrador ativo.</>,
    changePassword: <>Digite a senha atual primeiro. Este navegador continua conectado.</>,
    ...usageButtons(k),
  }); },
};
