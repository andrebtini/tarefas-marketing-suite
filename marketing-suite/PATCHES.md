Base: v1.4.2

| Arquivo | Tipo | Camada | Requisito | Motivo |
|---|---|---|---|---|
| `apps/api/plane/db/management/commands/test_email.py` | editado | backend | ARQ-002 | Usa o helper de texto puro (commit 297c3fdf2a, #9841). Reaplicar: nada se a base nova já tiver o commit 297c3fdf2a, senão git cherry-pick -x 297c3fdf2a. |
| `apps/api/plane/tests/unit/utils/test_email.py` | novo | backend | ARQ-002 | Teste do helper de texto puro (commit 297c3fdf2a, #9841). Reaplicar: nada se a base nova já tiver o commit 297c3fdf2a, senão git cherry-pick -x 297c3fdf2a. |
| `apps/api/plane/utils/email.py` | editado | licença | LIC-011 | Cabeçalho AGPL (commit 3e44777e6c, #9865) e decodificação de entidades (commit 297c3fdf2a, #9841). Reaplicar: ficar com o cabeçalho AGPL de 3 linhas e com o resto do arquivo da base nova. |
