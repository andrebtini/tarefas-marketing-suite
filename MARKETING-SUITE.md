# Marketing Suite Tarefas

Este repositório é o fork do [Vikunja](https://vikunja.io) que a Marketing Suite usa como
gerenciador de tarefas interno do time. O código é o do Vikunja, com a aparência, os textos e
os e-mails da marca Marketing Suite. A licença continua a do Vikunja, **AGPL-3.0-or-later**
(arquivo `LICENSE`), e o código modificado fica aqui, aberto, como a licença pede.

## Ramos e tags

| O quê | Para quê |
|---|---|
| `marketing-suite` | O ramo do fork. Nasceu da tag `v2.6.0` do Vikunja e recebe todas as mudanças da casa |
| `main` | Cópia do `main` do Vikunja, sem uso aqui. Não receba commit nele |
| `v2.6.0-ms.N` | Cada versão que vai para a produção: a versão do Vikunja de base, mais `-ms.` e um número que só cresce |

Os workflows do Vikunja que não fazem sentido num fork (publicação oficial, testes de PR,
tradução, robôs de issue) foram tirados do `marketing-suite`. Fica o `test.yml`, que só roda
quando outro workflow o chama, e entra o `imagem-ms.yml`.

## Como a imagem nasce

1. Commit no `marketing-suite` (ou num ramo que entra nele).
2. Tag no formato `v2.6.0-ms.N` e push da tag:

   ```bash
   git tag v2.6.0-ms.1
   git push origin v2.6.0-ms.1
   ```

3. O workflow `imagem-ms.yml` roda o `Dockerfile` da raiz, sem mudança, só para `linux/amd64`, e
   publica `ghcr.io/andrebtini/tarefas-marketing-suite:v2.6.0-ms.1`. Leva de 10 a 20 minutos.
   Também dá para disparar à mão, pela aba Actions, informando a tag.

Nenhuma imagem é construída no servidor. O front vai embutido no binário do Go
(`frontend/embed.go`), então qualquer mudança de tela, texto ou e-mail exige uma imagem nova.

## Como atualizar a partir do Vikunja

```bash
git fetch upstream --tags
git rebase --onto v2.7.0 v2.6.0 marketing-suite
```

Resolva os conflitos, que devem ser poucos se as mudanças da casa ficarem nos arquivos de
marca. Rode os testes, publique com `git push --force-with-lease origin marketing-suite`, e crie
a tag `v2.7.0-ms.1`. Leia antes o changelog do Vikunja: versão maior tem nota de migração
própria.

## Como a produção troca de imagem, e como volta

A produção roda esta imagem por Docker Compose, com uma pasta de dados e um usuário próprios. O
passo a passo completo mora no repositório interno da Marketing Suite. O resumo:

1. Backup do banco e dos anexos, guardado com o nome da versão de onde se sai.
2. No `docker-compose.yaml`, troca **só a linha `image:`** para a tag nova. O resto (usuário do
   container, volumes, limites) fica como está.
3. `docker compose pull` e `docker compose up -d`. As migrações do banco rodam na subida.

Para voltar: a tag anterior na linha `image:` **e** a restauração do backup. Voltar só a tag não
basta, porque a versão nova pode ter migrado o banco.

Nada sobe para a produção sem o "pode subir" do responsável.

## O que mudou em relação ao Vikunja

| Versão | Mudança |
|---|---|
| `v2.6.0-ms.0` | Nenhuma no produto: prova da esteira. Tira os workflows do Vikunja e acrescenta o `imagem-ms.yml` e este arquivo |
