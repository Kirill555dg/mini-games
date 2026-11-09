import { useEffect, useState } from "preact/hooks";
import { GameBoard, gameTitle, replayState } from "@/entities/game/game-board";
import { api, type GameSpec, type Match, unwrap } from "@/shared/api/game-api";
import { GAME_STATE, ROOM_CODE } from "@/shared/api/protocol.generated";
import { dateLocale, t } from "@/shared/i18n/i18n";
import { HISTORY_PAGE_SIZE } from "@/shared/ui/display-constants";
import { Button, Heading, Notice } from "@/shared/ui/ui-controls";
import { MoveReview } from "@/widgets/replay/move-review";

export function HistoryPage({ catalog }: { catalog: GameSpec[] }) {
  const [records, setRecords] = useState<Match[]>(),
    [error, setError] = useState(""),
    [code, setCode] = useState(""),
    [game, setGame] = useState(""),
    [visible, setVisible] = useState(HISTORY_PAGE_SIZE);
  useEffect(() => {
    let alive = true;
    unwrap(api.GET("/game-api/history"))
      .then((data) => {
        if (alive) setRecords(data.reverse());
      })
      .catch((failure) => {
        if (alive) setError(failure.message);
      });
    return () => {
      alive = false;
    };
  }, []);
  const filtered =
    records?.filter(
      (record) => record.code.includes(code.toUpperCase()) && (!game || game === record.game),
    ) || [];
  return (
    <section class="history-page">
      <Heading title={t("history")} hint={t("historyHint")} />
      <div class="history-filters">
        <input
          type="search"
          value={code}
          maxLength={ROOM_CODE.length}
          placeholder={t("code")}
          aria-label={t("historyCode")}
          onInput={(event) => {
            setCode(event.currentTarget.value);
            setVisible(HISTORY_PAGE_SIZE);
          }}
        />
        <select
          value={game}
          aria-label={t("games")}
          onChange={(event) => {
            setGame(event.currentTarget.value);
            setVisible(HISTORY_PAGE_SIZE);
          }}
        >
          <option value="">{t("allGames")}</option>
          {catalog.map((spec) => (
            <option value={spec.id} key={spec.id}>
              {gameTitle(catalog, spec.id)}
            </option>
          ))}
        </select>
      </div>
      {error ? (
        <Notice error>{error}</Notice>
      ) : !records ? (
        <Notice>{t("loading")}</Notice>
      ) : !filtered.length ? (
        <Notice>{t(records.length ? "historyFiltered" : "historyEmpty")}</Notice>
      ) : (
        <div class="match-list">
          {filtered.slice(0, visible).map((record) => (
            <MatchReview key={record.id} record={record} catalog={catalog} />
          ))}
        </div>
      )}
      {filtered.length > visible && (
        <Button onClick={() => setVisible(visible + HISTORY_PAGE_SIZE)}>{t("more")}</Button>
      )}
    </section>
  );
}
function MatchReview({ record, catalog }: { record: Match; catalog: GameSpec[] }) {
  const total = record.state.moves?.length || 0;
  const [index, setIndex] = useState(total),
    [open, setOpen] = useState(false);
  const state = replayState(record.game, record.state, index);
  const outcome =
    record.reason && record.winner === GAME_STATE.draw
      ? t("interrupted")
      : record.winner === GAME_STATE.draw
        ? t("draw")
        : t("winner", { name: record.players[record.winner] });
  return (
    <details class="match" onToggle={(event) => setOpen(event.currentTarget.open)}>
      <summary>
        <span class="match-code">{record.code}</span>
        <span class="match-description">
          <strong>{gameTitle(catalog, record.game)}</strong>
          <span class="muted">{record.players.join(" · ")}</span>
        </span>
        <span class="match-result">{outcome}</span>
        <time dateTime={record.finishedAt}>
          {new Date(record.finishedAt).toLocaleDateString(dateLocale(), { day: "numeric", month: "short" })}
        </time>
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
