#!/usr/bin/env bash
# Copyright (c) 2026-present Marketing Suite LTDA
# SPDX-License-Identifier: AGPL-3.0-only
# See the LICENSE file for details.
#
# LIC-020: F1 a F6 (o F7, das imagens, roda à parte). Saída vazia e código 0 = limpo.
cd "$(git rev-parse --show-toplevel)" || exit 2
APROVADA='https://api.fontshare.com/v2/css?f[]=satoshi@1'
saida=$(
{
# F1: nome de arquivo rastreado
git ls-files | grep -i satoshi
# F2: nome de arquivo em qualquer ponto do histórico, em qualquer ramo ou tag
git log --all --format= --name-only | grep -i satoshi
# F3: arquivo de fonte rastreado que não está na lista permitida (e vice-versa)
diff <(git ls-files '*.ttf' '*.otf' '*.woff' '*.woff2' '*.eot' | sort) \
     <(tr -d '\r' < marketing-suite/fontes-permitidas.txt | grep -v -e '^#' -e '^$' | sort)
# F4: nome da família dentro do arquivo de fonte (a tabela name vem em UTF-16 ou comprimida, grep -a não acha)
python3 -c 'import fontTools, brotli' 2>/dev/null || echo "FALTA: pip install fonttools brotli"
git ls-files -z '*.ttf' '*.otf' '*.woff' '*.woff2' | xargs -0 -r python3 -c '
import sys
from fontTools.ttLib import TTFont
for p in sys.argv[1:]:
    try:
        nomes = " ".join(r.toUnicode(errors="replace") for r in TTFont(p, lazy=True)["name"].names)
    except Exception:
        print("NAO LIDO: " + p)
        continue
    if "satoshi" in nomes.lower():
        print(p)
'
# F5: @font-face declarando a família Satoshi
git grep -l -z -i satoshi -- '*.css' '*.ts' '*.tsx' '*.html' \
  | xargs -0 -r perl -0777 -ne 'print "$ARGV\n" if /\@font-face\s*\{[^}]*satoshi/is'
# F6a: arquivo local da Satoshi, url(...) da Satoshi ou outra URL do Fontshare que não seja a aprovada
git grep -n -i -E 'satoshi[-_a-z0-9]*\.(woff2?|ttf|otf|eot)|url\([^)]*satoshi|https?://[^[:space:]"<>)]*fontshare' -- . ':(exclude)*.md' ':(exclude)marketing-suite/scripts/checar-fontes.sh' \
  | grep -v -F "$APROVADA"
# F6b: @import da Satoshi (a exclusão do marca.css da Fase 0 cai com a BLD-013)
git grep -n -i -E '@import[^;]*(satoshi|fontshare)' -- . ':(exclude)*.md' ':(exclude)marketing-suite/scripts/checar-fontes.sh' ':(exclude)marketing-suite/fase0/marca.css'
} 2>&1
)
[ -z "$saida" ] && exit 0
printf '%s\n' "$saida"
exit 1
