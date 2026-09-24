# skye

Painel de desktop para as sessões do Claude Code no WSL. Cada terminal vive num tmux só da skye
e aparece na janela ao vivo; quando uma sessão termina ou para esperando você, a skye late.

- [Instalar](#instalar) · [Atualizar](#atualizar) · [Usar](#usar) · [Atalhos](#atalhos) ·
  [Configurar](#configurar) · [Navegador](#abrir-pelo-navegador) ·
  [Problemas comuns](#problemas-comuns) · [Desenvolvimento](#desenvolvimento)

## Instalar

Requisitos:

- WSL2 com WSLg (`guiApplications=true` no `.wslconfig`, depois `wsl --shutdown`)
- Go ≥ 1.27.1, Node ≥ 20, tmux ≥ 3.2
- `sudo apt install -y pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev`

No WSL:

    make build
    install -m 755 skye ~/.local/bin/skye
    skye install-hooks        # depois, reinicie as sessões do claude
    make install-windows      # app do Windows + atalhos na área de trabalho e no Menu Iniciar

Abra a skye pelo atalho **skye** (ou tecla Windows e "skye"). Para fixar na barra de tarefas,
com ela aberta: botão direito no ícone → *Fixar na barra de tarefas*.

O `install-hooks` grava os hooks no `~/.claude/settings.json` (com backup `.bak`) e envolve o seu
`statusLine`: ele continua desenhando igual, e uma cópia do JSON vai para a skye (uso do plano,
contexto de cada sessão). Rode de novo se trocar o comando do `statusLine`.

## Atualizar

Os terminais sobrevivem a tudo abaixo (eles vivem no tmux). Com o `skye.exe` fechado:

    make build
    install -m 755 skye ~/.local/bin/skye
    make install-windows
    kill $(pgrep -x skye)     # o skye web antigo; fechar o skye.exe não o encerra
    skye install-hooks        # só quando a versão nova traz hooks novos; não faz mal repetir

Depois, abra a skye: o `skye.exe` sobe o `skye web` novo e reencontra os terminais. Se o
`install-hooks` mudou algo, reinicie as sessões do claude para elas passarem a reportar.

Não use **sair** para atualizar: ele encerra os terminais.

## Usar

A skye abre de três jeitos, com a mesma interface:

| Jeito | Como | Quando |
|---|---|---|
| App do Windows | atalho **skye** (`skye.exe`) | o normal |
| Janela Linux | `skye` no WSL (pelo WSLg) | sem o app do Windows |
| Navegador | `skye web` ([detalhes](#abrir-pelo-navegador)) | de outro navegador ou como PWA |

A janela Linux e o `skye web` não rodam ao mesmo tempo: quem abre primeiro fica com os hooks.
O app do Windows é só uma janela para o `skye web`.

### A lista de terminais

Cada item é um terminal. O cachorro mostra o estado, e a lista agrupa nesta ordem:

| Cachorro | Estado |
|---|---|
| amarelo | esperando você (permissão ou pergunta) |
| verde | terminou |
| azul | trabalhando |
| cinza | terminal sem claude |

- **Ordem:** quem acaba de terminar ou pedir algo sobe para o topo do seu grupo. Arraste um item
  para mudar a ordem dentro do grupo; a ordem fica salva.
- **Nome:** um terminal aberto sem nome leva a 1ª linha do 1º prompt do claude; sem sessão, o
  nome da pasta. Duplo clique renomeia; o × ao lado do nome fecha o terminal.
- **Linha de baixo:** o que o claude pede quando espera você (ex.: `Bash: git push`); se está
  compactando ou quantos subagentes tem rodando quando trabalha; senão, o título ou a pasta.
- **À direita:** há quanto tempo está no estado ("12 min"). Depois de reiniciar a skye, volta no
  próximo evento do claude.
- **Barra fina no pé:** quanto do contexto a sessão já usou; amarela a partir de 80%. Passar o
  mouse mostra contexto, modelo e custo.
- **Ponto azul:** terminou ou pediu algo enquanto você olhava outro terminal. Some ao abrir.
- **Rodapé:** o uso do plano (5h e 7d) com o horário de renovação. Atualiza a cada resposta de
  qualquer sessão; depois de reiniciar a skye, aparece na próxima resposta.
- **Encerradas:** as conversas que acabaram (sair do claude, fechar o terminal, **sair**).
  Começa recolhida; clique no título para abrir. *filtrar* acha pelo título ou pela pasta.
  Clicar numa conversa a retoma (`claude --resume` na pasta certa); o × a esquece.

Quando um terminal termina ou pede algo, a skye late (🔔 na barra liga e desliga) e, com a janela
fora de foco, mostra um aviso do Windows com o que o claude pede.

### O menu +

    LivIA                ← paths salvos: abrem o claude na pasta
    ───────────────
    Novo terminal        ← shell na home
    blog                 ← presets do config.toml
    nova worktree de…    ← repos do config.toml
    ───────────────
    + Adicionar path

- **+ Adicionar path:** nome (ex.: `LivIA`) e pasta (ex.: `~/projects/ai_livia_copilot`). A
  pasta precisa existir. O path aparece ao passar o mouse; o × remove (clique de novo para
  confirmar). Ficam em `~/.config/skye/paths.json`.
- **nova worktree de X…:** pede o nome da branch, cria a worktree (ver [`[[repo]]`](#configurar))
  e abre o terminal nela. A skye nunca apaga worktree, nem ao fechar o terminal.

### Lado a lado

`Ctrl+clique` num terminal da lista abre ele ao lado do atual. O painel com contorno azul recebe
o teclado (clique dentro do outro para trocar); o × no canto de cima volta a um terminal só. Com
a janela abaixo de 1100 px, só o painel em foco aparece.

### Monitor

O botão **monitor** na barra de cima troca os terminais por uma aba que atualiza a cada 3 s:

- **Windows · disponível:** a memória que acaba primeiro — inclui o que a VM do WSL segura como
  cache. É o mesmo número do Gerenciador de Tarefas.
- **WSL · disponível**, swap, pressão de memória e carga, de dentro da VM (teto do `.wslconfig`).
- **Gráfico das últimas 3 h** da memória no WSL, quando o mcp-sysagent está gravando o
  histórico (`~/.local/share/mcp-sysagent/history`).
- **Terminais da skye:** processos e memória de cada um (a árvore inteira: claude, MCPs, o que
  ele subiu). Clicar abre o terminal.
- **Fora da lista:** o que ficou para trás — processo que saiu de um terminal e continuou vivo,
  `claude` aberto fora da skye, outra skye, janela do tmux sem item.
- **Outros servidores tmux:** sockets que não são o da skye; os mortos podem ser apagados.

O monitor só mostra; para encerrar algo, `kill <PID>` num terminal.

### Fechar e sair

- **×** (ou ✕ no app do Windows) fecha só a janela: o `skye web` e os terminais continuam, e
  reabrir mostra tudo como estava. Na janela Linux, rodar `skye` de novo traz a janela.
- **sair** encerra todos os terminais, o `skye web` e a janela. As conversas vão para
  *Encerradas*.

## Atalhos

| Onde | Tecla | Faz |
|---|---|---|
| skye | `Ctrl+1` … `Ctrl+9` | abre o 1º … 9º terminal da lista |
| skye | `Ctrl+Shift+Espaço` | pula para o próximo que espera você (sem nenhum, o próximo com ponto azul) |
| skye | `Ctrl+=` / `Ctrl+-` / `Ctrl+0` | aumenta / diminui / volta o tamanho da fonte (também `Ctrl+roda do mouse`) |
| lista | `Ctrl+clique` | abre o terminal ao lado |
| lista | duplo clique no nome | renomeia |
| terminal | `Alt+Enter` | quebra linha no claude |
| terminal | selecionar | copia |
| terminal | `Ctrl+Shift+V` | cola |
| terminal | `Ctrl+clique` num link | abre no navegador padrão do Windows |

Numa aba do navegador, `Ctrl+1..9` fica com o navegador; no app do Windows e no app instalado
funciona. A fonte do terminal (JetBrains Mono ou Cascadia Mono) muda no ⚙; fonte e tamanho ficam
lembrados por aparelho.

## Configurar

`~/.config/skye/config.toml` (todos os campos são opcionais):

    sound = true              # latido ao terminar ou pedir algo
    web_port = 7810           # porta do skye web (1024–65535)
    shell = "/bin/zsh"        # padrão: o $SHELL

    [[preset]]                # item fixo no +
    name = "blog"
    command = "cd ~/projects/blog && claude"

    [[repo]]                  # "nova worktree de livia…" no +
    name = "livia"
    path = "~/projects/ai_livia_copilot"
    base = "origin/develop"   # de onde sai a branch; padrão: o HEAD do repo (sem fetch)
    dir = "~/projects"        # onde nasce a worktree; padrão: a pasta que contém o repo
    command = "claude"        # o que roda nela; padrão: claude

O `command` de preset e repo roda em `$SHELL -lic`, com o seu `.bashrc`/`.zshrc` carregado;
quando termina, o terminal continua como shell. A worktree nasce em `<dir>/<branch>`, com `/`
trocada por `-`. O `config.toml` é lido quando o `skye web` sobe: depois de mudar, reinicie-o
(`kill $(pgrep -x skye)` e abra a skye).

Outros arquivos:

| Arquivo | O que é |
|---|---|
| `~/.config/skye/paths.json` | paths salvos do + (pode editar à mão) |
| `~/.config/skye/web-token` | token do `skye web`; apagar revoga os apps instalados |
| `~/.config/skye/tmux.conf` | configuração do tmux da skye |
| `~/.local/state/skye/conversations.json` | *Encerradas* (as 50 mais recentes) |
| `~/.local/state/skye/launch/` | um arquivo por terminal aberto, para reencontrar depois de reiniciar |
| `frontend/public/bark.ogg` | o latido; sem ele, a skye toca um bipe |

## Abrir pelo navegador

    skye web              # ou: skye web --no-open

A skye sobe um servidor só em `127.0.0.1` (porta `web_port`, padrão `7810`), imprime o endereço
com o token — `http://127.0.0.1:7810/?token=...` — e abre no navegador padrão do Windows. O
endereço grava um cookie e some da barra; depois disso, `http://127.0.0.1:7810/` basta.

Para instalar como aplicativo: no Edge, menu → *Aplicativos* → *Instalar skye*; no Chrome, menu →
*Transmitir, salvar e compartilhar* → *Instalar página como app*. O app instalado abre numa
janela própria.

- `Ctrl+C` no `skye web` para o servidor; os terminais continuam no tmux.
- No navegador não há minimizar, maximizar nem ×.
- O token fica em `~/.config/skye/web-token`. Apague o arquivo para revogar o app instalado; o
  próximo `skye web` cria outro e imprime o endereço novo.

### Como o app do Windows se conecta

O `skye.exe` (WebView2) só mostra a janela; o motor continua no WSL. Se o `skye web` não estiver
de pé, o `skye.exe` sobe ele sozinho e escondido (`wsl.exe -e bash -lc 'skye web --no-open'`),
e a saída vai para `%LOCALAPPDATA%\skye\web.log`. A porta é `7810`, ou a da variável de
ambiente `SKYE_WEB_PORT` do Windows; ela precisa bater com o `web_port`.

### O que a skye roda no Windows

Para quem cuida da segurança da máquina (EDR, Wazuh):

- `powershell.exe -NoProfile -NonInteractive -Command -`, com o script na entrada padrão (nada
  codificado na linha de comando): mostra o aviso do Windows e, com a aba monitor aberta, lê a
  memória do Windows a cada 5 s (um processo só, que sai sozinho em até 5 min).
- `skye.exe` sem assinatura, em `%LOCALAPPDATA%\skye\`.
- Só ao rodar o `make`: `cmd.exe /c echo %LOCALAPPDATA%` e um `powershell.exe` que cria os
  atalhos `skye.lnk`.

## Problemas comuns

**`método desconhecido "X"`** — o app do Windows é mais novo que o `skye web` que está rodando.
Veja [Atualizar](#atualizar): `kill $(pgrep -x skye)` e abra a skye de novo.

**A lista não muda de estado, ou não aparece o pedido/contexto** — rode `skye install-hooks` e
reinicie as sessões do claude: as que já estavam abertas podem continuar com os hooks antigos.

**O uso de 5h/7d sumiu** — depois de reiniciar a skye, ele volta na próxima resposta de
qualquer sessão.

**"o servidor respondeu na porta N, mas o WebSocket … não abriu"** — confira se a porta bate com
o `web_port`, se não há uma janela Linux da skye aberta e se o `skye web` é desta versão (ele
precisa aceitar a origem `http://wails.localhost`). Se nada resolver, o WebView2 pode estar
bloqueando o acesso a `127.0.0.1` (Local Network Access do Chromium); enquanto isso, use
`skye web` no navegador.

**`make install-windows` não consegue copiar** — feche o `skye.exe`; o Windows não deixa
sobrescrever o arquivo aberto.

**A barra de tarefas mostra o ícone antigo** — desafixe e fixe de novo; o Windows guarda o ícone
em cache.

**O atalho não aparece na área de trabalho** — aperte F5 na área de trabalho. O atalho fica em
`C:\Users\<você>\Desktop\skye.lnk`; `make windows-shortcuts` o recria.

## Desenvolvimento

    make build                # frontend + binário Linux (o go build puro precisa do frontend/dist)
    make test                 # go test ./internal/... + vitest
    make windows              # só gera ./skye.exe
    make windows-shortcuts    # só recria os atalhos
    make windows-icon         # depois de trocar cmd/skye-win/winres/icon.png

Os testes de `cmd/skye` precisam das tags do build:
`go test -tags desktop,production,webkit2_41 ./cmd/skye`.

Specs e planos ficam em `docs/`.

### Conferência manual antes de uma release

Terminais e claude:

1. Abrir *Novo terminal* e um preset; `ls --color` e `git log` desenham certo.
2. `claude` no terminal: `Alt+Enter` quebra linha; colar 3 linhas com `Ctrl+Shift+V` chega como um prompt.
3. Digitar rápido enquanto outro terminal despeja saída: nada fora de ordem.
4. `echo https://example.com` e `Ctrl+clique` no link: abre no navegador do Windows (janela Linux, `skye.exe` e navegador); clique simples não abre.

Estados, avisos e a lista:

5. Pedir algo que exija permissão: cachorro amarelo, latido, aviso com a janela fora de foco; o item e o aviso mostram `Bash: …`.
6. Resposta termina: cachorro verde, latido; um minuto depois, nenhum segundo latido.
7. Pedir algo num terminal e trocar para outro: ao terminar, o primeiro ganha o ponto azul; abri-lo tira o ponto.
8. Terminal vazio com `claude`: o nome vira o 1º prompt; ao sair do claude, volta à pasta.
9. O tempo à direita do item zera quando o estado muda; a barra de contexto cresce com a conversa e o hover mostra modelo e custo.
10. `/compact` mostra "compactando"; um pedido com subagente mostra "1 subagente".
11. Mandar um prompt: o rodapé mostra `5h` e `7d` com barra e horário de renovação; o statusline do terminal continua igual.
12. Arrastar um terminal para cima de outro do mesmo grupo: troca de lugar; `pkill -x skye` e reabrir mantém a ordem.

Sessões e Encerradas:

13. `/clear`: o terminal continua o mesmo, sem nada novo em *Encerradas*; o próximo prompt vira o nome. `/exit` manda a conversa para *Encerradas*.
14. **sair**, reabrir, clicar na conversa em *Encerradas*: `claude --resume` volta na pasta certa.
15. *filtrar* em *Encerradas* acha por parte do título ou da pasta; `Esc` limpa.

Menu +, atalhos e lado a lado:

16. *+ Adicionar path* com uma pasta que não existe mostra o erro; com uma que existe, o nome aparece no topo e abre o claude nela; × e "remover?" o tiram.
17. *nova worktree de…* com um `[[repo]]` configurado cria a pasta e abre o terminal na branch nova.
18. `Ctrl+2` abre o 2º terminal; com um terminal esperando, `Ctrl+Shift+Espaço` pula para ele.
19. `Ctrl+clique` num item abre lado a lado; clicar num painel move o contorno azul; o × volta a um só.

Janela, reinício e monitor:

20. Fechar a janela no ×, reabrir: mesma janela, terminais intactos. Com a skye escondida, um pedido de permissão ainda mostra o aviso.
21. `pkill -x skye` e reabrir: terminais reencontrados com os nomes.
22. **monitor**: a caixa do Windows bate com o Gerenciador de Tarefas; os terminais aparecem com processos e memória; `setsid sleep 600 &` num terminal e fechá-lo: o `sleep` aparece em *Fora da lista*.
