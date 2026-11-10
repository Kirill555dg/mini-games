import { useState } from "preact/hooks";
import { GameBoard } from "@/entities/game/game-board";
import { FleetEditor } from "@/features/fleet/fleet-editor";
import type { Command, GameState } from "@/shared/api/game-api";
import { GAME_STATE } from "@/shared/api/protocol.generated";
import { t } from "@/shared/i18n/i18n";
import { ActionIcon } from "@/shared/ui/action-icon";
import { Button } from "@/shared/ui/ui-controls";
import "@/widgets/naval/naval-game.css";

type FieldMode = "turn" | "enemy" | "own" | "both";
type Props = {
  state: GameState;
  actor: number;
  enabled: boolean;
  onAction: (values: Partial<Command>) => Promise<boolean>;
  playerNames: string[];
};

export function NavalGame({ state, actor, enabled, onAction, playerNames }: Props) {
  const [mode, setMode] = useState<FieldMode>("turn");
  const player = actor >= 0 && actor < playerNames.length;
  const setup = state.phase === GAME_STATE.setup;
  const editing = player && setup && !state.ready?.[actor];
  const ownPlayer = player ? actor : 0;
  const enemyPlayer = player ? 1 - actor : 1;
  const defender = 1 - state.turn;
  const mobileView =
    mode === "turn" ? (setup ? (player ? "own" : "both") : defender === ownPlayer ? "own" : "enemy") : mode;
  const ownLabel = player
    ? t("yourField")
    : t("fieldOf", { name: playerNames[ownPlayer] || t("player", { n: ownPlayer + 1 }) });
  const enemyLabel = player
    ? t("enemyField")
    : t("fieldOf", { name: playerNames[enemyPlayer] || t("player", { n: enemyPlayer + 1 }) });
  return (
    <div class="naval-game">
      <fieldset class="naval-view-controls" aria-label={t("board")}>
        {(
          [
            ["turn", t("fieldCurrent"), "arrow"],
            ["enemy", enemyLabel, "ship"],
            ["own", ownLabel, "ship"],
            ["both", t("bothFields"), "grid"],
          ] as const
        ).map(([value, label, icon]) => (
          <Button
            key={value}
            class={`small ${mode === value ? "selected" : ""}`}
            aria-pressed={mode === value}
            onClick={() => setMode(value)}
          >
            <ActionIcon name={icon} />
            {label}
          </Button>
        ))}
      </fieldset>
      <div class="naval-fields" data-mobile-view={mobileView}>
        {([true, false] as const).map((own) => {
          const index = own ? ownPlayer : enemyPlayer;
          const defending = !setup && state.phase === GAME_STATE.playing && index === defender;
          return (
            <section
              class={`naval-field ${own ? "naval-own" : "naval-enemy"} ${defending ? "is-defending" : ""}`}
              key={String(own)}
            >
              <div class="naval-field-heading">
                <h3>{own ? ownLabel : enemyLabel}</h3>
                {defending && (
                  <span class="naval-defending">
                    <ActionIcon name="arrow" />
                    {t("incomingFire")}
                  </span>
                )}
              </div>
              {own && editing ? (
                <FleetEditor state={state} enabled={enabled} onAction={onAction} />
              ) : (
                <div class="naval-chart" style={{ "--columns": state.width, "--rows": state.height }}>
                  <div class="naval-columns" aria-hidden="true">
                    {Array.from({ length: state.width }, (_, column) => (
                      <span key={column}>{String.fromCharCode("A".charCodeAt(0) + column)}</span>
                    ))}
                  </div>
                  <div class="naval-rows" aria-hidden="true">
                    {Array.from({ length: state.height }, (_, row) => (
                      <span key={row}>{row + 1}</span>
                    ))}
                  </div>
                  <div class="naval-chart-board">
                    <GameBoard
                      game="battleship"
                      state={state}
                      actor={actor}
                      own={own}
                      enabled={!own && enabled && state.phase === GAME_STATE.playing && state.turn === actor}
                      onMove={(move) => void onAction({ kind: "move", ...move })}
                    />
                  </div>
                </div>
              )}
              {setup && !(own && editing) && (
                <p class="naval-field-note">{t(state.ready?.[index] ? "fleetPrepared" : "fleetWaiting")}</p>
              )}
            </section>
          );
        })}
      </div>
    </div>
  );
}
