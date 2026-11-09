import type { Command, GameSpec, GameState } from "@/shared/api/game-api";
import { t, translated } from "@/shared/i18n/i18n";
import { Mark } from "@/shared/ui/ui-controls";

export const gameTitle = (catalog: GameSpec[], id: string) =>
  translated(
    catalog.find((game) => game.id === id),
    "title",
  ) || t("games");
export type BoardAction = Pick<Command, "cell" | "kind" | "vertical">;
type Props = {
  game: string;
  state: GameState;
  actor?: number;
  enabled?: boolean;
  onMove?: (move: BoardAction) => void;
  own?: boolean;
  vertical?: boolean;
};

export function GameBoard({
  game,
  state,
  actor = -1,
  enabled = false,
  onMove = () => {},
  own = false,
  vertical = false,
}: Props) {
  if (game === "dotsboxes") return <DotsBoard state={state} enabled={enabled} onMove={onMove} />;
  const naval = game === "battleship";
  const values = naval
    ? (own ? state.own : state.enemy) || Array(state.width * state.height).fill(0)
    : state.board || [];
  return (
    <fieldset
      class={`board ${naval ? "naval-board" : game === "connectfour" ? "connect-board" : "ttt-board"}`}
      aria-label={naval ? t(own ? "yourField" : "enemyField") : t("board")}
      style={{ "--columns": state.width, "--rows": state.height }}
      onKeyDown={(event) => {
        const cells = [...event.currentTarget.querySelectorAll<HTMLButtonElement>("button")];
        const index = cells.indexOf(document.activeElement as HTMLButtonElement);
        const delta: Record<string, number> = {
          ArrowLeft: -1,
          ArrowRight: 1,
          ArrowUp: -state.width,
          ArrowDown: state.width,
        };
        if (index >= 0 && delta[event.key] && cells[index + delta[event.key]]) {
          event.preventDefault();
          cells[index + delta[event.key]].focus();
        }
      }}
    >
      {values.map((value, index) => {
        const placing =
          naval &&
          own &&
          state.phase === "setup" &&
          actor >= 0 &&
          !state.ready?.[actor] &&
          Boolean(state.fleet?.length);
        const allowed =
          enabled &&
          (naval
            ? placing || (!own && state.phase === "playing" && !value)
            : game === "connectfour"
              ? !values[index % state.width]
              : !value);
        const cell = `${String.fromCharCode(65 + (index % state.width))}${Math.floor(index / state.width) + 1}`;
        const label = naval
          ? t(value === 1 ? "ship" : value === 2 ? "hit" : value === -1 ? "miss" : "unknown")
          : value
            ? t("cellPlayer", { n: value })
            : t("emptyCell");
        return (
          <button
            type="button"
            key={index}
            class={`cell player-${value} ${naval && value === 1 ? "ship" : ""} ${value === 2 && naval ? "hit" : ""} ${state.line?.includes(index) ? "winning" : ""} ${state.moves?.at(-1)?.cell === index && !naval ? "last-move" : ""}`}
            aria-label={`${cell}: ${label}`}
            aria-disabled={!allowed}
            data-cell={index}
            data-field={naval ? (own ? "own" : "enemy") : "board"}
            onClick={() => {
              if (allowed)
                onMove(
                  placing
                    ? { kind: "place", cell: index, vertical }
                    : { cell: game === "connectfour" ? index % state.width : index },
                );
            }}
          >
            {naval ? (
              value === 2 ? (
                <Mark player={1} />
              ) : value === -1 ? (
                <span class="miss" />
              ) : null
            ) : game === "connectfour" ? (
              value ? (
                <span class={`disc player-${value}`} />
              ) : null
            ) : (
              <Mark player={value} />
            )}
          </button>
        );
      })}
    </fieldset>
  );
}
function DotsBoard({
  state,
  enabled,
  onMove,
}: {
  state: GameState;
  enabled: boolean;
  onMove: (move: BoardAction) => void;
}) {
  const size = state.width,
    horizontal = (size + 1) * size;
  return (
    <fieldset class="board dots-board" aria-label={t("edges")}>
      {Array.from({ length: (size * 2 + 1) ** 2 }, (_, position) => {
        const row = Math.floor(position / (size * 2 + 1)),
          col = position % (size * 2 + 1);
        if (!(row % 2) && !(col % 2)) return <span key={position} class="dot" />;
        if (row % 2 && col % 2) {
          const owner = state.boxes?.[Math.floor(row / 2) * size + Math.floor(col / 2)] || 0;
          return (
            <span key={position} class={`box player-${owner}`}>
              {owner || ""}
            </span>
          );
        }
        const isHorizontal = row % 2 === 0;
        const index = isHorizontal
          ? (row / 2) * size + (col - 1) / 2
          : horizontal + ((row - 1) / 2) * (size + 1) + col / 2;
        const owner = state.board?.[index] || 0;
        return (
          <button
            type="button"
            key={position}
            class={`edge ${isHorizontal ? "horizontal" : "vertical"} player-${owner}`}
            data-cell={index}
            aria-label={`${t(isHorizontal ? "edgeH" : "edgeV")} ${Math.floor(row / 2) + 1}, ${Math.floor(col / 2) + 1}: ${t(owner ? "taken" : "emptyCell")}`}
            aria-disabled={Boolean(owner) || !enabled}
            onClick={() => {
              if (enabled && !owner) onMove({ cell: index });
            }}
          />
        );
      })}
    </fieldset>
  );
}
export function replayState(game: string, state: GameState, count: number, viewer = -1): GameState {
  if (count >= (state.moves?.length || 0)) return state;
  const copy: GameState = {
    ...state,
    line: [],
    boxes: [],
    scores: [],
    moves: (state.moves || []).slice(0, count),
  };
  if (game === "battleship") {
    copy.own = Array(state.width * state.height).fill(0);
    copy.enemy = [...copy.own];
    for (const move of copy.moves || [])
      (move.player === Math.max(viewer, 0) ? copy.enemy : copy.own)[move.cell] = move.hit ? 2 : -1;
  } else {
    copy.board = Array(state.board?.length || 0).fill(0);
    if (game === "dotsboxes") {
      copy.boxes = Array(state.width * state.height).fill(0);
      copy.scores = Array(state.scores?.length || 4).fill(0);
    }
    for (const move of copy.moves || []) {
      copy.board[move.cell] = move.player + 1;
      for (const box of move.boxes || []) {
        if (copy.boxes && copy.scores) {
          copy.boxes[box] = move.player + 1;
          copy.scores[move.player]++;
        }
      }
    }
  }
  return copy;
}
