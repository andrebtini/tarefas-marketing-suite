#!/usr/bin/env bash
# Copyright (c) 2026-present Marketing Suite LTDA
# SPDX-License-Identifier: AGPL-3.0-only
# See the LICENSE file for details.

# Cria /opt/plane/.env a partir do .env.example, sorteando cada valor "GERAR".
# Roda no servidor, como root. Recusa se o .env ja existe: trocar SECRET_KEY ou senha do banco
# depois da instalacao quebra sessoes e o acesso ao Postgres ja criado.
set -euo pipefail
cd "$(dirname "$0")"

if [ -e .env ]; then
  echo "ERRO: .env ja existe. Nada foi alterado." >&2
  exit 1
fi

umask 077
tmp=$(mktemp .env.XXXXXX)
while IFS= read -r linha || [ -n "$linha" ]; do
  if [[ "$linha" =~ ^([A-Z_]+)=GERAR$ ]]; then
    # So letras e numeros: vai dentro de URL (postgresql://, amqp://) sem precisar escapar.
    valor=$(openssl rand -hex 32)
    printf '%s=%s\n' "${BASH_REMATCH[1]}" "$valor"
  else
    printf '%s\n' "$linha"
  fi
done < .env.example > "$tmp"

chown root:root "$tmp"
chmod 600 "$tmp"
mv "$tmp" .env
echo "OK: .env criado com $(grep -c '' .env) linhas; os segredos nao foram mostrados."
