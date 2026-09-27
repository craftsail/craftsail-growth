// SPDX-License-Identifier: AGPL-3.0-or-later

// Audit rule texts. Codes and English text live in internal/service/audit/issues.go.
export const pt = {
  "AI_UA_BLOCKED": {
    "title": "WAF ou CDN rejeita os user agents de rastreadores de IA",
    "why": "A página inicial carrega no navegador, mas retorna 403/406 para os user agents reais de rastreadores de IA, mesmo com o robots.txt permitindo.",
    "fix": "Libere esses user agents nas regras de proteção contra bots e rastreie de novo."
  },
  "BROKEN_INTERNAL_LINK": {
    "title": "Link interno quebrado",
    "why": "Rastreadores seguem links para descobrir páginas; links quebrados desperdiçam requisições e escondem o destino.",
    "fix": "Aponte o link para a URL ativa ou remova-o."
  },
  "CANONICAL_MISMATCH": {
    "title": "Canonical aponta para outra URL",
    "why": "Os sinais são consolidados no destino, então esta página não será a exibida.",
    "fix": "Confirme se a consolidação é intencional; caso contrário, aponte o canonical para esta URL."
  },
  "CLIENT_ERROR": {
    "title": "Página retorna 4xx",
    "why": "URLs 4xx não são indexadas; se estiverem em links ou no sitemap, desperdiçam rastreamento.",
    "fix": "Restaure a página ou remova-a dos links e do sitemap e redirecione para a página ativa mais próxima."
  },
  "DUPLICATE_BODY": {
    "title": "Páginas com texto quase idêntico",
    "why": "Mecanismos de busca indexam só uma versão de conteúdo duplicado; a que você quer pode perder.",
    "fix": "Mantenha uma versão canônica e aplique 301 ou canonical nas demais."
  },
  "DUPLICATE_META_DESCRIPTION": {
    "title": "Páginas com a mesma meta description",
    "why": "O Google recomenda uma descrição única por página.",
    "fix": "Escreva uma descrição específica por página ou remova a duplicada."
  },
  "DUPLICATE_TITLE": {
    "title": "Páginas com o mesmo título",
    "why": "Títulos idênticos dificultam distinguir as páginas e costumam indicar URLs duplicadas.",
    "fix": "Escreva um título distinto por página ou consolide duplicatas com canonical ou 301."
  },
  "FEW_EXTERNAL_LINKS": {
    "title": "Cita poucas fontes",
    "why": "Citar fontes esteve entre as reescritas mais eficazes no experimento GEO-bench.",
    "fix": "Adicione links para as fontes das suas afirmações."
  },
  "FEW_H2": {
    "title": "Poucas seções",
    "why": "As páginas mais citadas (quartil superior) tinham em média 10,6 títulos. Apenas correlação.",
    "fix": "Divida o conteúdo longo em seções que respondam a uma pergunta cada."
  },
  "HREFLANG_LOW": {
    "title": "Site multilíngue com pouco hreflang",
    "why": "O hreflang diz aos mecanismos qual versão de idioma exibir; sem ele, as versões competem entre si.",
    "fix": "Adicione links hreflang recíprocos entre as versões de idioma."
  },
  "IMAGES_MISSING_ALT": {
    "title": "Imagens sem texto alternativo",
    "why": "O Google usa o texto alt para entender imagens, e a acessibilidade exige isso.",
    "fix": "Descreva imagens informativas no alt; use alt=\"\" nas decorativas."
  },
  "LANG_IMBALANCE": {
    "title": "Idiomas muito desiguais",
    "why": "O site publica em chinês e inglês, mas um dos idiomas tem bem menos páginas de conteúdo.",
    "fix": "Reduza a diferença no idioma mais fraco, começando pelas páginas ligadas às suas perguntas."
  },
  "LLMS_TXT_BROKEN_LINKS": {
    "title": "llms.txt aponta para páginas quebradas ou bloqueadas",
    "why": "Um mapa que aponta para 404 ou caminhos bloqueados é pior do que nenhum.",
    "fix": "Corrija ou remova as entradas quebradas."
  },
  "LOW_CONTENT_PAGE": {
    "title": "Página funcional com pouco texto",
    "why": "Páginas de login, carrinho e contato costumam ser curtas; aparecem aqui só para não serem confundidas com páginas vazias.",
    "fix": "Nenhuma ação necessária, a menos que a página precise ranquear."
  },
  "LOW_LIST_DENSITY": {
    "title": "Quase sem listas",
    "why": "Páginas mais citadas tinham densidade de listas de 0,43 contra 0,05 nas menos citadas. Apenas correlação.",
    "fix": "Transforme enumerações em listas ul/ol."
  },
  "LOW_RELEVANCE": {
    "title": "Títulos não usam as palavras das perguntas",
    "why": "A relevância para a consulta foi o preditor isolado mais forte de influência (r = 0,432).",
    "fix": "Use as palavras dos compradores no title, no H1 e nos H2."
  },
  "MISSING_H1": {
    "title": "Página de conteúdo sem H1",
    "why": "O título principal é um dos sinais que o Google usa para os links de título e mostra o tema ao leitor.",
    "fix": "Adicione um H1 que diga o tema da página."
  },
  "MISSING_META_DESCRIPTION": {
    "title": "Sem meta description",
    "why": "O Google pode usar a descrição no snippet; sem ela, o trecho vem do texto da página.",
    "fix": "Adicione um resumo de uma ou duas frases."
  },
  "MISSING_TITLE": {
    "title": "Sem <title>",
    "why": "O elemento title é a principal fonte do Google para os links de título.",
    "fix": "Adicione um título único e descritivo."
  },
  "MULTIPLE_H1": {
    "title": "Mais de um H1",
    "why": "Geralmente é erro de template, como o logo marcado como H1. O Google não penaliza; listado para clareza.",
    "fix": "Mantenha um único H1 para o título principal."
  },
  "NOINDEX": {
    "title": "Página com meta robots noindex",
    "why": "noindex diz aos mecanismos para não mostrar a página, o que também a exclui dos recursos de IA baseados no índice.",
    "fix": "Remova o noindex das páginas que devem ser encontradas."
  },
  "NON_200_STATUS": {
    "title": "Página retorna código de sucesso diferente de 200 ou redirecionamento",
    "why": "Rastreadores tratam respostas 202 e 3xx de forma diferente de 200.",
    "fix": "Sirva a URL canônica diretamente com 200."
  },
  "NO_AUTHOR_ENTITY": {
    "title": "Marcação de artigo sem autor",
    "why": "As diretrizes do Google perguntam quem criou o conteúdo; a marcação Article pode informar.",
    "fix": "Adicione author à marcação Article e mostre a assinatura."
  },
  "NO_CANONICAL": {
    "title": "Sem link canonical",
    "why": "Sem ele, os mecanismos escolhem sozinhos a URL canônica quando há duplicatas.",
    "fix": "Adicione um rel=canonical autorreferente."
  },
  "NO_COMPARISON": {
    "title": "Sem comparação",
    "why": "Páginas comparativas tiveram influência média 55,3% maior no conjunto de dados de absorção.",
    "fix": "Adicione uma tabela comparando as opções pelos mesmos critérios."
  },
  "NO_DATE": {
    "title": "Sem data visível",
    "why": "No estudo de mecanismos chineses, o conteúdo citado para consultas sensíveis ao tempo teve meia-vida de cerca de 39 dias; a data ajuda a julgar a atualidade.",
    "fix": "Mostre as datas de publicação e atualização e inclua-as na marcação Article."
  },
  "NO_DEFINITION": {
    "title": "Sem frase de definição",
    "why": "Páginas com definição tiveram influência média 57,3% maior no conjunto de dados de absorção.",
    "fix": "Comece com uma frase que diga o que é o assunto."
  },
  "NO_HOWTO": {
    "title": "Sem passos",
    "why": "Conteúdo de passo a passo teve influência média 41,2% maior no conjunto de dados de absorção.",
    "fix": "Adicione passos numerados onde o leitor precisa agir."
  },
  "NO_JSONLD": {
    "title": "Sem dados estruturados",
    "why": "Dados estruturados ajudam os mecanismos a classificar a página. O Google diz que não são obrigatórios para recursos de IA.",
    "fix": "Adicione Organization na página inicial e Article ou Product onde fizer sentido."
  },
  "NO_LLMS_TXT": {
    "title": "Sem /llms.txt",
    "why": "O llms.txt é uma proposta de 2024, não um padrão, e nenhum mecanismo documenta que o lê. Custo baixo, benefício incerto.",
    "fix": "Opcional: publique um /llms.txt gerado a partir dos fatos da marca."
  },
  "NO_NUMBERS": {
    "title": "Poucos números concretos",
    "why": "Adicionar estatísticas esteve entre as reescritas mais eficazes no GEO-bench; páginas com números tiveram influência 61,6% maior no conjunto de absorção.",
    "fix": "Adicione números com fonte, unidade e data."
  },
  "NO_QUOTABLE_PASSAGE": {
    "title": "Sem trecho autossuficiente",
    "why": "A recuperação escolhe trechos, não páginas. Regra: uma seção de ao menos 60 palavras com número, definição ou passo. A regra é deste projeto.",
    "fix": "Reescreva duas ou três seções-chave para que a primeira frase responda ao título."
  },
  "NO_SITEMAP": {
    "title": "Sem sitemap.xml",
    "why": "O sitemap diz aos rastreadores quais URLs importam; sites grandes ou mal interligados são os que mais ganham.",
    "fix": "Publique um /sitemap.xml só com URLs canônicas."
  },
  "ORPHAN_PAGE": {
    "title": "Nenhuma página rastreada aponta para cá",
    "why": "Páginas acessíveis só pelo sitemap são mais difíceis de descobrir e recebem pouco contexto interno.",
    "fix": "Adicione um link a partir de uma página relacionada."
  },
  "PAGE_UNREACHABLE": {
    "title": "Página não pôde ser acessada",
    "why": "Erros de rede e timeouts significam que nenhum rastreador recebe o conteúdo.",
    "fix": "Verifique DNS, TLS e os logs do servidor para esta URL."
  },
  "REDIRECT_CHAIN": {
    "title": "Dois ou mais redirecionamentos antes da página",
    "why": "Cada salto aumenta a latência, e rastreadores param de seguir cadeias longas.",
    "fix": "Redirecione direto para a URL final e atualize os links internos."
  },
  "ROBOTS_BLOCKS_AI": {
    "title": "robots.txt bloqueia rastreadores de IA",
    "why": "Pela RFC 9309, um grupo Disallow correspondente impede esse rastreador de acessar os caminhos listados.",
    "fix": "Adicione um grupo User-agent que permita GPTBot, OAI-SearchBot, ClaudeBot, PerplexityBot, Google-Extended e os rastreadores regionais que você quer atingir."
  },
  "SCHEMA_CONTENT_MISMATCH": {
    "title": "Dados estruturados não batem com o conteúdo visível",
    "why": "As diretrizes do Google exigem que a marcação descreva conteúdo visível ao usuário.",
    "fix": "Remova a marcação ou adicione o conteúdo visível correspondente."
  },
  "SERVER_ERROR": {
    "title": "Erro de servidor (5xx)",
    "why": "O Google documenta que respostas 5xx repetidas reduzem o rastreamento e podem tirar URLs do índice.",
    "fix": "Corrija o erro do servidor ou retorne 404/410 se a página não existe mais."
  },
  "SHORT_CONTENT": {
    "title": "Página de conteúdo curta",
    "why": "No conjunto de citações, as páginas do quartil superior tinham em média 1.943 palavras e as do inferior 170. É correlação, não meta.",
    "fix": "Acrescente substância útil (números, comparações, passos), não enchimento."
  },
  "SITEMAP_LOW_VALUE_URLS": {
    "title": "Sitemap lista URLs de parâmetro, busca ou paginação",
    "why": "Sitemaps devem listar só as URLs canônicas que você quer indexadas.",
    "fix": "Remova essas URLs do sitemap."
  },
  "SITEMAP_NOT_DECLARED": {
    "title": "robots.txt não declara o sitemap",
    "why": "Uma linha Sitemap: permite que qualquer rastreador encontre o sitemap sem aviso.",
    "fix": "Adicione `Sitemap: https://example.com/sitemap.xml` ao robots.txt."
  },
  "SLOW_RESPONSE": {
    "title": "Resposta lenta do servidor",
    "why": "O HTML levou mais de 3 segundos; rastreadores buscam menos páginas em hosts lentos. O limite de 3 s é regra deste projeto.",
    "fix": "Faça cache do HTML na borda ou investigue a rota lenta."
  },
  "SPA_SHELL": {
    "title": "HTML estático sem conteúdo",
    "why": "O Google renderiza JavaScript numa fila atrasada e muitos rastreadores de IA leem só HTML estático, então uma página só no cliente parece vazia.",
    "fix": "Use renderização no servidor ou pré-renderização nas páginas de conteúdo."
  },
  "TITLE_TOO_LONG": {
    "title": "Título longo",
    "why": "O Google não tem limite de tamanho, mas corta pela largura de exibição. Limite aqui: cerca de 60 caracteres latinos ou 32 CJK.",
    "fix": "Coloque a frase-chave primeiro."
  },
  "XROBOTS_NOINDEX": {
    "title": "Cabeçalho X-Robots-Tag com noindex",
    "why": "O cabeçalho tem o mesmo efeito da meta tag, mas não aparece no código-fonte; costuma ser injetado pelo CDN.",
    "fix": "Remova a regra do cabeçalho no servidor ou no CDN."
  }
};
