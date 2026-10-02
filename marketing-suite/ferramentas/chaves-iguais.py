# Copyright (c) 2026-present Marketing Suite LTDA
# SPDX-License-Identifier: AGPL-3.0-only
# See the LICENSE file for details.
"""Lista valores do pt-BR iguais ao ingles que ainda precisam de traducao (IDI-008)."""
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from catalogo import pares, sem_chaves

PERMITIDAS = "marketing-suite/ferramentas/chaves-iguais-permitidas.txt"
PALAVRA = re.compile(r"[A-Za-z]{3,}")
DOMINIO = re.compile(r"^[A-Za-z0-9.-]+\.[A-Za-z]{2,}$")


def excecoes():
    saida = set()
    for linha in open(PERMITIDAS, encoding="utf-8"):
        linha = linha.strip()
        if linha and not linha.startswith("#"):
            saida.add(linha)
    return saida


def placeholder(valor):
    texto = valor.strip()
    if texto.startswith(("http://", "https://", "api://")):
        return True
    if "@" in texto:
        return True
    return bool(DOMINIO.fullmatch(texto))


en, pt = pares("en"), pares("pt-BR")
livres = excecoes()
achou = False
for chave in sorted(set(en) & set(pt)):
    if en[chave] != pt[chave]:
        continue
    valor = pt[chave]
    if valor in livres or placeholder(valor):
        continue
    if not PALAVRA.search(sem_chaves(valor)):
        continue
    print(f"{chave}\t{valor}")
    achou = True
sys.exit(1 if achou else 0)
