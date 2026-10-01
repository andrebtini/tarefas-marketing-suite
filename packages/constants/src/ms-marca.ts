/**
 * Copyright (c) 2026-present Marketing Suite LTDA
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export const MS_NOME_PRODUTO = "Marketing Suite";
export const MS_URL_SITE = "https://marketingsuite.com.br";
export const MS_EMAIL_SUPORTE = "atendimento@marketingsuite.com.br";
export const MS_URL_GUIA_INTERNO = "";
export const MS_URL_REPOSITORIO = "https://github.com/andrebtini/tarefas-marketing-suite";
export const MS_COMMIT_SHA = process.env.VITE_MS_COMMIT_SHA || "";
export const MS_VERSAO = process.env.VITE_MS_VERSAO || "";
export const MS_DATA_VERSAO = process.env.VITE_MS_DATA_VERSAO || "";
export const MS_URL_CODIGO_FONTE = MS_COMMIT_SHA
  ? `${MS_URL_REPOSITORIO}/tree/${MS_COMMIT_SHA}`
  : MS_URL_REPOSITORIO;
export const MS_URL_LICENCA = `${MS_URL_REPOSITORIO}/blob/${MS_COMMIT_SHA || "main"}/LICENSE.txt`;
export const MS_URL_MODIFICACOES = `${MS_URL_REPOSITORIO}/blob/${MS_COMMIT_SHA || "main"}/marketing-suite/MODIFICACOES.md`;
export const MS_URL_AVISOS_TERCEIROS = `${MS_URL_REPOSITORIO}/blob/${MS_COMMIT_SHA || "main"}/marketing-suite/AVISOS-DE-TERCEIROS.md`;
export const MS_URL_APP = "https://tarefas.marketingsuite.online";
export const MS_URL_TERMOS = `${MS_URL_SITE}/termos`;
export const MS_URL_PRIVACIDADE = `${MS_URL_SITE}/privacidade`;
