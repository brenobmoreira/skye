# skye

Painel de desktop para as sessões do Claude Code rodando no WSL. Cada terminal vive num
tmux só da skye e aparece na janela ao vivo; quando uma sessão para esperando você, a
skye late.

## Requisitos

- WSL2 com WSLg (`guiApplications=true` no `.wslconfig`, depois `wsl --shutdown`)
- Go ≥ 1.27.1, Node ≥ 20, tmux ≥ 3.2
- `sudo apt install -y pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev`

## Build e instalação

    make build
    install -m 755 skye ~/.local/bin/skye
    skye install-hooks        # uma vez; depois reinicie as sessões do claude

O `install-hooks` também envolve o seu `statusLine`: ele continua desenhando igual, e uma cópia
do JSON vai para a skye, que mostra o uso do plano (5h e 7d) no rodapé da lista de terminais. Rode de novo
depois de trocar o comando do `statusLine`.

Build com `make build` (o `go build` puro precisa do `frontend/dist`).

## Configuração

`~/.config/skye/config.toml`:

    sound = true
    web_port = 7810           # porta do `skye web` (1024–65535)

    [[preset]]
    name = "blog"
    command = "cd ~/projects/blog && claude"

O comando roda em `$SHELL -lic`, com o seu `.bashrc`/`.zshrc` carregado. Quando ele
termina, o terminal continua como shell.

Para abrir uma branch nova numa worktree própria, cadastre o repositório:

    [[repo]]
    name = "livia"
    path = "~/projects/ai_livia_copilot"
    base = "origin/develop"   # de onde sai a branch; padrão: o HEAD do repo (sem fetch)
    dir = "~/projects"        # onde nasce a worktree; padrão: a pasta que contém o repo
    command = "claude"        # o que roda nela; padrão: claude

No **+**, *nova worktree de livia…* pede o nome da branch, cria `~/projects/<branch>` (com `/`
trocada por `-`) e abre o terminal lá. A skye nunca apaga worktree, nem ao fechar o terminal.

O latido é `frontend/public/bark.ogg`; sem o arquivo, a skye toca um bipe.

## Uso

- **+** abre um *Novo terminal*, um preset ou um path salvo. *+ Adicionar path* pede um nome
  (ex.: `LivIA`) e uma pasta (ex.: `~/projects/ai_livia_copilot`); o nome passa a ficar no topo do
  menu e abre o `claude` naquela pasta. O × ao passar o mouse remove (clique de novo para
  confirmar). Os paths ficam em `~/.config/skye/paths.json`.
- Na lista, um terminal vazio leva o nome da sessão do claude (a 1ª linha do 1º prompt) ou,
  sem sessão, o da pasta. Duplo clique no nome renomeia; o × ao lado do nome fecha o terminal.
  Um ponto azul marca o terminal que terminou ou pediu algo enquanto você olhava outro.
  Quem está esperando você mostra, embaixo do nome e no aviso do Windows, o que o claude pede
  (ex.: `Bash: git push`); quem está trabalhando mostra se está compactando ou quantos
  subagentes tem rodando. Depois de atualizar a skye, rode `skye install-hooks` de novo.
  Uma barra fina no pé do item mostra quanto do contexto a sessão já usou (amarela a partir de
  80%); passar o mouse mostra contexto, modelo e custo. À direita, há quanto tempo o terminal está no estado ("12 min"); depois de reiniciar a skye,
  o tempo volta no próximo evento do claude.
- A lista agrupa por estado: esperando você, terminou, trabalhando, terminal. Quem acaba de
  terminar ou pedir algo sobe para o topo do seu grupo. Arraste um item para mudar a ordem dentro
  do grupo; a ordem fica salva.
- `Ctrl+clique` num terminal da lista abre ele ao lado do atual; o painel com contorno azul recebe
  o teclado (clique dentro dele para trocar) e o `×` no canto volta a um terminal só. Com a janela
  estreita (menos de 1100 px), só o painel em foco aparece.
- Teclado: `Ctrl+1` … `Ctrl+9` abre o 1º … 9º terminal da lista; `Ctrl+Shift+Espaço` pula para o
  próximo que está esperando você (sem nenhum esperando, para o próximo com ponto azul). Numa aba
  do navegador, `Ctrl+1..9` fica com o navegador; no `skye.exe` e no app instalado funciona.
- No terminal: `Alt+Enter` quebra linha no claude; selecionar copia; `Ctrl+Shift+V` cola;
  `Ctrl+clique` num link abre no seu navegador (no WSL, o navegador padrão do Windows).
- **monitor** (na barra de cima) troca os terminais por uma aba com a memória do Windows (a que
  acaba primeiro: inclui o que a VM do WSL segura; lida por um `powershell.exe` que só roda com a
  aba aberta), a memória dentro do WSL, quanto
  cada terminal usa (a árvore de processos inteira) e o que ficou para trás: processo que saiu de
  um terminal e continuou vivo, `claude` aberto fora da skye, outra skye, janela do tmux sem item e
  sockets de tmux mortos. Só mostra; para encerrar, `kill <PID>`. O gráfico das últimas 3 h aparece
  quando o mcp-sysagent está gravando o histórico (`~/.local/share/mcp-sysagent/history`).
  Clicar num terminal volta para ele.
- **×** esconde a janela (os terminais continuam); rodar `skye` de novo traz a janela.
- **sair** encerra todos os terminais; as conversas vão para *Encerradas*, com *retomar*.
  *Encerradas* começa recolhida; clique no título para abrir (a escolha fica lembrada). Aberta,
  o campo *filtrar* acha a conversa por palavras do título ou da pasta (`Esc` limpa).

## Abrir pelo navegador

    skye web              # ou: skye web --no-open

A skye sobe um servidor só em `127.0.0.1` (porta `web_port` do `config.toml`, padrão
`7810`), imprime o endereço com o token — `http://127.0.0.1:7810/?token=...` — e abre no
navegador padrão do Windows. O endereço grava um cookie e some da barra; depois disso,
`http://127.0.0.1:7810/` basta.

Para instalar como aplicativo: no Edge, menu → *Aplicativos* → *Instalar skye*; no
Chrome, menu → *Transmitir, salvar e compartilhar* → *Instalar página como app* (ou
"Instalar este site como aplicativo"). O app instalado abre numa janela própria.

- A janela (`skye`) e o navegador (`skye web`) não rodam ao mesmo tempo: quem abre
  primeiro fica com os hooks; o segundo avisa e sai.
- `Ctrl+C` no `skye web` para o servidor; os terminais continuam no tmux. **sair** encerra
  os terminais e o `skye web`.
- No navegador não há minimizar, maximizar nem ×.
- O token fica em `~/.config/skye/web-token`. Apague o arquivo para revogar o app
  instalado; o próximo `skye web` cria outro e imprime o endereço novo.

## App do Windows (skye.exe)

Uma janela nativa do Windows (WebView2) com a mesma interface. O motor continua no WSL: o
`skye.exe` só mostra a janela e fala com o `skye web`.

    make windows              # gera ./skye.exe
    make install-windows      # copia para %LOCALAPPDATA%\skye\skye.exe e cria os atalhos
    make windows-shortcuts    # só recria os atalhos

O ícone do `skye.exe` (janela e barra de tarefas) é o logo em `cmd/skye-win/winres/icon.png`;
depois de trocar a imagem, rode `make windows-icon`. Se a barra de tarefas mostrar o ícone antigo,
desafixe e fixe de novo: o Windows guarda o ícone em cache.

O `install-windows` cria o atalho `skye` na área de trabalho e no Menu Iniciar (a tecla Windows
e "skye" acham o app). Para fixar na barra de tarefas, abra a skye e clique com o botão direito no
ícone da barra → *Fixar na barra de tarefas*. Feche o `skye.exe` antes de rodar
`make install-windows` de novo (o Windows não deixa sobrescrever o arquivo aberto).

- Precisa da skye instalada no WSL (`~/.local/bin/skye`, ver *Build e instalação*).
- Se o `skye web` não estiver de pé, o `skye.exe` sobe ele sozinho e escondido
  (`wsl.exe -e bash -lc 'skye web --no-open'`); não é preciso deixar terminal aberto.
- Fechar a janela no ✕ fecha só o `skye.exe`: o `skye web` e os terminais continuam no WSL,
  e reabrir mostra tudo como estava. **sair** encerra os terminais, o `skye web` e a janela.
- Se algo falhar, a janela mostra o comando que falhou e a saída dele, com *tentar de novo*.
  A saída do `skye web` que o `skye.exe` subiu fica em `%LOCALAPPDATA%\skye\web.log`.
- Porta: `7810`, ou a da variável de ambiente `SKYE_WEB_PORT` do Windows. Ela precisa bater
  com o `web_port` do `config.toml`.
- A janela Linux (`skye`) e o `skye web` não rodam ao mesmo tempo: com a janela Linux aberta,
  o `skye.exe` mostra o erro do `skye web` que não subiu.

Se a janela mostrar "o servidor respondeu na porta N, mas o WebSocket … não abriu":
confira se a porta bate com o `web_port`, se não há uma janela Linux da skye aberta e se o
`skye web` do WSL é desta versão (ele precisa aceitar a origem `http://wails.localhost`).
Se nada disso resolver, o WebView2 pode estar bloqueando o acesso a `127.0.0.1` (Local
Network Access do Chromium); enquanto isso, use `skye web` no navegador.

## Testes

    make test

## Conferência manual antes de uma release

1. Abrir terminal vazio e preset; `ls --color`, `git log` desenham certo.
2. `claude` no terminal: `Alt+Enter` quebra linha; colar 3 linhas com `Ctrl+Shift+V` chega como um prompt.
3. Pedir algo que exija permissão: cachorro amarelo, latido, toast com a janela fora de foco; o item e o toast mostram a ferramenta e o comando (`Bash: …`).
4. Resposta termina: cachorro verde, latido; um minuto depois, nenhum segundo latido.
5. `/clear`: o terminal continua o mesmo, sem nada novo em *Encerradas*; o próximo prompt vira o nome. Sair do claude (`/exit`) manda a conversa para *Encerradas*.
6. Fechar a janela no ×, rodar `skye` de novo: mesma janela, terminais intactos.
7. `pkill skye` e reabrir: terminais reencontrados com os nomes.
8. **sair**, reabrir, *retomar*: `claude --resume` volta na pasta certa.
9. Esconder com × e provocar um pedido de permissão: o toast aparece.
10. Digitar rápido enquanto outro terminal despeja saída: nada fora de ordem.
11. Com a skye escondida, rodar `skye` de novo: a mesma janela volta.
12. Terminal vazio com `claude`: o nome na lista vira o 1º prompt; ao sair do claude, volta à pasta.
13. Pedir algo num terminal e trocar para outro: ao terminar, o primeiro ganha o ponto azul; abri-lo tira o ponto.
14. Arrastar um terminal para cima de outro do mesmo grupo: troca de lugar; `pkill skye` e reabrir mantém a ordem.
15. `echo https://example.com` e `Ctrl+clique` no link: abre no navegador do Windows (janela Linux, `skye.exe` e navegador); clique simples não abre.
16. Depois de `skye install-hooks`, mandar um prompt: a lista de terminais mostra, no rodapé, `5h` e `7d` com barra e horário de renovação; o statusline do terminal continua igual.
17. **monitor**: os terminais aparecem com processos e memória; rodar `setsid sleep 600 &` num terminal e fechá-lo: o `sleep` aparece em *Fora da lista* como processo solto.
