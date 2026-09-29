# Plane da Marketing Suite

Gerenciador de tarefas do time em **https://tarefas.marketingsuite.online**, instalado em 29/09/2026.

Este repositório tem duas coisas:
- o código do Plane (cópia do `makeplane/plane`, ramo `preview`, 83 commits depois da v1.4.2), como
  base para personalizar depois;
- esta pasta, `marketing-suite/`, com a instalação que roda no VPS.

**O servidor não compila nada deste código.** Ele roda as imagens oficiais do Plane na tag fixa
`v1.4.2` (`APP_RELEASE` no `.env`). Personalizar exige gerar imagens próprias antes (GitHub Actions
está desligado neste repo; os workflows em `.github/` são os do Plane original).

## O que roda

VPS `srv1876522` (2 vCPU, 7,9 GB de RAM, divide com Postiz, Chatwoot e a Base), pasta `/opt/plane`
(root, 700):

| Arquivo | Para quê |
|---|---|
| `docker-compose.yml` | Os 12 serviços, com teto de memória em cada um |
| `.env` | Segredos (root, 600), gerado pelo `gerar-env.sh`; nunca sai do servidor |
| `.env.example` | Modelo do `.env` |
| `gerar-env.sh` | Cria o `.env` sorteando os segredos; recusa se ele já existe |
| `backup-plane.sh` | Backup do banco e dos anexos em `/opt/plane/backups`, 14 dias |
| `plane.cron` | Vira `/etc/cron.d/plane`: backup às 3h45 |

Caminho de uma requisição: internet, nginx do CloudPanel (site `tarefas`, HTTPS), `127.0.0.1:8098`,
proxy do Plane (Caddy, só HTTP), e dali para `web`, `space`, `admin`, `live`, `api` e o MinIO.

Diferenças para o compose oficial (`deployments/cli/community/docker-compose.yml`):
1. imagens na tag fixa, nunca `stable`;
2. proxy só em `127.0.0.1:8098`, sem certificado próprio;
3. teto de memória por container e `max_connections=100` no Postgres (o original pede 1000);
4. senhas do banco e da fila vindas do `.env` (o original deixa `plane:plane` nas URLs);
5. MinIO do fork `pgsty/minio`: o `minio/minio` saiu do Docker Hub e o quay.io passou a pedir
   login em setembro de 2026. O próprio Plane adotou o fork (makeplane/plane#9823).

O VPS ganhou um swap extra de 4 GB em `/swap-plane` (linha no `/etc/fstab`), ao lado dos 2 GB do
`dphys-swapfile`, porque o swap original estava 97% cheio antes do Plane.

## Instalação (o que foi feito)

```bash
mkdir -m 700 /opt/plane    # e copiar os arquivos desta pasta para lá
cd /opt/plane && ./gerar-env.sh
docker compose pull && docker compose up -d     # o "migrator" cria as tabelas e sai com 0
install -o root -g root -m 644 plane.cron /etc/cron.d/plane
clpctl site:add:reverse-proxy --domainName=tarefas.marketingsuite.online \
  --reverseProxyUrl=http://127.0.0.1:8098 --siteUser=tarefas --siteUserPassword=<sorteada e descartada>
clpctl lets-encrypt:install:certificate --domainName=tarefas.marketingsuite.online
```

No arquivo do nginx (`/etc/nginx/sites-enabled/tarefas.marketingsuite.online.conf`) entraram a linha
`X-Robots-Tag` (subdomínio interno, fora da busca) e uma trava temporária: `/god-mode` e
`/api/instances/admins/` só abrem de um IP. O arquivo antes da mudança está em
`/root/tarefas.marketingsuite.online.conf.antes-plane-2026-09-29`.

## Primeiro acesso

1. Em `https://tarefas.marketingsuite.online/god-mode/`, criar o administrador da instância (só abre
   do IP liberado na trava). Quem chega primeiro nessa tela vira dono do Plane.
2. Tirar os dois blocos `location` da trava no nginx, `nginx -t` e `systemctl reload nginx`.
3. No god-mode: desligar o cadastro aberto e configurar o e-mail (SMTP do Google Workspace com uma
   senha de app nova, digitada pela própria tela).
4. Criar o workspace e convidar o time.

## Operação

- Estado: `cd /opt/plane && docker compose ps` e `docker stats --no-stream | grep plane-`.
- `docker compose restart` não relê o `.env`: depois de mudar o `.env`, use `docker compose up -d`.
- **Atualizar:** ler as notas da versão nova, rodar o backup, trocar `APP_RELEASE` no `.env`,
  `docker compose pull && docker compose up -d`. O `migrator` roda sozinho.
- **Restaurar o banco:**
  `docker compose stop api worker beat-worker live`, depois
  `zcat backups/plane-AAAA-MM-DD.sql.gz | docker compose exec -T plane-db psql -h /var/run/postgresql -U plane -d plane`
  num banco vazio, e `docker compose up -d`.
- **Restaurar anexos:** extrair `backups/anexos-AAAA-MM-DD.tgz` no volume `plane_uploads` com um
  container descartável, com o MinIO parado.
- Os backups ficam no mesmo disco do servidor: não protegem contra perda do VPS.
