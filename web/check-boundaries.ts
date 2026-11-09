import { readdir, readFile } from "node:fs/promises";
import path from "node:path";

const layers = ["app", "pages", "widgets", "features", "entities", "shared"];
async function walk(dir: string) {
  for (const item of await readdir(dir, { withFileTypes: true })) {
    const file = path.join(dir, item.name);
    if (item.isDirectory()) {
      await walk(file);
      continue;
    }
    if (!/\.[jt]sx?$/.test(file)) continue;
    const source = file.split(path.sep);
    const text = await readFile(file, "utf8");
    for (const match of text.matchAll(/(?:from\s*|import\s*)["']([^"']+)["']/g)) {
      if (match[1].startsWith(".")) throw new Error(`Relative import: ${file}`);
      if (!match[1].startsWith("@/")) continue;
      const target = `src/${match[1].slice(2)}`.split("/");
      const a = layers.indexOf(source[1]),
        b = layers.indexOf(target[1]);
      if (
        a < 0 ||
        b < 0 ||
        b < a ||
        (a === b && !["app", "shared"].includes(source[1]) && source[2] !== target[2])
      )
        throw new Error(`FSD boundary: ${file} -> ${match[1]}`);
      if (b > a && target.length > 4) throw new Error(`Import the slice's named public module: ${file}`);
    }
  }
}
await walk("src");
console.log("FSD boundaries OK");
