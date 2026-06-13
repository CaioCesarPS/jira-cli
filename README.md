# jira-cli

CLI em Go para interagir com o Jira Cloud via REST API v3.

## Pré-requisitos

- [Go 1.21+](https://go.dev/dl/) instalado e configurado no `PATH`

## Instalação

### Instalação automática (recomendada)

O script detecta seu sistema operacional e arquitetura, baixa a última release do GitHub e coloca o binário no `PATH`.

**Linux / macOS:**

```bash
curl -sSL https://raw.githubusercontent.com/caiocesarps/jira-cli/main/install.sh | bash
```

**Windows (PowerShell, executar como Administrador):**

```powershell
powershell -c "irm https://raw.githubusercontent.com/caiocesarps/jira-cli/main/install.ps1 | iex"
```

> Após instalar no Windows, feche e reabra o terminal para que o `PATH` seja atualizado.

### Via GitHub Releases (manual)

Baixe o binário pré-compilado para a sua plataforma na página de [Releases](https://github.com/caiocesarps/jira-cli/releases) e coloque-o em um diretório no seu `PATH`.

```bash
# Exemplo: Linux amd64
VERSION=$(curl -s https://api.github.com/repos/caiocesarps/jira-cli/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L -o jira "https://github.com/caiocesarps/jira-cli/releases/download/${VERSION}/jira-cli-${VERSION}-linux-amd64"
chmod +x jira
sudo mv jira /usr/local/bin/
```

### Build local

**macOS / Linux:**

```bash
# Build local
make build
./bin/jira --help

# Instala no $GOPATH/bin (disponível globalmente)
make install
```

**Windows:**

O `make` não é nativo no Windows. Use o comando Go diretamente:

```powershell
# Build local
go build -o bin\jira.exe .\cmd\jira
.\bin\jira.exe --help

# Instala no %USERPROFILE%\go\bin (disponível globalmente)
go install .\cmd\jira
```

> Certifique-se de que `%USERPROFILE%\go\bin` está no seu `PATH`.

## Setup

```bash
jira config init
# ou para um perfil específico:
jira config init --profile client-x
```

O arquivo de configuração fica em `~/.jira-cli/config.yaml`.

### Exemplo de config.yaml

```yaml
default_profile: work

profiles:
  work:
    base_url: https://work.atlassian.net
    email: you@work.com
    api_token: SEU_TOKEN_AQUI
    default_project_key: WORK
    default_issue_type: Task
  client-x:
    base_url: https://clientx.atlassian.net
    email: you@clientx.com
    api_token: OUTRO_TOKEN
    default_project_key: CX
    default_issue_type: Story
```

Gere seu API token em: https://id.atlassian.com/manage-profile/security/api-tokens

## Uso

### Trocar de perfil

```bash
# Via flag (sessão única)
jira --profile client-x issue create --summary "Bug"

# Via variável de ambiente (sessão do shell)
export JIRA_PROFILE=client-x
jira issue create --summary "Bug"
```

### Criar uma issue

```bash
jira issue create --summary "Título da task"
jira issue create --summary "Bug no login" --description "Detalhes..." --project PROJ --type Bug
```

### Atualizar descrição

```bash
jira issue describe PROJ-123 --description "Nova descrição"
```

### Mudar status

```bash
jira issue transition PROJ-123 --status "In Progress"
```

### Adicionar comentário

```bash
jira issue comment PROJ-123 --body "Comentário via CLI"
```

### Saída JSON

Adicione `--json` a qualquer comando para saída estruturada:

```bash
jira --json issue create --summary "Bug" | jq .data.issue_key
```

### Listar perfis

```bash
jira config list
jira --json config list
```

## Variáveis de ambiente

| Variável         | Descrição                                          |
|------------------|----------------------------------------------------|
| `JIRA_PROFILE`   | Perfil ativo (sobrescreve `default_profile`)       |
| `JIRA_BASE_URL`  | URL base do Jira (sobrescreve o perfil)            |
| `JIRA_EMAIL`     | Email Atlassian (sobrescreve o perfil)             |
| `JIRA_API_TOKEN` | API token (sobrescreve o perfil)                   |
| `JIRA_PROJECT`   | Project key padrão (sobrescreve o perfil)          |

## Exit codes

| Código | Significado                       |
|--------|-----------------------------------|
| 0      | Sucesso                           |
| 1      | Erro geral / API error            |
| 2      | Input inválido                    |
| 3      | Autenticação falhou (401/403)     |
| 4      | Recurso não encontrado (404)      |

## Build para distribuição

```bash
make build-all
# gera binários em bin/ para Darwin (arm64/amd64), Linux (arm64/amd64) e Windows
```

## Versionamento e changelog

O projeto segue [Semantic Versioning](https://semver.org/lang/pt-BR/) e [Conventional Commits](https://www.conventionalcommits.org/pt-br/v1.0.0/).

Para publicar uma nova versão:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

O workflow `Release` do GitHub Actions compila os binários para Linux, macOS e Windows, gera o changelog automaticamente a partir dos commits e publica tudo na página de [Releases](https://github.com/caiocesarps/jira-cli/releases).

Veja também o arquivo [CHANGELOG.md](./CHANGELOG.md).
