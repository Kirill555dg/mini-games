import { useEffect, useState } from "preact/hooks";
import { GameBoard, gameTitle, replayState } from "@/entities/game/game-board";
import { api, type GameSpec, type Match, unwrap } from "@/shared/api/game-api";
import { GAME_STATE, ROOM_CODE } from "@/shared/api/protocol.generated";
import { dateLocale, t } from "@/shared/i18n/i18n";
import { ActionIcon } from "@/shared/ui/action-icon";
import { HISTORY_PAGE_SIZE } from "@/shared/ui/display-constants";
import { Button, Heading, Notice } from "@/shared/ui/ui-controls";
import { MoveReview } from "@/widgets/replay/move-review";
import "@/pages/history/history-page.css";

function filtersFromURL() {
  const params = new URLSearchParams(location.hash.split("?")[1]);
  return { games: params.getAll("game"), code: params.get("code") || "", match: params.get("match") || "" };
}
const COMPACT_FILTER_QUERY = "(max-width: 760px)";

export function HistoryPage({ catalog }: { catalog: GameSpec[] }) {
  const [records, setRecords] = useState<Match[]>(),
    [error, setError] = useState(""),
    [attempt, setAttempt] = useState(0);
  const [filterOpen, setFilterOpen] = useState(() => !matchMedia(COMPACT_FILTER_QUERY).matches);
  const [filters, setFilters] = useState(filtersFromURL),
    [visible, setVisible] = useState(HISTORY_PAGE_SIZE);
  useEffect(() => {
    let alive = true;
    const controller = new AbortController();
    setError("");
    unwrap(api.GET("/game-api/history", { signal: controller.signal }))
      .then((data) => {
        if (alive) setRecords(data.reverse());
      })
      .catch((failure) => {
        if (alive) setError(failure.message);
      });
    const sync = () => setFilters(filtersFromURL());
    window.addEventListener("hashchange", sync);
    return () => {
      alive = false;
      controller.abort();
      window.removeEventListener("hashchange", sync);
    };
  }, [attempt]);
  function change(next: Partial<typeof filters>) {
    const value = { ...filters, ...next };
    setFilters(value);
    if (next.games !== undefined || next.code !== undefined) setVisible(HISTORY_PAGE_SIZE);
    const params = new URLSearchParams();
    for (const game of value.games) params.append("game", game);
    if (value.code) params.set("code", value.code);
    if (value.match) params.set("match", value.match);
    history.replaceState(null, "", `/#/history${params.size ? `?${params}` : ""}`);
  }
  const filtered =
    records?.filter(
      (record) =>
        record.code.includes(filters.code.toUpperCase()) &&
        (!filters.games.length || filters.games.includes(record.game)),
    ) || [];
  const active = Boolean(filters.games.length || filters.code);
  const shown = Math.max(visible, filtered.findIndex((record) => String(record.id) === filters.match) + 1);
  return (
    <section class="history-page">
      <Heading title={t("history")} hint={t("historyHint")} />
      <details
        class="game-filter-menu"
        open={filterOpen}
        onToggle={(event) => setFilterOpen(event.currentTarget.open)}
      >
        <summary>
          <ActionIcon name="grid" />
          <strong>{t("selectedGames")}</strong>
          <span>
            {filters.games.length ? t("selectedCount", { n: filters.games.length }) : t("allGames")}
          </span>
          <ActionIcon name="chevron" />
        </summary>
        <fieldset class="game-filters" aria-label={t("selectedGames")}>
          <Button
            class="game-filter"
            aria-pressed={!filters.games.length}
            onClick={() => change({ games: [], match: "" })}
          >
            <ActionIcon name="grid" />
            {t("allGames")}
          </Button>
          {catalog.map((game) => (
            <Button
              class="game-filter"
              key={game.id}
              aria-pressed={filters.games.includes(game.id)}
              onClick={() =>
                change({
                  games: filters.games.includes(game.id)
                    ? filters.games.filter((value) => value !== game.id)
                    : [...filters.games, game.id],
                  match: "",
                })
              }
            >
              <img src={`/${game.id}.svg`} width="30" height="25" alt="" />
              <span>{gameTitle(catalog, game.id)}</span>
              <span class="filter-check" aria-hidden="true">
                {filters.games.includes(game.id) && <ActionIcon name="check" />}
              </span>
            </Button>
          ))}
        </fieldset>
      </details>
      <div class="history-filters">
        <label class="search-field">
          <ActionIcon name="search" />
          <input
            type="search"
            name="match-code"
            value={filters.code}
            maxLength={ROOM_CODE.length}
            placeholder={t("historyCode")}
            aria-label={t("historyCode")}
            autoCapitalize="characters"
            autoComplete="off"
            spellcheck={false}
            onInput={(event) => change({ code: event.currentTarget.value, match: "" })}
          />
        </label>
        <span class="muted">{t("historyCount", { n: filtered.length })}</span>
        {active && (
          <Button class="small" onClick={() => change({ games: [], code: "", match: "" })}>
            {t("clearFilters")}
          </Button>
        )}
      </div>
      {error ? (
        <div class="empty-state">
          <Notice error>{error}</Notice>
          <Button onClick={() => setAttempt(attempt + 1)}>{t("retry")}</Button>
        </div>
      ) : !records ? (
        <Notice>{t("loading")}</Notice>
      ) : !filtered.length ? (
        <div class="empty-state">
          <ActionIcon name="clock" />
          <h2>{t(records.length ? "historyFiltered" : "historyEmpty")}</h2>
          {active && (
            <Button onClick={() => change({ games: [], code: "", match: "" })}>{t("clearFilters")}</Button>
          )}
        </div>
      ) : (
        <div class="match-list">
          {filtered.slice(0, shown).map((record) => (
            <MatchReview
              key={record.id}
              record={record}
              catalog={catalog}
              open={filters.match === String(record.id)}
              toggle={(open) => change({ match: open ? String(record.id) : "" })}
            />
          ))}
        </div>
      )}
      {filtered.length > shown && (
        <Button onClick={() => setVisible(shown + HISTORY_PAGE_SIZE)}>{t("more")}</Button>
      )}
    </section>
  );
}
function MatchReview({
  record,
  catalog,
  open,
  toggle,
}: {
  record: Match;
  catalog: GameSpec[];
  open: boolean;
  toggle: (open: boolean) => void;
}) {
  const total = record.state.moves?.length || 0,
    [index, setIndex] = useState(total);
  const state = replayState(record.game, record.state, index);
  const outcome =
    record.reason && record.winner === GAME_STATE.draw
      ? t("interrupted")
      : record.winner === GAME_STATE.draw
        ? t("draw")
        : t("winner", { name: record.players[record.winner] });
  return (
    <details
      class="match"
      open={open}
      onToggle={(event) => {
        if (event.currentTarget.open !== open) toggle(event.currentTarget.open);
      }}
    >
      <summary>
        <img class="match-icon" src={`/${record.game}.svg`} width="56" height="46" alt="" />
        <span class="match-description">
          <strong>{gameTitle(catalog, record.game)}</strong>
          <span class="muted">{record.players.join(" · ")}</span>
        </span>
        <span class="match-outcome">
          <span class="match-result">{outcome}</span>
          <span class="match-code" translate={false}>
            {record.code}
          </span>
        </span>
        <time dateTime={record.finishedAt}>
          {new Date(record.finishedAt).toLocaleDateString(dateLocale(), { day: "numeric", month: "short" })}
        </time>
        <ActionIcon name="chevron" />
      </summary>
      {open && (
        <div class="match-review">
          <div class={`review-boards ${record.game === "battleship" ? "naval-review" : ""}`}>
            {record.game === "battleship" ? (
              [true, false].map((own) => (
                <section key={String(own)}>
                  <h3>{t("shotsBy", { name: record.players[own ? 1 : 0] })}</h3>
                  <GameBoard game={record.game} state={state} own={own} />
                </section>
              ))
            ) : (
              <GameBoard game={record.game} state={state} />
            )}
          </div>
          <MoveReview index={index} total={total} change={setIndex} />
        </div>
      )}
    </details>
  );
}
