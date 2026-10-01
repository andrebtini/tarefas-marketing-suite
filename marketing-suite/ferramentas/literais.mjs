// Copyright (c) 2026-present Marketing Suite LTDA
// SPDX-License-Identifier: AGPL-3.0-only
// See the LICENSE file for details.
//
// TXT-030: literais visiveis. Saida vazia e codigo 0 = tudo na allowlist.
// Codigo 1: achado fora da allowlist. Codigo 2: motivo invalido na allowlist.
import fs from "fs";
import path from "path";
import { parse } from "@babel/parser";

const DIRS = [
  "apps/web",
  "apps/admin",
  "apps/space",
  "packages/editor",
  "packages/propel/src",
  "packages/ui/src",
  "packages/constants/src",
  "packages/utils/src",
];
const SKIP_DIR = new Set(["node_modules", "dist", "build", ".next", ".turbo", ".react-router"]);
const SKIP_FILE = new Set([
  "packages/utils/src/auth.ts",
  "packages/propel/src/icons/constants.tsx",
]);
const ATTRS = new Set([
  "placeholder", "title", "label", "aria-label", "alt", "tooltipContent", "tooltipHeading",
  "description", "content", "header", "subHeader", "heading", "text", "message",
  "buttonText", "primaryButtonText", "secondaryButtonText",
]);
const PROPS = new Set([
  "title", "message", "label", "description", "placeholder", "text", "heading",
  "header", "content", "tooltip", "subHeader", "shortTitle",
]);
const SKIP_DECL = {
  "packages/constants/src/calendar.ts": new Set(["MONTHS_LIST", "DAYS_LIST", "CALENDAR_LAYOUTS"]),
  "packages/constants/src/profile.ts": new Set(["START_OF_THE_WEEK_OPTIONS"]),
  "apps/web/core/components/gantt-chart/data/index.ts": new Set(["weeks", "months", "quarters"]),
};
const MOTIVOS = new Set(["base-v1.4.2", "venda-secao-8", "nome-proprio", "god-mode-direto"]);
const ALLOW = "marketing-suite/ferramentas/literais-permitidos.txt";

function walk(dir, out) {
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    if (SKIP_DIR.has(e.name)) continue;
    const p = path.join(dir, e.name);
    if (e.isDirectory()) walk(p, out);
    else if (/\.(ts|tsx)$/.test(e.name) && !/\.(test|spec|stories)\./.test(e.name) && !e.name.endsWith(".d.ts")) {
      out.push(p);
    }
  }
}

function normTemplate(node) {
  let s = "";
  node.quasis.forEach((q, i) => {
    s += q.value.cooked ?? q.value.raw;
    if (i < node.expressions.length) s += "{}";
  });
  return s;
}

function umaLinha(s) {
  return s.replace(/\s+/g, " ").trim();
}

function isText(s) {
  if (typeof s !== "string") return false;
  if (!/\p{L}/u.test(s)) return false;
  const t = s.trimStart();
  if (/^(https?:|\/|\.\/|\.\.\/|mailto:|#|@)/.test(t)) return false;
  if (!/\s/.test(s)) {
    if (!/^\p{Lu}/u.test(s)) return false;
    const letters = s.match(/\p{L}/gu) || [];
    if (letters.length < 2) return false;
    if (/^[A-Z0-9_]+$/.test(s)) return false;
    if (/[/._:]/.test(s)) return false;
    if (/\p{Ll}\p{Lu}/u.test(s)) return false;
  }
  return true;
}

function textOf(node) {
  if (!node) return null;
  if (node.type === "StringLiteral") return node.value;
  if (node.type === "TemplateLiteral") return normTemplate(node);
  return null;
}

function calleeName(node) {
  if (!node) return "";
  if (node.type === "Identifier") return node.name;
  if (node.type === "MemberExpression" && !node.computed && node.property.type === "Identifier") return node.property.name;
  return "";
}

function achar() {
  const files = [];
  for (const d of DIRS) {
    const abs = path.join(d);
    if (fs.existsSync(abs)) walk(abs, files);
  }
  const achados = [];
  for (const file of files) {
    const rel = file.replaceAll("\\", "/");
    if (SKIP_FILE.has(rel)) continue;
    if (/^packages\/propel\/src\/empty-state\/assets\/[^/]+\/constant\.tsx$/.test(rel)) continue;
    let ast;
    try {
      ast = parse(fs.readFileSync(file, "utf8"), {
        sourceType: "module",
        plugins: ["typescript", "jsx"],
        errorRecovery: true,
      });
    } catch {
      continue;
    }
    const skipNames = SKIP_DECL[rel] || null;
    function walkNode(node, ctx) {
      if (!node || typeof node.type !== "string") return;
      if (node.type === "ImportDeclaration") return;
      if (node.type.startsWith("TS") && node.type !== "TSAsExpression" && node.type !== "TSNonNullExpression" && node.type !== "TSSatisfiesExpression" && node.type !== "TSTypeAssertion") {
        return;
      }
      let next = ctx;
      if (node.type === "VariableDeclarator" && skipNames && node.id && node.id.type === "Identifier" && skipNames.has(node.id.name)) {
        next = { ...ctx, skipProps: true };
      }
      if (node.type === "CallExpression" && (calleeName(node.callee) === "t" || calleeName(node.callee) === "translate")) {
        const args = node.arguments || [];
        if (args[0]) walkNode(args[0], { ...ctx, skipKey: true });
        for (let i = 1; i < args.length; i++) walkNode(args[i], ctx);
        return;
      }
      if (!ctx.skipKey) {
        if (node.type === "JSXText") {
          const s = node.value;
          if (isText(s)) achados.push({ rel, linha: node.loc?.start?.line || 0, texto: umaLinha(s), tipo: "jsx" });
        }
        if (node.type === "JSXExpressionContainer" && node.parentHint === "child") {
          const s = textOf(node.expression);
          if (s != null && isText(s)) achados.push({ rel, linha: node.loc?.start?.line || 0, texto: umaLinha(s), tipo: "jsx-expr" });
        }
        if (node.type === "JSXAttribute" && node.name && node.name.type === "JSXIdentifier") {
          const nome = node.name.name;
          if (ATTRS.has(nome) && node.value) {
            let alvo = node.value;
            if (alvo.type === "JSXExpressionContainer") alvo = alvo.expression;
            const s = textOf(alvo);
            if (s != null && isText(s)) achados.push({ rel, linha: node.loc?.start?.line || 0, texto: umaLinha(s), tipo: "attr:" + nome });
          }
        }
        if (!ctx.skipProps && node.type === "ObjectProperty" && !node.computed) {
          let nome = "";
          if (node.key.type === "Identifier") nome = node.key.name;
          else if (node.key.type === "StringLiteral") nome = node.key.value;
          if (PROPS.has(nome)) {
            const irmaos = ctx.siblingKeys || new Set();
            let temI18n = false;
            for (const k of irmaos) if (String(k).startsWith("i18n_")) temI18n = true;
            const s = textOf(node.value);
            if (!temI18n && s != null && isText(s)) achados.push({ rel, linha: node.loc?.start?.line || 0, texto: umaLinha(s), tipo: "prop:" + nome });
          }
        }
      }
      if (node.type === "ObjectExpression") {
        const keys = new Set();
        for (const p of node.properties) {
          if (p.type !== "ObjectProperty" || p.computed) continue;
          if (p.key.type === "Identifier") keys.add(p.key.name);
          else if (p.key.type === "StringLiteral") keys.add(p.key.value);
        }
        for (const p of node.properties) walkNode(p, { ...next, siblingKeys: keys });
        return;
      }
      if (node.type === "JSXElement" || node.type === "JSXFragment") {
        for (const c of node.children || []) {
          if (c.type === "JSXExpressionContainer") c.parentHint = "child";
          walkNode(c, next);
        }
        if (node.type === "JSXElement" && node.openingElement) walkNode(node.openingElement, next);
        return;
      }
      for (const k of Object.keys(node)) {
        if (["loc", "start", "end", "extra", "leadingComments", "trailingComments", "innerComments"].includes(k)) continue;
        const v = node[k];
        if (Array.isArray(v)) v.forEach((c) => walkNode(c, next));
        else if (v && typeof v.type === "string") walkNode(v, next);
      }
    }
    walkNode(ast.program, { skipProps: false, skipKey: false, siblingKeys: new Set() });
  }
  return achados;
}

const achados = achar();
if (process.argv.includes("--gerar-base")) {
  const vistos = new Set();
  const linhas = [];
  for (const a of achados) {
    const chave = `${a.rel}\t${a.texto}`;
    if (vistos.has(chave)) continue;
    vistos.add(chave);
    linhas.push(`${a.rel}\t${a.texto}\tbase-v1.4.2`);
  }
  linhas.sort();
  process.stdout.write(linhas.join("\n") + (linhas.length ? "\n" : ""));
  process.exit(0);
}

const permitidos = new Set();
const invalidos = [];
const linhasAllow = fs.existsSync(ALLOW) ? fs.readFileSync(ALLOW, "utf8").split(/\r?\n/) : [];
for (const linha of linhasAllow) {
  if (!linha || linha.startsWith("#")) continue;
  const partes = linha.split("\t");
  if (partes.length < 3 || !MOTIVOS.has(partes[2])) {
    invalidos.push(linha);
    continue;
  }
  permitidos.add(`${partes[0]}\t${partes[1]}`);
}
if (invalidos.length) {
  for (const linha of invalidos) console.log(`MOTIVO INVALIDO\t${linha}`);
  process.exit(2);
}
const vistos = new Set();
for (const a of achados) vistos.add(`${a.rel}\t${a.texto}`);
for (const chave of permitidos) {
  if (!vistos.has(chave)) console.error(`AVISO: allowlist sem achado: ${chave.split("\t")[0]}`);
}
let fora = 0;
for (const a of achados) {
  if (permitidos.has(`${a.rel}\t${a.texto}`)) continue;
  fora += 1;
  console.log(`${a.rel}:${a.linha}\t${a.texto}\t${a.tipo}`);
}
process.exit(fora ? 1 : 0);
