# wingetprune

Ferramenta de linha de comando, escrita em Go, que percorre os pacotes instalados via [winget](https://learn.microsoft.com/windows/package-manager/winget/) **um por um**, pergunta se você quer mantê-lo ou desinstalá-lo e, no final, remove só os que você marcou.

## O que é

O winget não tem um modo interativo de revisão: você precisa saber o id do pacote e rodar `winget uninstall` manualmente para cada um. O `wingetprune` preenche essa lacuna. Ele lista tudo o que o winget consegue gerenciar no seu sistema, mostra cada pacote no formato `Nome - id` e deixa você decidir com uma tecla.

É um binário único, compilado, sem dependências além do próprio winget.

## Para que serve

- Fazer uma faxina em um Windows que acumulou programas ao longo do tempo.
- Revisar o que foi instalado via winget depois de configurar uma máquina nova.
- Evitar copiar e colar ids de pacote para desinstalar vários programas.

## Como funciona

1. Roda `winget export` para obter a lista de pacotes (JSON, bem mais confiável do que parsear texto).
2. Roda `winget list` para descobrir o nome legível de cada pacote. As colunas são detectadas pela linha de traços da tabela, então funciona com o winget em qualquer idioma.
3. Pergunta pacote por pacote o que fazer.
4. Mostra o resumo do que foi marcado e pede confirmação.
5. Roda `winget uninstall --id <id> -e` para cada pacote confirmado.

Nada é desinstalado antes da confirmação final.

## Requisitos

- Windows 10 ou 11 com o winget instalado (vem com o "App Installer" da Microsoft Store).
- [Go](https://go.dev/dl/) 1.21 ou superior, apenas para compilar.

## Instalação

Clone ou baixe a pasta do projeto e, dentro dela:

```
go mod init wingetprune
go build -ldflags="-s -w" -o winget-prune.exe
```

Isso gera o `wingetprune.exe`. Você pode movê-lo para qualquer pasta que esteja no seu `PATH`.

### Compilando a partir de outro sistema (cross-compile)

O Go consegue gerar o executável do Windows a partir de Linux ou macOS:

```
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o winget-prune.exe
```

No PowerShell, o equivalente é:

```
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -ldflags="-s -w" -o winget-prune.exe
```

## Uso

Teste primeiro sem desinstalar nada:

```
.\wingetprune.exe --dry-run
```

Quando estiver satisfeito, rode sem a flag:

```
.\wingetprune.exe
```

Para cada pacote aparece um prompt como este:

```
(12/87) Visual Studio Code - Microsoft.VisualStudioCode >
```

| Tecla   | Ação                                       |
| ------- | ------------------------------------------ |
| `Enter` | Manter o pacote                            |
| `u`     | Marcar para desinstalar                    |
| `i`     | Mostrar detalhes do pacote (`winget show`) |
| `q`     | Parar a revisão e ir para o resumo         |

Depois da revisão, o programa lista tudo o que foi marcado e pergunta `Confirmar? [s/N]`. Qualquer resposta diferente de `s` cancela.

## Suporte a outros sistemas operacionais

Hoje o `wingetprune` **só funciona no Windows**, porque depende do winget, que só existe nele. O binário pode ser compilado em qualquer SO (veja cross-compile acima), mas só roda onde o winget estiver disponível.

A ideia para uma versão universal é separar o programa em duas partes: a interface de revisão (que já é independente de gerenciador) e uma camada com duas operações, `List()` e `Uninstall(id)`, com uma implementação por gerenciador de pacotes (apt, dnf, pacman, brew, scoop, etc.), escolhida automaticamente pelo SO ou por uma flag. Isso ainda **não está implementado**.

## Limitações conhecidas

- Só aparecem pacotes que o winget consegue associar a uma fonte (winget ou msstore). Programas instalados por fora, que o `winget list` mostra com id `ARP\...`, não entram na lista.
- Se o nome de algum pacote não puder ser lido do `winget list`, o programa mostra só o id.
- Desinstalar runtimes e bibliotecas (VC++ Redistributable, .NET, etc.) pode quebrar outros programas. Use `i` para conferir antes de marcar.
- Algumas desinstalações abrem o instalador do próprio programa e podem pedir permissão de administrador.
- Há uma chamada extra ao `winget list` no início, então a primeira pergunta demora alguns segundos.
