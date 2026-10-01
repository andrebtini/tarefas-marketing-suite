# Copyright (c) 2026-present Marketing Suite LTDA
# SPDX-License-Identifier: AGPL-3.0-only
# See the LICENSE file for details.
"""Compara as chaves achatadas dos catalogos en e pt-BR. Rodar da raiz do clone."""
import json
import pathlib
import sys

RAIZ = pathlib.Path("packages/i18n/src/locales")


def achatar(valor, prefixo=""):
    if isinstance(valor, dict):
        for chave, filho in valor.items():
            yield from achatar(filho, f"{prefixo}.{chave}" if prefixo else chave)
    else:
        yield prefixo


def chaves(idioma):
    achadas = set()
    for arquivo in sorted((RAIZ / idioma).glob("*.json")):
        dados = json.loads(arquivo.read_text(encoding="utf-8"))
        achadas.update(f"{arquivo.name}:{c}" for c in achatar(dados))
    return achadas


en, pt = chaves("en"), chaves("pt-BR")
for c in sorted(en - pt):
    print(f"FALTA no pt-BR: {c}")
for c in sorted(pt - en):
    print(f"SOBRA no pt-BR: {c}")
sys.exit(1 if en != pt else 0)
