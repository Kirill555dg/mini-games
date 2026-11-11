import { useState } from "preact/hooks";
import { cellsInRegion } from "@/entities/game/nested-board";
import type { GameState } from "@/shared/api/game-api";
import { GAME_STATE } from "@/shared/api/protocol.generated";
import { t } from "@/shared/i18n/i18n";
import { Mark } from "@/shared/ui/ui-controls";
import "@/entities/game/ultimate-board.css";

export function UltimateBoard({
  state,
  enabled,
  play,
}: {
  state: GameState;
  enabled: boolean;
  play: (cell: number) => void;
}) {
  const [focused, setFocused] = useState(0);
  const size = state.regionSize || 0;
  if (!size) return null;
  const regions = state.regions || [];
  const board = state.board || [];
  const active = state.activeBoard;
  const firstTarget = regions.flatMap((result, region) =>
    result === 0 && (active === undefined || active === region)
      ? cellsInRegion(region, state.width, size).filter((cell) => !board[cell])
      : [],
  );
  const tabCell = firstTarget.includes(focused) ? focused : (firstTarget[0] ?? focused);
  return (
    <div class="ultimate-game">
      <fieldset
        class="board ultimate-board"
        aria-label={t("board")}
        style={{ "--regions-columns": state.width / size, "--region-size": size }}
        onKeyDown={(event) => {
          if (!(event.target instanceof HTMLButtonElement)) return;
          const cell = Number(event.target.dataset.cell);
          const delta: Record<string, number> = {
            ArrowLeft: -1,
            ArrowRight: 1,
            ArrowUp: -state.width,
            ArrowDown: state.width,
          };
          const target =
            delta[event.key] === undefined
              ? undefined
              : event.currentTarget.querySelector<HTMLButtonElement>(
                  `[data-cell="${cell + delta[event.key]}"]`,
                );
          if (target) {
            event.preventDefault();
            target.focus();
          }
        }}
      >
        {regions.map((result, region) => {
          const target =
            state.phase === GAME_STATE.playing && !result && (active === undefined || active === region);
          const won = result > 0;
          const label = `${t("smallBoard", { n: region + 1 })}${result ? `: ${t(won ? "regionClaimed" : "draw", { n: result })}` : ""}`;
          return (
            <fieldset
              class={`ultimate-region player-${won ? result : 0} ${target && active !== undefined ? "is-target" : ""} ${result ? "is-closed" : ""} ${state.regionLine?.includes(region) ? "is-winning" : ""}`}
              key={region}
              data-region={region}
              data-closed={Boolean(result)}
              aria-label={label}
            >
              <div class="ultimate-cells">
                {cellsInRegion(region, state.width, size).map((cell) => {
                  const mark = board[cell];
                  const allowed = enabled && target && !mark;
                  const coordinate = `${String.fromCharCode("A".charCodeAt(0) + (cell % state.width))}${Math.floor(cell / state.width) + 1}`;
                  return (
                    <button
                      type="button"
                      key={cell}
                      class={`cell player-${mark} ${state.moves?.at(-1)?.cell === cell ? "last-move" : ""}`}
                      data-cell={cell}
                      aria-label={`${coordinate}: ${t(mark ? "cellPlayer" : "emptyCell", { n: mark })}`}
                      aria-disabled={!allowed}
                      tabIndex={tabCell === cell ? 0 : -1}
                      onFocus={() => setFocused(cell)}
                      onClick={() => {
                        if (allowed) play(cell);
                      }}
                    >
                      <Mark player={mark} />
                    </button>
                  );
                })}
              </div>
              {result ? (
                <span class="region-result" aria-hidden="true">
                  {won ? <Mark player={result} /> : "="}
                </span>
              ) : null}
            </fieldset>
          );
        })}
      </fieldset>
      {state.phase !== GAME_STATE.finished && (
        <p class="ultimate-hint" role="status">
          {t(active === undefined ? "ultimateAnyBoard" : "ultimateActiveBoard", { n: (active ?? 0) + 1 })}
        </p>
      )}
    </div>
  );
}
