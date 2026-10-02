# Copyright (c) 2026-present Marketing Suite LTDA
# SPDX-License-Identifier: AGPL-3.0-only
# See the LICENSE file for details.
"""Leitura achatada dos catalogos en e pt-BR. Rodar da raiz do clone."""
import json
import pathlib
import re

RAIZ = pathlib.Path("packages/i18n/src/locales")
CHAVE = re.compile(r"\{[^{}]+\}")


def achatar(valor, prefixo=""):
    if isinstance(valor, dict):
        for chave, filho in valor.items():
            caminho = f"{prefixo}.{chave}" if prefixo else chave
            yield from achatar(filho, caminho)
    else:
        yield prefixo, valor


def pares(idioma):
    saida = {}
    for arquivo in sorted((RAIZ / idioma).glob("*.json")):
        dados = json.loads(arquivo.read_text(encoding="utf-8"))
        namespace = arquivo.stem
        for caminho, valor in achatar(dados):
            if isinstance(valor, str):
                saida[f"{namespace}:{caminho}"] = valor
    return saida


def sem_chaves(texto):
    return CHAVE.sub(" ", texto)
