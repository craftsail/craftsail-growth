// SPDX-License-Identifier: AGPL-3.0-or-later

import { Formula, FX, Table, Tip, Warn, type HelpDoc } from "../kit";

export const pt: HelpDoc = {
  groups: { start: "Primeiros passos", read: "Entender os resultados", act: "Agir", connect: "Conexões", help: "Ajuda" },
  topics: {
    start: { group: "start", label: "Comece aqui", keys: "visão geral ciclo o que é início rápido primeiro" },
    setup: { group: "start", label: "Configurar um projeto", keys: "projeto criar checklist chaves marca concorrentes perguntas agenda onboarding" },
    prompts: { group: "start", label: "Escrever boas perguntas", keys: "pergunta prompt lista grupo com marca sem marca categoria genérica zero" },
    terms: { group: "read", label: "Termos principais", keys: "glossário visibilidade reconhecimento participação de voz citação expansão período execução amostra acesso" },
    numbers: { group: "read", label: "Como os números funcionam", keys: "fórmula fórmulas intervalo ic wilson newcombe amostra pequena não medido api web falhas estabilidade" },
    pages: { group: "read", label: "Página por página", keys: "visão geral visibilidade participação citações expansão respostas busca relatórios navegação menu" },
    opportunities: { group: "act", label: "Plano de ação", keys: "oportunidades aceitar descartar prioridade corrigir primeiro verificada regrediu status" },
    audit: { group: "act", label: "Auditoria do site", keys: "prontidão camadas acesso descoberta compreensão citação bloqueado severidade evidência problemas exportar" },
    manual: { group: "act", label: "Amostragem manual", keys: "planilha importar exportar chatgpt web google ai overviews baidu sem api" },
    providers: { group: "connect", label: "Provedores de modelos", keys: "api chave mecanismo relay endpoint modelo busca na web custo" },
    google: { group: "connect", label: "Google Search e GA4", keys: "oauth cliente conta de serviço search console ga4 propriedade sincronizar redirecionamento" },
    access: { group: "connect", label: "Usuários e acesso", keys: "usuário usuários função administrador membro permissão ver editar senha compartilhar convidar equipe login" },
    schedule: { group: "connect", label: "Agenda e execuções", keys: "período todo dia execuções por dia tokens custo tentar de novo tarefa interrompida" },
    troubleshooting: { group: "help", label: "Solução de problemas", keys: "problema erro 0% não medido falhou travado reconectar vazio spa bloqueado" },
    limits: { group: "help", label: "O que isto não é", keys: "limites promessa garantia palavra-chave backlink" },
  },
  body: ({ page, topic, n }) => ({
    start: (
      <>
        <h2>Comece aqui</h2>
        <p>O craftsail-growth responde a uma pergunta: <strong>quando as pessoas perguntam aos mecanismos de IA sobre a sua categoria, as respostas mencionam e citam você, e isso está melhorando?</strong></p>
        <p>Ele funciona como um ciclo que se repete a cada período:</p>
        <ol>
          <li><strong>Medir.</strong> Faz suas perguntas a cada mecanismo várias vezes e registra cada resposta: quem é mencionado, em que ordem e quais fontes são citadas.</li>
          <li><strong>Diagnosticar.</strong> Rastreia e audita seu site em quatro camadas: os mecanismos conseguem buscá-lo, encontrá-lo, entendê-lo e citá-lo.</li>
          <li><strong>Agir.</strong> Achados da auditoria, lacunas de citação, dados de busca e quedas viram um único plano de ação ordenado. Você aceita os itens que vai fazer.</li>
          <li><strong>Verificar.</strong> O período seguinte confere cada ação aceita pela sua regra e a marca como verificada, ou como regrediu se ela voltar a falhar.</li>
        </ol>
        <p>A barra lateral segue o mesmo ciclo: <strong>{n("nav.sections.measure")}</strong> mede, <strong>{n("nav.sections.improve")}</strong> diagnostica e age, <strong>{n("nav.sections.project")}</strong> guarda a configuração deste projeto e <strong>{n("nav.sections.workspace")}</strong> o que todos os projetos compartilham.</p>
        <h3>Seus primeiros 15 minutos</h3>
        <ol>
          <li>Adicione pelo menos uma chave de mecanismo em {page("settings/providers", "nav.providers")}. Faça isso <strong>antes</strong> de criar um projeto; a chave também é usada para rascunhar os fatos da marca e as perguntas.</li>
          <li>Crie um projeto em {page("settings/projects", "nav.projects")} com “Executar agora o primeiro período” marcado.</li>
          <li>Enquanto ele roda, leia {topic("prompts", "Escrever boas perguntas")} e depois confira a lista rascunhada em {page("settings/questions", "nav.questions")}.</li>
          <li>Quando terminar, abra {page("overview", "nav.overview")} e depois o {page("opportunities", "nav.actionPlan")}.</li>
        </ol>
        <p>Depois, administradores podem convidar colegas ou clientes em {page("settings/users", "nav.users")}; veja {topic("access", "Usuários e acesso")}.</p>
        <Tip>O primeiro período só mostra onde você está. O valor vem de repeti-lo: ative uma agenda em {page("settings/schedule", "nav.schedule")}.</Tip>
      </>
    ),
    setup: (
      <>
        <h2>Configurar um projeto</h2>
        <p>Um projeto é uma marca e o seu site. Todo o resto, das perguntas aos relatórios, pertence a um projeto. Troque de projeto pelo nome do projeto no início da trilha de navegação, no topo de cada página.</p>
        <Table head={["Passo", "Onde", "Por que importa"]} rows={[
          ["1. Conectar mecanismos", page("settings/providers", "nav.providers"), "Sem chave nada é amostrado, e as perguntas voltam a modelos genéricos."],
          ["2. Criar o projeto", page("settings/projects", "nav.projects"), "Informe a URL do site. Sem site, marque “Sem site próprio” e informe o nome da marca."],
          ["3. Conferir os fatos da marca", page("settings/brand", "nav.brand"), "Um formulário curto: nome e outros nomes, definição em uma linha, categoria, público, números-chave com fontes, quando serve e limites. Rascunhado a partir do site; campos vazios não foram encontrados lá. O llms.txt e o JSON-LD gerados vêm daqui."],
          ["4. Conferir concorrentes", page("settings/competitors", "nav.competitors"), "De três a seis concorrentes reais, com apelidos. A participação de voz e a posição são contadas contra essa lista."],
          ["5. Ajustar as perguntas", page("settings/questions", "nav.questions"), <>O passo mais importante. Veja {topic("prompts", "Escrever boas perguntas")}.</>],
          ["6. Conectar o Google (opcional)", page("settings/google", "nav.google"), "Adiciona cliques, consultas e sessões de busca, e itens de busca no plano de ação."],
          ["7. Agendar", page("settings/schedule", "nav.schedule"), "Execute um período por semana ou por dia para que tendências e verificações funcionem."],
          ["8. Convidar pessoas (opcional)", page("settings/users", "nav.users"), "Adicione colegas ou clientes e compartilhe cada projeto para ver ou editar."],
        ]} />
        <Warn>Criou o projeto antes de adicionar uma chave? As perguntas dele são modelos genéricos. Adicione uma chave, abra {page("settings/questions", "nav.questions")} e clique em <strong>Refazer com IA</strong>. Isso substitui as perguntas e os concorrentes por um rascunho escrito a partir do seu site; revise os dois depois.</Warn>
      </>
    ),
    prompts: (
      <>
        <h2>Escrever boas perguntas</h2>
        <p>Perguntas, também chamadas de prompts, são o que a ferramenta pergunta a cada mecanismo. Elas decidem o que é medido; uma lista fraca torna todos os outros números sem sentido. Edite-as em {page("settings/questions", "nav.questions")}.</p>
        <h3>Nomeie a categoria como um comprador faria</h3>
        <Table head={["Fraca", "Melhor"]} rows={[
          ["Quais são as melhores ferramentas desta categoria?", "Quais são as melhores ferramentas de linha de comando para converter documentos Word em PDF?"],
          ["O que uma equipe pequena deve usar?", "Qual API de automação de documentos uma pequena equipe de SaaS deve usar?"],
          ["Por onde começo no primeiro dia?", "Como gerar relatórios Excel em Python sem o Microsoft Office?"],
        ]} />
        <p>Uma pergunta que não nomeia a categoria faz os mecanismos pedirem esclarecimento em vez de recomendar algo. A visibilidade fica então em 0%, faça o que fizer no site.</p>
        <h3>Grupos</h3>
        <p>Cada pergunta pertence a um grupo. Recomendação, Comparação, Alternativas e Preço são perguntas de <strong>comprador</strong>. Riscos e Caso de uso explicam a categoria. <strong>Verificação de marca</strong> cita sua marca de propósito.</p>
        <h3>Com marca e sem marca</h3>
        <p>Uma pergunta que contém o nome da marca, um apelido ou o seu domínio é <strong>com marca</strong>; a coluna Sistema mostra isso. Perguntas com marca mencionam você quase por definição, então são contadas como <strong>reconhecimento</strong> e nunca entram na visibilidade. Prefira perguntas de comprador sem marca.</p>
        <h3>Regras práticas</h3>
        <ul>
          <li>Escreva as perguntas no idioma dos seus compradores. Cada pergunta ativa é feita em cada mecanismo conectado.</li>
          <li>Comece com 10 a 20 perguntas. Cada uma custa uma chamada por mecanismo por execução.</li>
          <li>Use suas próprias tags (por exemplo, uma linha de produto) para filtrar as páginas de visibilidade em IA.</li>
          <li>Desative uma pergunta em vez de excluí-la para manter o histórico.</li>
          <li><strong>Refazer com IA</strong> pede a um modelo conectado uma nova lista escrita a partir do seu site. Ela substitui as perguntas e os concorrentes atuais.</li>
        </ul>
      </>
    ),
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
    pages: (
      <>
        <h2>Página por página</h2>
        <p>Páginas com várias visões, como {n("nav.audit")} e {n("nav.search")}, mostram-nas como abas abaixo do título.</p>
        <Table head={["Página", "Responde à pergunta", "O que fazer em seguida"]} rows={[
          [page("overview", "nav.overview"), "Onde estamos? Visibilidade, participação de voz, citações próprias, prontidão, principais ações.", "Abra a área mais fraca."],
          [page("ai/visibility", "nav.visibility"), "Quais perguntas nos mencionam, em qual mecanismo, ao longo do tempo.", "Abra uma pergunta para ler as respostas."],
          [page("ai/share-of-voice", "nav.sov"), "Quem é citado no nosso lugar.", <>Adicione concorrentes que faltam em {n("nav.competitors")}.</>],
          [page("ai/citations", "nav.citations"), "Em quais fontes os mecanismos se apoiam, por tipo de fonte e de página; quais citam concorrentes e não nós.", <>Trabalhe essas fontes a partir do {n("nav.actionPlan")}.</>],
          [page("ai/fan-out", "nav.fanout"), "O que os mecanismos realmente buscam. Só mecanismos com busca na web informam isso.", "Use as palavras adicionadas nas suas páginas e perguntas."],
          [page("ai/answers", "nav.answers"), "Cada resposta bruta, com citações e buscas.", "Corrija leituras erradas; importe amostras manuais."],
          [page("opportunities", "nav.actionPlan"), "O que fazer em seguida, em ordem, e se funcionou.", "Aceitar, começar, marcar como concluída."],
          [page("audit", "nav.audit"), "Os mecanismos conseguem buscar, encontrar, entender e citar o site.", "Corrija a primeira camada que falha."],
          [page("search", "nav.search"), "Cliques, impressões, sessões, consultas e páginas de entrada do Google.", "Compare com a visibilidade em IA."],
          [page("reports", "nav.reports"), "Uma página compartilhável por período.", "Baixe em HTML ou Markdown."],
        ]} />
        <p>Quando o título da página mostra <strong>{n("access.viewOnly")}</strong>, você só pode ver esse projeto: todos os botões que alteram dados ficam ocultos. Veja {topic("access", "Usuários e acesso")}.</p>
        <Tip>Os filtros (modelos, tags, período) ficam no topo de cada página de visibilidade em IA. Eles ficam no endereço da página, então você pode salvar ou compartilhar uma visão filtrada.</Tip>
      </>
    ),
    opportunities: (
      <>
        <h2>Plano de ação</h2>
        <p>O {page("opportunities", "nav.actionPlan")} é a única lista de tarefas. As sugestões vêm de quatro fontes e são agrupadas em <strong>Corrigir primeiro</strong>, <strong>Vale a pena</strong> e <strong>Se sobrar tempo</strong>:</p>
        <Table head={["Fonte", "Exemplo", "Grupo"]} rows={[
          ["Auditoria do site", "Um WAF bloqueia rastreadores de IA; sem dados estruturados", "Crítico → Corrigir primeiro, aviso → Vale a pena, informação → Se sobrar tempo. Regras observacionais nunca vão para Corrigir primeiro."],
          ["Citações em IA", "Concorrentes são citados numa pergunta e você não", "Vale a pena, ou Se sobrar tempo quando a fonte é difícil de alcançar"],
          ["Busca", "Uma consulta fica logo fora das três primeiras posições; taxa de cliques baixa", "Se sobrar tempo"],
          ["Mudança de métrica", "A visibilidade caiu além do ruído", "Corrigir primeiro"],
        ]} />
        <h3>Abas e ciclo de vida</h3>
        <ol>
          <li><strong>Sugestões</strong>: aceite um item ou descarte-o se não se aplicar. Itens descartados podem ser restaurados na aba deles.</li>
          <li><strong>Em andamento</strong>: itens aceitos são registrados com uma linha de base, por exemplo “3 páginas afetadas”. Clique em Começar e depois em Marcar como concluída.</li>
          <li><strong>Verificadas</strong> quando a verificação do período seguinte passa. Uma ação verificada que volta a falhar fica como <strong>Regrediu</strong> e retorna para Em andamento.</li>
        </ol>
        <h3>Como “concluída” é verificada</h3>
        <ul>
          <li>Ações de auditoria passam quando o problema some da auditoria mais recente.</li>
          <li>Ações de citação passam quando a visibilidade naquela pergunta sobe além do ruído.</li>
          <li>Uma queda de visibilidade passa quando a visibilidade volta ao ruído da linha de base.</li>
          <li>Ações de busca são verificadas por você: marque como concluída quando terminar.</li>
        </ul>
        <h3>Quando cada sugestão aparece</h3>
        <p><strong>Citações em IA.</strong> Uma pergunta recebe uma sugestão de citação quando os mecanismos citam sites de concorrentes para ela e nunca o seu:</p>
        <Formula>{FX.gap}</Formula>
        <p><strong>Busca.</strong> Do Google Search Console, em 28 dias que terminam três dias atrás (os dados do Google chegam com atraso), comparados com os 28 dias anteriores:</p>
        <Formula>{FX.striking}</Formula>
        <p><strong>Mudança de métrica.</strong> A visibilidade dos últimos 30 dias comparada com os 30 dias anteriores, com a regra “Alta, queda ou ruído” de {topic("numbers", "Como os números funcionam")}.</p>
        <h3>Como uma ação é verificada</h3>
        <Formula>{FX.verify}</Formula>
        <p>Consultas de busca com a sua marca ficam de fora: quem busca pelo seu nome já encontrou você.</p>
        <Tip>Cada item mostra “Como corrigir” e “Concluída quando”. A CLI tem a mesma lista: <code>craftsail-growth opportunities --slug &lt;project&gt;</code>.</Tip>
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
    google: (
      <>
        <h2>Google Search e GA4</h2>
        <p>Opcional. Adiciona cliques, impressões e consultas do Search Console e sessões do GA4 a {n("nav.search")}, e itens de busca ao plano de ação. A visibilidade em IA e a auditoria funcionam sem isso.</p>
        <h3>Passo 1: conectar uma conta Google</h3>
        <p>Este servidor precisa do seu próprio <strong>cliente OAuth</strong> uma vez. Ele é a identidade do app no Google: ao clicar em Conectar Google, o Google mostra a tela de consentimento deste app e envia o resultado de volta para a URI de redirecionamento. Uma instalação própria não pode compartilhar um cliente, porque cada servidor tem seu próprio endereço.</p>
        <ol>
          <li>Ative a Search Console API, a Google Analytics Data API e a Analytics Admin API no Google Cloud.</li>
          <li>Crie um cliente OAuth do tipo Aplicativo da Web e adicione exatamente a URI de redirecionamento mostrada na página.</li>
          <li>Cole o ID e a chave secreta do cliente, salve e clique em Conectar Google.</li>
        </ol>
        <p>Prefere não entrar pelo navegador? Use uma <strong>conta de serviço</strong> em Avançado e adicione o e-mail dela à propriedade do Search Console e como Leitor na propriedade do GA4.</p>
        <h3>Passo 2: escolher as propriedades</h3>
        <p>Escolha a propriedade do Search Console e a do GA4 deste projeto e clique em <strong>Sincronizar agora</strong>. Cada cartão mostra o status e o último dia importado.</p>
        <h3>Como os números de busca são calculados</h3>
        <p>Os totais vêm dos totais diários do Google. As linhas por consulta e por página deixam de fora consultas anonimizadas, então somam sempre menos; a diferença é mostrada, nunca preenchida.</p>
        <Formula>{FX.search}</Formula>
        <Warn>Enquanto a tela de consentimento estiver em Teste, o Google expira o login após cerca de 7 dias e o status muda para “Precisa entrar de novo”. Publique o app ou reconecte quando isso acontecer. Os dados importados são mantidos.</Warn>
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
    troubleshooting: (
      <>
        <h2>Solução de problemas</h2>
        <Table head={["Você vê", "Causa provável", "O que fazer"]} rows={[
          ["Visibilidade 0% em todas as perguntas", "As perguntas não nomeiam a categoria, então os mecanismos perguntam o que você quer dizer. Comum quando o projeto foi criado sem chave de mecanismo.", <>Reescreva-as ({topic("prompts", "guia")}) ou conecte um modelo e clique em Refazer com IA em {n("nav.questions")}.</>],
          ["“Não medido” ou —", "Ainda não há o que contar, por exemplo nenhum concorrente citado.", "Adicione concorrentes; faça mais amostras."],
          ["“amostra pequena”", "Menos de 30 respostas no período.", "Aumente as execuções por dia ou amplie o período."],
          ["Muitas execuções com falha", "Chave errada, cota, ou um relay que não conhece o modelo.", "Teste o provedor; confira o nome do modelo em Avançado; Tentar de novo as falhas."],
          ["Nenhuma resposta", "Nenhum mecanismo conectado, ou a amostragem nunca rodou.", "Conecte um provedor e clique em Amostrar agora."],
          ["Camada de acesso falhando: AI_UA_BLOCKED", "Uma CDN ou WAF rejeita user agents de rastreadores de IA enquanto navegadores funcionam.", "Libere GPTBot, ClaudeBot, PerplexityBot e similares nas configurações de bots da CDN."],
          ["Páginas quase sem texto", "O site é renderizado em JavaScript; os rastreadores veem uma casca vazia.", "Renderize no servidor ou pré-renderize as páginas principais."],
          ["Google “Precisa entrar de novo”", "O token de atualização expirou (modo Teste) ou foi revogado.", <>Reconecte em {n("nav.google")}.</>],
          ["Totais de busca ≠ soma das consultas", "O Google oculta consultas anonimizadas nos relatórios por linha.", "Esperado. Os totais vêm dos totais diários do Google."],
          ["Uma tarefa continua “em execução”", "Ainda está trabalhando; períodos com muitas perguntas demoram.", <>Acompanhe em {n("nav.schedule")}; reinicie o servidor só se o log parar.</>],
          ["Um projeto não aparece na minha lista", "Ele não foi compartilhado com você.", <>Peça a um administrador para dar Ver ou Editar em {n("nav.users")}.</>],
          [<>Sem botões de salvar ou executar; o título mostra {n("access.viewOnly")}</>, "Você só pode ver este projeto.", "Peça a um administrador o acesso Editar."],
          ["Saí da conta depois que um administrador mudou algo", "Sua senha foi redefinida ou sua conta foi desativada.", "Entre com a nova senha ou fale com o administrador."],
          ["Mensagens ou relatório em inglês", "Mensagens do servidor, logs de tarefas e o conteúdo dos relatórios estão em inglês em todos os idiomas.", "Esperado por enquanto."],
        ]} />
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
  }),
  buttons: ({ n, page }) => ({
    serve: <>Use depois de configurar um projeto ou mudar muita coisa de uma vez. O progresso e o log ficam em Tarefas, em {page("settings/schedule", "nav.schedule")}; roda uma tarefa por projeto de cada vez. Com a agenda ligada, roda sozinho.</>,
    sample: <>Só uma rodada. Para intervalos mais estreitos, aumente “Execuções por pergunta e mecanismo por dia” e deixe a agenda rodar. Cada chamada gasta tokens; veja a fórmula de custo em {n("nav.providers")}.</>,
    syncGoogle: <>Precisa do Google conectado e das propriedades escolhidas em {page("settings/google", "nav.google")}. Os dados do Google chegam com uns três dias de atraso.</>,
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
    redraft: <>Roda em segundo plano e pode levar alguns minutos; você pode sair da página. Não começa se outra tarefa estiver rodando no projeto. Revise as duas listas depois.</>,
    createProject: <>Adicione uma chave de modelo antes, senão as perguntas voltam a modelos genéricos. Sem site, marque “{n("projects.noSite")}” e informe o nome da marca.</>,
    saveBrand: <>Campos vazios não foram encontrados no site; preencha o que você sabe. Só administradores mudam o nome da marca, porque ele também é o nome do projeto.</>,
    addCompetitor: <>Adicione de três a seis concorrentes reais, com os nomes que as pessoas usam. Concorrentes não confirmados encontrados nas respostas podem ser confirmados aqui.</>,
    saveCompetitors: <>Respostas antigas não são recontadas; vale a partir da próxima amostragem.</>,
    accept: <>A linha de base (por exemplo “3 páginas afetadas”) é registrada neste momento, então a verificação compara com o ponto de partida.</>,
    dismiss: <>Itens descartados vão para a aba Descartadas e não são sugeridos de novo enquanto estiverem descartados.</>,
    progress: <>Começar marca a ação como Em andamento; Marcar como concluída a entrega à verificação do próximo período. Ações de busca são verificadas por você.</>,
    restore: <>Ela volta como ação aceita em Em andamento, pronta para Começar.</>,
    buildReport: <>Usa os mesmos números e regras do painel. O texto do relatório é em inglês, qualquer que seja o idioma do painel.</>,
    downloadReport: <>O HTML abre em qualquer navegador e pode ser enviado por e-mail como está; o Markdown serve para colar em documentos e wikis.</>,
    searchSync: <>Os totais vêm dos totais diários do Google e as linhas dos relatórios de consulta; eles diferem porque o Google oculta consultas anonimizadas.</>,
    saveKeyword: <>Salvar marca a consulta neste projeto para você achá-la de novo entre muitas linhas.</>,
    googleConnect: <>Precisa do cliente OAuth deste servidor uma vez (em Avançado). O Google mostra a tela de consentimento e traz você de volta.</>,
    googleSyncNow: <>Cada cartão mostra o último dia importado. A primeira importação pode demorar em propriedades grandes.</>,
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
  }),
};
