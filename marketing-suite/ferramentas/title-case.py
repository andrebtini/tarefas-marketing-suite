# Copyright (c) 2026-present Marketing Suite LTDA
# SPDX-License-Identifier: AGPL-3.0-only
# See the LICENSE file for details.
"""Lista rotulos em Title Case no pt-BR (IDI-012)."""
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from catalogo import pares

PALAVRA = re.compile(r"[A-Za-z0-9]+(?:['’][A-Za-z0-9]+)?|[X]|ID")
pt = pares("pt-BR")
achou = False
for chave in sorted(pt):
    valor = pt[chave].strip()
    if len(valor) > 40 or valor.endswith("."):
        continue
    palavras = valor.split()
    if len(palavras) < 2:
        continue
    if all(re.match(r"^(?:[A-Z]|X$|ID$)", parte) or parte[:1].isupper() for parte in palavras):
        if all(parte[:1].isupper() or parte in {"X", "ID"} for parte in palavras):
            print(f"{chave}\t{pt[chave]}")
            achou = True
sys.exit(1 if achou else 0)
