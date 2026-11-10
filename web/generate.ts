import { spawnSync } from "node:child_process";
import { readFile, writeFile } from "node:fs/promises";

const result = spawnSync(
  process.execPath,
  [
    "node_modules/openapi-typescript/bin/cli.js",
    "../api/openapi.json",
    "-o",
    "src/shared/api/openapi.generated.ts",
  ],
  { stdio: "inherit" },
);
if (result.status) process.exit(result.status);
const schema = JSON.parse(await readFile("../api/openapi.json", "utf8"));
const code = schema["x-room-code"],
  state = schema["x-game-values"],
  actions = schema["x-room-actions"],
  naval = schema["x-naval-cells"];
if (!code || !state || !actions || !naval) throw new Error("Missing protocol constants in OpenAPI");
await writeFile(
  "src/shared/api/protocol.generated.ts",
  `// Generated from api/openapi.json. Do not edit.\nexport const ROOM_CODE = ${JSON.stringify(code)} as const;\nexport const GAME_STATE = ${JSON.stringify(state)} as const;\nexport const ROOM_ACTION = ${JSON.stringify(actions)} as const;\nexport const NAVAL_CELL = ${JSON.stringify(naval)} as const;\n`,
);
