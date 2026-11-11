import { CONNECTION_POLICY } from "@/shared/api/connection-policy";
import { APIError, api, type Lobby, type RoomView, unwrap } from "@/shared/api/game-api";

// SSE is an optimization; bounded polling keeps the same API and permissions.
export function live<T extends Lobby | RoomView>(
  room: string,
  update: (value: T | null) => void,
  connection: (online: boolean) => void,
) {
  let source: EventSource | undefined,
    timer: ReturnType<typeof setInterval>,
    stopped = false,
    lastEvent = Date.now(),
    streamHealthy = false,
    polling = false;
  const controller = new AbortController();
  let instance = "",
    revision = -1;
  function accept(data: T | null) {
    if (stopped) return;
    if (!data && room) {
      stopped = true;
      source?.close();
      clearInterval(timer);
      controller.abort();
      update(null);
      return;
    }
    connection(navigator.onLine !== false);
    if (data && data.instance === instance && data.revision < revision) return;
    if (data) {
      instance = data.instance;
      revision = data.revision;
    }
    update(data);
  }
  async function poll() {
    if (polling || stopped || document.hidden || navigator.onLine === false) return;
    polling = true;
    try {
      const value = room
        ? await unwrap(api.GET("/game-api/state", { params: { query: { room } }, signal: controller.signal }))
        : await unwrap(api.GET("/game-api/lobby", { signal: controller.signal }));
      accept(value as T);
    } catch (error) {
      if (!stopped) {
        if (room && error instanceof APIError && error.status === 404) accept(null);
        else connection(false);
      }
    } finally {
      polling = false;
    }
  }
  function connect() {
    if (stopped || !window.EventSource || navigator.onLine === false) return;
    source = new EventSource(`/game-api/events${room ? `?room=${encodeURIComponent(room)}` : ""}`);
    lastEvent = Date.now();
    streamHealthy = false;
    source.addEventListener("state", (event) => {
      try {
        lastEvent = Date.now();
        streamHealthy = true;
        accept(JSON.parse(event.data) as T | null);
      } catch {
        connection(false);
      }
    });
    source.onerror = () => {
      if (!stopped) {
        streamHealthy = false;
        connection(false);
        void poll();
      }
    };
  }
  connect();
  void poll();
  timer = setInterval(() => {
    if (!streamHealthy || Date.now() - lastEvent > CONNECTION_POLICY.staleAfterMs) void poll();
    if (source && Date.now() - lastEvent > CONNECTION_POLICY.reconnectAfterMs) {
      source.close();
      connect();
    }
  }, CONNECTION_POLICY.pollIntervalMs);
  const visible = () => {
    if (!document.hidden) void poll();
  };
  const offline = () => {
    streamHealthy = false;
    source?.close();
    source = undefined;
    connection(false);
  };
  const online = () => {
    source?.close();
    connect();
    void poll();
  };
  document.addEventListener("visibilitychange", visible);
  window.addEventListener("offline", offline);
  window.addEventListener("online", online);
  return () => {
    stopped = true;
    source?.close();
    controller.abort();
    clearInterval(timer);
    document.removeEventListener("visibilitychange", visible);
    window.removeEventListener("offline", offline);
    window.removeEventListener("online", online);
  };
}
