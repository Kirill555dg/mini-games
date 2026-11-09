import { copyFile, mkdir, readdir, readFile, rm, writeFile } from "node:fs/promises";

await rm("dist", { recursive: true, force: true });
await mkdir("dist", { recursive: true });
const result = await Bun.build({
  entrypoints: ["src/app/main.tsx"],
  outdir: "dist",
  naming: "app-[hash].[ext]",
  minify: true,
  target: "browser",
});
if (!result.success) throw new AggregateError(result.logs, "Frontend build failed");
const files = await readdir("dist");
const js = files.find((file) => file.endsWith(".js"));
const css = files.find((file) => file.endsWith(".css"));
if (!js || !css) throw new Error("Missing JS/CSS build output");
const html = (await readFile("index.html", "utf8"))
  .replace("@SCRIPT@", `/${js}`)
  .replace("@STYLE@", `/${css}`);
for (const name of ["index.html", "status.html", "404.html"])
  await writeFile(`dist/${name}`, `${html.replace(/>\s+</g, "><").trim()}\n`);
for (const name of await readdir("public")) await copyFile(`public/${name}`, `dist/${name}`);
console.log(
  `Built ${js}, ${css}; ${result.outputs.reduce((sum, output) => sum + output.size, 0)} bytes before compression`,
);
