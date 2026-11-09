import createClient from "openapi-fetch";
import { CONNECTION_POLICY } from "@/shared/api/connection-policy";
import type { components, paths } from "@/shared/api/openapi.generated";
import { language, t } from "@/shared/i18n/i18n";

export type GameSpec = components["schemas"]["engine_Spec"];
export type GameState = components["schemas"]["engine_State"];
export type Move = components["schemas"]["engine_Move"];
export type RoomView = components["schemas"]["rooms_View"];
export type RoomSummary = components["schemas"]["rooms_Summary"];
export type Lobby = components["schemas"]["rooms_Lobby"];
export type Command = components["schemas"]["rooms_Command"];
export type Match = components["schemas"]["history_Record"];
export type AppStatus = components["schemas"]["httpapi_Status"];

export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
  }
}
export const api = createClient<paths>({
  baseUrl: location.origin,
  async fetch(input) {
    const controller = new AbortController();
    const abort = () => controller.abort();
    input.signal.addEventListener("abort", abort, { once: true });
    if (input.signal.aborted) abort();
    const timer = setTimeout(abort, CONNECTION_POLICY.requestTimeoutMs);
    try {
      const response = await fetch(input, {
        signal: controller.signal,
        cache: "no-store",
        credentials: "same-origin",
      });
      // These endpoints return finite JSON. Keep the deadline through body consumption.
      const body = await response.arrayBuffer();
      return new Response(body, {
        status: response.status,
        statusText: response.statusText,
        headers: response.headers,
      });
    } finally {
      clearTimeout(timer);
      input.signal.removeEventListener("abort", abort);
    }
  },
});
export async function unwrap<T>(
  pending: Promise<{ data?: T; error?: { error: string }; response: Response }>,
): Promise<T> {
  try {
    const result = await pending;
    if (!result.response.ok)
      throw new APIError(
        language() === "ru" && result.error?.error
          ? result.error.error
          : t(
              `error${[400, 403, 404, 409, 429].includes(result.response.status) ? result.response.status : 503}`,
            ),
        result.response.status,
      );
    return result.data as T;
  } catch (error) {
    if (error instanceof APIError) throw error;
    throw new APIError(t("noNetwork"), 0);
  }
}
