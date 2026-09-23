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

Build com `make build` (o `go build` puro precisa do `frontend/dist`).

## Configuração

`~/.config/skye/config.toml`:

    sound = true

    [[preset]]
    name = "blog"
    command = "cd ~/projects/blog && claude"

O comando roda em `$SHELL -lic`, com o seu `.bashrc`/`.zshrc` carregado. Quando ele
termina, o terminal continua como shell.

O latido é `frontend/public/bark.ogg`; sem o arquivo, a skye toca um bipe.

## Uso

- **+** abre um terminal vazio ou um preset.
- O composer embaixo do terminal: `Enter` quebra linha, `Ctrl+Enter` envia.
- No terminal: `Alt+Enter` quebra linha no claude; selecionar copia; `Ctrl+Shift+V` cola.
- **×** esconde a janela (os terminais continuam); rodar `skye` de novo traz a janela.
- **sair** encerra todos os terminais; as conversas vão para *Encerradas*, com *retomar*.

## Abrir pelo navegador

    skye web              # ou: skye web --no-open

A skye sobe um servidor só em `127.0.0.1` (porta `web_port` do `config.toml`, padrão
`7810`), imprime o endereço com o token — `http://localhost:7810/?token=...` — e abre no
navegador padrão do Windows. O endereço grava um cookie e some da barra; depois disso,
`http://localhost:7810/` basta.

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

## Testes

    make test

## Conferência manual antes de uma release

1. Abrir terminal vazio e preset; `ls --color`, `git log` desenham certo.
2. `claude` no terminal: `Alt+Enter` quebra linha; composer com 3 linhas chega como um prompt.
3. Pedir algo que exija permissão: bolinha amarela, latido, toast com a janela fora de foco.
4. Resposta termina: bolinha verde, latido; um minuto depois, nenhum segundo latido.
5. `/clear`: a conversa anterior aparece em *Encerradas*.
6. Fechar a janela no ×, rodar `skye` de novo: mesma janela, terminais intactos.
7. `pkill skye` e reabrir: terminais reencontrados com os nomes.
8. **sair**, reabrir, *retomar*: `claude --resume` volta na pasta certa.
9. Esconder com × e provocar um pedido de permissão: o toast aparece.
10. Digitar rápido enquanto outro terminal despeja saída: nada fora de ordem.
11. Com a skye escondida, rodar `skye` de novo: a mesma janela volta.
