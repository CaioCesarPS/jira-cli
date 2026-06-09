# Bug: ADF Conversion Failure em `jira issue create --description`

**Data:** 2026-06-05  
**Descoberto em:** criação da issue SED-510  
**Status:** Workaround documentado — bug na CLI, não no Jira

---

## Descrição do Problema

Ao tentar criar uma issue Jira com descrição rica (contendo blocos de código com triple backticks) usando o comando:

```bash
jira issue create \
  --summary "Título" \
  --description "$(cat /tmp/arquivo_com_code_blocks.txt)"
```

A CLI retorna o seguinte erro:

```
"description: The field value is not valid Atlassian Document Format (ADF) content."
```

A issue **não é criada**.

---

## Causa Raiz

A CLI `jira` converte Markdown para ADF (Atlassian Document Format) antes de enviar à API do Jira. O subcomando `create` usa um path de serialização ADF mais restrito que os outros subcomandos.

**Quando o conteúdo contém triple backticks (` ``` `)**, a serialização falha no `create` porque a interpolação shell `"$(cat /tmp/file.txt)"` passa o conteúdo bruto como argumento de linha de comando, e o parser ADF interno do `create` não consegue processar blocos de código nesse contexto.

**Por que `describe` e `comment` funcionam?**  
São subcomandos com implementações de serialização ADF distintas. O `describe` e o `comment` foram implementados com maior tolerância ao Markdown complexo — o mesmo arquivo que falha no `create` é aceito por eles sem erros.

---

## Reprodução

````bash
# Cria arquivo com code block
cat > /tmp/test.txt << 'EOF'
## Título

```typescript
const x = 1;
````

EOF

# Falha: erro ADF

jira issue create --summary "Test" --description "$(cat /tmp/test.txt)"

# → "description: The field value is not valid Atlassian Document Format (ADF) content."

# Funciona: cria sem descrição, depois atualiza

KEY=$(jira --json issue create --summary "Test" | jq -r '.data.issue_key')
jira issue describe $KEY --description "$(cat /tmp/test.txt)"

# → "Updated description for $KEY"

````

---

## Workaround — Two-Step Flow

**NUNCA** passar descrição rica diretamente no `jira issue create`. Usar sempre dois passos:

**Passo 1 — Criar com descrição de uma linha (sem code blocks):**

```bash
KEY=$(jira --json issue create \
  --summary "Título da Feature" \
  --project SED \
  --description "Ver descrição completa abaixo." \
  | jq -r '.data.issue_key')
echo "Created: $KEY"
````

**Passo 2 — Atualizar a descrição com conteúdo completo:**

````bash
cat > /tmp/jira_body.txt << 'EOF'
## Seção

Conteúdo completo com **bold**, listas e blocos de código:

```typescript
export class MyService {}
````

- item 1
- item 2
  EOF

jira issue describe $KEY --description "$(cat /tmp/jira_body.txt)"

````

**Passo 3 — Atribuir ao usuário atual:**

```bash
jira issue assign $KEY --assign-me
````

> **Atenção:** `jira issue assign` aceita apenas `--assign-me` ou `--assignee <accountId>`. Não aceita email como argumento posicional — isso também causa erro.

---

## O que ajustar na CLI

Para corrigir o bug na CLI, o subcomando `create` deveria usar o mesmo path de serialização ADF que `describe` e `comment`. O ponto de divergência está na forma como o parser ADF do `create` processa o argumento `--description` quando recebido via interpolação de subshell (`$(cat ...)`).

Possíveis correções:

1. Unificar o parser ADF entre os subcomandos `create`, `describe` e `comment`
2. No subcomando `create`, aceitar também um arquivo como input (`--description-file <path>`) para evitar a interpolação shell
3. Normalizar os backticks antes de passar para o parser ADF no `create`
