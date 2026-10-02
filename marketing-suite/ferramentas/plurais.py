# Copyright (c) 2026-present Marketing Suite LTDA
# SPDX-License-Identifier: AGPL-3.0-only
# See the LICENSE file for details.
"""Lista plurais ICU que o pt-BR perdeu e o sufixo {plural} (IDI-010)."""
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from catalogo import pares

MARCA = re.compile(r"\{\s*\w+\s*,\s*plural")
en, pt = pares("en"), pares("pt-BR")
achou = False
for chave in sorted(set(en) & set(pt)):
    if MARCA.search(en[chave]) and not MARCA.search(pt[chave]):
        print(chave)
        achou = True
for chave in sorted(pt):
    if "{plural}" in pt[chave]:
        print(f"{chave}\t{{plural}}")
        achou = True
sys.exit(1 if achou else 0)
