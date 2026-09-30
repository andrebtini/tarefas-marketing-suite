#!/usr/bin/env bash
# Copyright (c) 2026-present Marketing Suite LTDA
# SPDX-License-Identifier: AGPL-3.0-only
# See the LICENSE file for details.

# Backup diario do Plane: banco (pg_dump) e anexos (volume do MinIO), em /opt/plane/backups.
# Chamado pelo /etc/cron.d/plane. Guarda DIAS dias. Para com erro se sobrar menos de 5 GB livres.
set -euo pipefail
cd /opt/plane

DIAS="${PLANE_BACKUP_DIAS:-14}"
DEST=/opt/plane/backups
HOJE=$(date +%F)
mkdir -p "$DEST"
chmod 700 "$DEST"

livre_gb=$(df -BG --output=avail "$DEST" | tail -1 | tr -dc '0-9')
if [ "$livre_gb" -lt 5 ]; then
  echo "$(date -Is) ERRO: so ${livre_gb} GB livres, backup nao feito" >&2
  exit 1
fi

# Banco: dump logico, consistente mesmo com o Plane no ar. Pelo socket local (-h): o container
# herda PGHOST=plane-db, e pela rede o Postgres pede a senha.
docker compose exec -T plane-db pg_dump -h /var/run/postgresql -U plane -d plane --no-owner \
  | gzip > "$DEST/.plane-$HOJE.sql.gz.parcial"
mv "$DEST/.plane-$HOJE.sql.gz.parcial" "$DEST/plane-$HOJE.sql.gz"

# Anexos: le o volume do MinIO por um container descartavel, sem parar nada.
vol=$(docker volume ls -q | grep -E '^plane_uploads$')
docker run --rm -v "$vol":/dados:ro alpine:3.20 tar -czf - -C /dados . \
  > "$DEST/.anexos-$HOJE.tgz.parcial"
mv "$DEST/.anexos-$HOJE.tgz.parcial" "$DEST/anexos-$HOJE.tgz"

find "$DEST" -maxdepth 1 -type f \( -name 'plane-*.sql.gz' -o -name 'anexos-*.tgz' \) -mtime +"$DIAS" -delete
echo "$(date -Is) OK: $(du -h "$DEST/plane-$HOJE.sql.gz" | cut -f1) de banco, $(du -h "$DEST/anexos-$HOJE.tgz" | cut -f1) de anexos"
