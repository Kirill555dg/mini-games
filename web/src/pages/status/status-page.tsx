import { useCallback, useEffect, useRef, useState } from "preact/hooks";
import { CONNECTION_POLICY } from "@/shared/api/connection-policy";
import { type AppStatus, api, unwrap } from "@/shared/api/game-api";
import { dateLocale, t } from "@/shared/i18n/i18n";
import { SECONDS } from "@/shared/ui/display-constants";
import { Button, Heading, Notice } from "@/shared/ui/ui-controls";

export function StatusPage({ connection }: { connection: (online: boolean) => void }) {
  const [value, setValue] = useState<AppStatus>(),
    [latency, setLatency] = useState(0),
    [checked, setChecked] = useState<Date>(),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const alive = useRef(true),
    controller = useRef<AbortController | null>(null);
  const check = useCallback(async () => {
    if (!alive.current) return;
    controller.current?.abort();
    const request = new AbortController();
    controller.current = request;
    setBusy(true);
    const start = performance.now();
    try {
      const data = await unwrap(api.GET("/game-api/status", { signal: request.signal }));
      if (!alive.current || controller.current !== request) return;
      setValue(data);
      setLatency(Math.round(performance.now() - start));
      setChecked(new Date());
      setError("");
      connection(true);
    } catch (failure) {
      if (!alive.current || controller.current !== request) return;
      setError(failure instanceof Error ? failure.message : t("error503"));
      connection(false);
    } finally {
      if (alive.current && controller.current === request) setBusy(false);
    }
  }, [connection]);
  useEffect(() => {
    alive.current = true;
    void check();
    const timer = setInterval(() => {
      if (!document.hidden) void check();
    }, CONNECTION_POLICY.statusRefreshMs);
    return () => {
      alive.current = false;
      controller.current?.abort();
      clearInterval(timer);
    };
  }, [check]);
  const seconds = value?.uptimeSeconds || 0;
  const duration =
    seconds >= SECONDS.day
      ? `${Math.floor(seconds / SECONDS.day)} ${t("dayShort")} ${Math.floor((seconds % SECONDS.day) / SECONDS.hour)} ${t("hourShort")}`
      : seconds >= SECONDS.hour
        ? `${Math.floor(seconds / SECONDS.hour)} ${t("hourShort")} ${Math.floor((seconds % SECONDS.hour) / SECONDS.minute)} ${t("minShort")}`
        : `${Math.floor(seconds / SECONDS.minute)} ${t("minShort")} ${seconds % SECONDS.minute} ${t("secShort")}`;
  return (
    <section class="status-page">
      <Heading title={t("status")} hint={t("statusHint")} />
      <div class="status-summary">
        <span class={`status-badge ${error ? "bad" : value ? "ok" : ""}`} role="status">
          {t(error ? "unavailable" : value ? "available" : "checking")}
        </span>
        <Button disabled={busy} onClick={() => void check()}>
          {t("checkNow")}
        </Button>
      </div>
      {value && (
        <dl class="status-metrics">
          {[
            [t("uptime"), duration],
            [t("responseTime"), `${latency} ${t("ms")}`],
            [t("startedAt"), new Date(value.startedAt).toLocaleString(dateLocale())],
            [t("checkedAt"), checked?.toLocaleTimeString(dateLocale()) || ""],
          ].map(([label, text]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>{text}</dd>
            </div>
          ))}
        </dl>
      )}
      {error && <Notice error>{error}</Notice>}
      <p class="muted status-explanation">{t("uptimeHint")}</p>
    </section>
  );
}
