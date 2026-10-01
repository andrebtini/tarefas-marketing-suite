Base: v1.4.2

| Arquivo | Tipo | Camada | Requisito | Motivo |
|---|---|---|---|---|
| `apps/api/plane/db/management/commands/test_email.py` | editado | backend | ARQ-002 | Usa o helper de texto puro (commit 297c3fdf2a, #9841). Reaplicar: nada se a base nova já tiver o commit 297c3fdf2a, senão git cherry-pick -x 297c3fdf2a. |
| `apps/api/plane/tests/unit/utils/test_email.py` | novo | backend | ARQ-002 | Teste do helper de texto puro (commit 297c3fdf2a, #9841). Reaplicar: nada se a base nova já tiver o commit 297c3fdf2a, senão git cherry-pick -x 297c3fdf2a. |
| `apps/api/plane/utils/email.py` | editado | licença | LIC-011 | Cabeçalho AGPL (commit 3e44777e6c, #9865) e decodificação de entidades (commit 297c3fdf2a, #9841). Reaplicar: ficar com o cabeçalho AGPL de 3 linhas e com o resto do arquivo da base nova. |
| `.github/workflows/build-branch.yml` | removido | build | BLD-003 | Push em preview e canary publica imagem no Docker Hub. Reaplicar: apagar de novo se a base nova trouxer o arquivo. |
| `.github/workflows/feature-deployment.yml` | removido | build | BLD-003 | Dispatch de imagem AIO de feature. Reaplicar: apagar de novo se a base nova trouxer o arquivo. |
| `.github/workflows/codeql.yml` | removido | build | BLD-003 | CodeQL nos ramos preview, canary e master. Reaplicar: apagar de novo se a base nova trouxer o arquivo. |
| `.github/workflows/react-doctor.yml` | removido | build | BLD-003 | Lint do upstream em todo pull request. Reaplicar: apagar de novo se a base nova trouxer o arquivo. |
| `.github/workflows/i18n-sync-check.yml` | removido | build | BLD-003 | Exige os 19 idiomas. A checagem nossa é o job chaves. Reaplicar: apagar de novo se a base nova trouxer o arquivo. |
| `.github/workflows/copyright-check.yml` | removido | build | BLD-003 | SPDX do upstream. A checagem nossa é o job spdx. Reaplicar: apagar de novo se a base nova trouxer o arquivo. |
| `.github/workflows/check-version.yml` | removido | build | BLD-003 | Versão do upstream em pull request para master. Reaplicar: apagar de novo se a base nova trouxer o arquivo. |
| `.github/workflows/pull-request-build-lint-api.yml` | removido | build | BLD-003 | Lint da API do upstream. Reaplicar: apagar de novo se a base nova trouxer o arquivo. |
| `.github/workflows/pull-request-build-lint-web-apps.yml` | removido | build | BLD-003 | Lint do web do upstream. Reaplicar: apagar de novo se a base nova trouxer o arquivo. |
| `.github/workflows/ms-checks.yml` | novo | build | BLD-005 | Seis checagens em todo push e pull request. |
| `.github/workflows/ms-imagens.yml` | novo | build | BLD-004 | Publica as quatro imagens a partir de uma tag ms. |
| `.gitignore` | editado | licença | LIC-020 | Exceção para marketing-suite/scripts e para o package-lock das ferramentas. |
| `marketing-suite/ferramentas/chaves-paridade.py` | novo | build | BLD-005 | Compara as chaves achatadas de en e pt-BR. |
| `marketing-suite/ferramentas/literais.mjs` | novo | build | TXT-030 | Scanner de texto visível. A allowlist nasce com a base v1.4.2. |
| `marketing-suite/scripts/checar-fontes.sh` | novo | licença | LIC-020 | F1 a F6. O F7 roda na imagem. |
| `packages/tailwind-config/ms-tema.css` | novo | tema | ARQ-009 | Tokens da secao 4. Nao editar variables.css. |
| `packages/tailwind-config/index.css` | editado | tema | ARQ-009 | Importa ms-tema.css depois de animations.css. |
| `packages/constants/src/ms-marca.ts` | novo | marca | ARQ-011 | Constantes MS_ do produto. |
| `packages/constants/src/index.ts` | editado | marca | ARQ-011 | Exporta ms-marca. |
| `packages/constants/src/endpoints.ts` | editado | marca | ARQ-011 | WEBSITE_URL e SUPPORT_EMAIL deixam de usar plane.so. |
| `apps/web/app/root.tsx` | editado | tema | VIS-011 | Link da Satoshi antes do CSS global. |
| `apps/admin/app/root.tsx` | editado | tema | VIS-011 | Link da Satoshi antes do CSS global. |
| `apps/space/app/root.tsx` | editado | tema | VIS-011 | Link da Satoshi antes do CSS global. |
| `marketing-suite/fase0/marca.css` | editado | tema | F0-018 | Tokens saem deste arquivo e ficam em ms-tema.css. |
| `apps/api/plane/utils/ms_marca.py` | novo | backend | ARQ-014 | Constantes da marca e MS_IDIOMA para o perfil novo. |
| `packages/i18n/src/constants/language.ts` | editado | idioma | IDI-001 | DEFAULT_LANGUAGE pt-BR. FALLBACK continua en. |
| `packages/i18n/src/index.ts` | editado | idioma | IDI-001 | Exporta DEFAULT_LANGUAGE. |
| `packages/i18n/src/core/instance.ts` | editado | idioma | IDI-002 | Idioma inicial e localStorage ficam em pt-BR. |
| `apps/web/core/store/root.store.ts` | editado | idioma | IDI-003 | Sair nao volta o idioma para en. |
| `apps/web/core/store/user/profile.store.ts` | editado | idioma | IDI-004 | Perfil aplica e envia pt-BR. |
| `apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx` | editado | idioma | IDI-007 | Seletor de idioma sai da tela. Fuso e semana ficam. |
| `apps/web/core/components/power-k/config/preferences-commands.ts` | editado | idioma | IDI-007 | Comando de idioma fica invisivel. |
