import { ROOM_CODE } from "@/shared/api/protocol.generated";
export const codePattern = new RegExp(`^[${ROOM_CODE.alphabet}]{${ROOM_CODE.length}}$`);
export const INVITE_INPUT_MAX_LENGTH = 512;
export function parseInvitation(value: string): { code: string; watch?: boolean; seat?: number } {
  let candidate = value.trim();
  let watch: boolean | undefined, seat: number | undefined;
  if (candidate.includes("/")) {
    try {
      const url = new URL(candidate);
      const [path, query] = url.hash.split("?");
      candidate = path.match(/^#\/(?:join|room)\/([A-Za-z0-9-]+)$/i)?.[1] || "";
      const params = new URLSearchParams(query);
      watch = params.get("mode") === "watch";
      const raw = params.get("seat");
      if (raw !== null && /^\d+$/.test(raw) && Number.isSafeInteger(Number(raw))) seat = Number(raw);
    } catch {
      return { code: "" };
    }
  }
  candidate = candidate.toUpperCase().replace(/[\s-]/g, "");
  return { code: codePattern.test(candidate) ? candidate : "", watch, seat: watch ? undefined : seat };
}
export function normalizeCode(value: string) {
  return parseInvitation(value).code;
}

export function invitation(code: string, { watch = false, seat }: { watch?: boolean; seat?: number } = {}) {
  const url = new URL(`/#/join/${code}`, location.origin);
  const query = new URLSearchParams();
  if (watch) query.set("mode", "watch");
  if (seat !== undefined && seat !== null && !watch) query.set("seat", String(seat));
  if (query.size) url.hash += `?${query}`;
  return url.href;
}
export function savedName() {
  try {
    return localStorage.getItem("mini-games-name") || localStorage.getItem("game-name") || "";
  } catch {
    return "";
  }
}
export function saveName(name: string) {
  try {
    localStorage.setItem("mini-games-name", name.trim());
  } catch {
    /* Optional. */
  }
}
