import { useCallback, useEffect, useRef, useState } from "preact/hooks";
import { CONNECTION_POLICY } from "@/shared/api/connection-policy";
import { type AppStatus, api, unwrap } from "@/shared/api/game-api";
import { dateLocale, t } from "@/shared/i18n/i18n";
import { ActionIcon } from "@/shared/ui/action-icon";
import { SECONDS } from "@/shared/ui/display-constants";
import { Button, Heading, Notice } from "@/shared/ui/ui-controls";

const STATE_LABEL = {
  ok: "statusOperational",
  degraded: "statusDegraded",
  down: "statusDown",
  unknown: "statusUnknown",
} as const;
export function StatusPage({ connection }: { connection: (online: boolean) => void }) {
  const [value, setValue] = useState<AppStatus>(),
    [latency, setLatency] = useState(0),
    [checked, setChecked] = useState<Date>(),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [selected, setSelected] = useState("");
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
  const days = value?.history.days || [],
    day = days.find((item) => item.date === selected) || days.at(-1);
  const date = (value: string) =>
    new Date(`${value}T00:00:00Z`).toLocaleDateString(dateLocale(), {
      day: "numeric",
      month: "long",
      timeZone: "UTC",
    });
  return (
    <section class="status-page">
      <Heading title={t("availabilityTitle")} hint={t("availabilityHint")} />
      <div class={`availability-banner ${error ? "down" : value ? "ok" : "unknown"}`}>
        <span class="status-orbit">
          <ActionIcon name={error ? "close" : "check"} />
        </span>
        <div class="availability-message">
          <h2 aria-live="polite">{t(error ? "unavailable" : value ? "statusOperational" : "checking")}</h2>
          <p>
            {checked ? `${t("checkedAt")}: ${checked.toLocaleTimeString(dateLocale())}` : t("checking")}
            {busy && value ? ` · ${t("checking")}` : ""}
          </p>
        </div>
        <Button disabled={busy} onClick={() => void check()}>
          <ActionIcon name="rotate" />
          {t("checkNow")}
        </Button>
      </div>
      {error && <Notice error>{error}</Notice>}
      <section class="availability-history" aria-labelledby="availability-history-title">
        <div class="section-heading">
          <h2 id="availability-history-title">{t("statusHistory")}</h2>
          {value && (
            <span class="muted availability-period">{t("last30Days", { n: value.history.periodDays })}</span>
          )}
        </div>
        {value ? (
          <>
            <fieldset class="availability-bars" aria-label={t("statusHistory")}>
              {days.map((item) => {
                const total = item.observedSeconds + item.unknownSeconds,
                  coverage = total ? item.observedSeconds / total : 0;
                const label = `${date(item.date)}: ${t(STATE_LABEL[item.state])}, ${t("statusCoverage", { n: Math.floor(item.observedSeconds / SECONDS.minute) })}`;
                return (
                  <button
                    type="button"
                    key={item.date}
                    class={`availability-day ${item.state}`}
                    aria-pressed={day?.date === item.date}
                    aria-label={label}
                    title={label}
                    onClick={() => setSelected(item.date)}
                    style={{ "--coverage": `${coverage * 100}%` }}
                  >
                    <span />
                  </button>
                );
              })}
            </fieldset>
            <div class="timeline-labels">
              <span>{days[0] ? date(days[0].date) : ""}</span>
              <span>{t("today")}</span>
            </div>
            <div class="availability-legend">
              {Object.entries(STATE_LABEL).map(([state, label]) => (
                <span key={state}>
                  <i class={state} />
                  {t(label)}
                </span>
              ))}
            </div>
            {day && (
              <div class="availability-detail" role="status">
                <strong>{date(day.date)}</strong>
                <span class={`day-state ${day.state}`}>{t(STATE_LABEL[day.state])}</span>
                <div class="day-detail-metrics">
                  <span>{t("statusCoverage", { n: Math.floor(day.observedSeconds / SECONDS.minute) })}</span>
                  <span>{t("statusUnknownTime", { n: Math.ceil(day.unknownSeconds / SECONDS.minute) })}</span>
                  <span>{t("statusRequests", { n: day.requests })}</span>
                  <span>{t("statusErrors", { n: day.serverErrors })}</span>
                </div>
              </div>
            )}
          </>
        ) : error ? (
          <div class="empty-state">
            <p>{t("statusHistoryUnavailable")}</p>
            <Button disabled={busy} onClick={() => void check()}>
              {t("retry")}
            </Button>
          </div>
        ) : (
          <div class="availability-skeleton" role="status">
            {t("loading")}
          </div>
        )}
        <p class="status-explanation">{t("statusSelfObservation")}</p>
        {value && <p class="field-help">{t("statusSampling", { n: value.history.sampleIntervalSeconds })}</p>}
      </section>
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
      <p class="field-help">{t("uptimeHint")}</p>
    </section>
  );
}
