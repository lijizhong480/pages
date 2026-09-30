import { readFile, writeFile, mkdir, readdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "../..");
const source = path.join(root, "创业");
const output = path.join(here, "../data/imports");

await mkdir(output, { recursive: true });

const css = await readFile(path.join(source, "assets/style.css"), "utf8");
const renderer = await readFile(path.join(source, "assets/render.js"), "utf8");
const directories = (await readdir(source, { withFileTypes: true }))
  .filter((entry) => entry.isDirectory() && /^\d{2}-/.test(entry.name))
  .map((entry) => entry.name)
  .sort();

for (const directory of directories) {
  const plan = await readFile(path.join(source, directory, "plan.js"), "utf8");
  const html = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <style>${css}</style>
</head>
<body>
  <header class="topbar"><div class="shell"><span class="brand">创业项目商业计划书</span><span class="nav">可打印版本</span></div></header>
  <section class="hero" id="hero"></section>
  <main class="shell"><div class="layout"><nav class="toc" id="toc"></nav><article class="content" id="content"></article></div></main>
  <footer class="footer"><div class="shell">版本日期：<span id="updated"></span> · 财务数字需用真实订单校准。</div></footer>
  <script>${plan}</script>
  <script>${renderer}</script>
</body>
</html>`;
  const file = path.join(output, `${directory}.html`);
  await writeFile(file, html);
  console.log(file);
}
