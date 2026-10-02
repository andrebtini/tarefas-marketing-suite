# Copyright (c) 2026-present Marketing Suite LTDA
# SPDX-License-Identifier: AGPL-3.0-only
# See the LICENSE file for details.
"""Varre o pt-BR contra o glossario fechado (IDI-019)."""
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from catalogo import pares, sem_chaves

EXCECOES = "marketing-suite/ferramentas/glossario-excecoes.txt"

REGRAS = [
    ("admissao", re.compile(r"admiss[ãa]o|admiss[õo]es", re.I)),
    ("deletar", re.compile(r"delet[a-zà-ú]*", re.I)),
    ("nota_adesiva", re.compile(r"notas? adesivas?", re.I)),
    ("cronograma", re.compile(r"cronograma", re.I)),
    ("email", re.compile(r"(?<!-)email", re.I)),
    ("data_entrega", re.compile(r"data alvo|datas? de vencimento", re.I)),
    ("status", re.compile(r"status", re.I)),
    ("visualizacao", re.compile(r"visualiza[çc][õo]es|visualiza[çc][ãa]o", re.I)),
    ("rotulo", re.compile(r"r[óo]tulos?", re.I)),
    ("duplicado", re.compile(r"duplicado de", re.I)),
    ("bloqueio", re.compile(r"bloqueando|bloqueado por", re.I)),
    ("em_andamento", re.compile(r"Em Andamento")),
    ("usuario", re.compile(r"usu[áa]rios?", re.I)),
    ("toast_exclamacao", re.compile(r"^(Sucesso!|Erro!)$")),
    ("por_favor", re.compile(r"por favor", re.I)),
    ("com_sucesso", re.compile(r"com sucesso", re.I)),
    ("issue", re.compile(r"(?<![\w{])issues?(?![\w}])", re.I)),
    ("workspace", re.compile(r"(?<![\w{])workspaces?(?![\w}])", re.I)),
    ("subtarefa", re.compile(r"sub-tarefa", re.I)),
]

LAYOUTS = {
    "work-item:issue.layouts.list": "Lista",
    "work-item:issue.layouts.kanban": "Quadro",
    "work-item:issue.layouts.calendar": "Calendário",
    "work-item:issue.layouts.spreadsheet": "Planilha",
    "work-item:issue.layouts.gantt": "Linha do tempo",
    "work-item:issue.layouts.title.list": "Layout de lista",
    "work-item:issue.layouts.title.kanban": "Layout de quadro",
    "work-item:issue.layouts.title.calendar": "Layout de calendário",
    "work-item:issue.layouts.title.spreadsheet": "Layout de planilha",
    "work-item:issue.layouts.title.gantt": "Layout de linha do tempo",
}


def excecoes():
    saida = set()
    for linha in open(EXCECOES, encoding="utf-8"):
        linha = linha.strip()
        if not linha or linha.startswith("#"):
            continue
        chave, regra, _motivo = linha.split("\t")
        saida.add((chave, regra))
    return saida


def outro_idioma():
    from catalogo import pares as ler

    en, pl, pt = ler("en"), ler("pl"), ler("pt-BR")
    achados = []
    for chave, valor in pt.items():
        if not chave.startswith("template:"):
            continue
        if "podpory" in valor.lower():
            achados.append((chave, "outro_idioma", valor))
            continue
        if chave in pl and valor == pl[chave] and valor != en.get(chave) and len(valor.split()) >= 2:
            if valor.strip() == "URL JWKS":
                continue
            achados.append((chave, "outro_idioma", valor))
    return achados


pt = pares("pt-BR")
livres = excecoes()
linhas = []
for chave in sorted(pt):
    valor = pt[chave]
    for nome, padrao in REGRAS:
        if (chave, nome) in livres:
            continue
        alvo = valor if nome == "toast_exclamacao" else sem_chaves(valor)
        if padrao.search(alvo):
            linhas.append(f"{chave}\t{nome}\t{valor}")
    if chave in LAYOUTS and valor != LAYOUTS[chave]:
        linhas.append(f"{chave}\tlayout\t{valor}")
linhas.extend(f"{c}\t{r}\t{v}" for c, r, v in outro_idioma())
for linha in linhas:
    print(linha)
sys.exit(1 if linhas else 0)
