import type { GameSpec, GameState, RoomView } from "@/shared/api/game-api";
import { GAME_STATE } from "@/shared/api/protocol.generated";
import { sideName, t } from "@/shared/i18n/i18n";
import { ActionIcon } from "@/shared/ui/action-icon";
import { Mark } from "@/shared/ui/ui-controls";

export function RoomSeats({
  room,
  state,
  actor,
  spec,
  enabled,
  choose,
}: {
  room: RoomView["room"];
  state: GameState;
  actor: number;
  spec?: GameSpec;
  enabled: boolean;
  choose: (seat: number) => void;
}) {
  const preparing = !room.started && !room.finished;
  return (
    <fieldset class="seat-list" aria-label={t("playerCount")} style={{ "--seats": room.seats.length }}>
      {room.seats.map((seat, index) => {
        const mine = actor === index;
        const selectable = preparing && actor >= 0 && !mine && enabled;
        const presence = t(
          !seat.taken
            ? "waitingPlayer"
            : state.phase === GAME_STATE.setup && room.started
              ? state.ready?.[index]
                ? "fleetPrepared"
                : "placingFleet"
              : preparing && seat.ready
                ? "readyLabel"
                : seat.online
                  ? "online"
                  : "returnGrace",
        );
        const detail = `${sideName(spec, index)} · ${presence}`;
        return (
          <button
            type="button"
            key={index}
            class={`seat-card player-${index + 1} ${mine ? "is-you" : ""} ${room.started && !room.finished && state.turn === index ? "is-turn" : ""}`}
            aria-pressed={preparing ? mine : undefined}
            aria-label={
              selectable
                ? t(seat.taken ? "offerSeatSwap" : "chooseSeat", {
                    side: sideName(spec, index),
                    name: seat.name,
                  })
                : undefined
            }
            disabled={!selectable}
            onClick={() => choose(index)}
          >
            <span class="seat-number">
              {room.game === "tictactoe" || room.game === "ultimate" ? (
                <Mark player={index + 1} />
              ) : (
                index + 1
              )}
            </span>
            <span class="seat-copy">
              <span class="seat-name">
                <strong title={seat.name || t("free")}>{seat.name || t("free")}</strong>
                {mine && <small>{t("you")}</small>}
              </span>
              <span class="seat-presence" title={detail}>
                {detail}
              </span>
            </span>
            {state.scores?.length ? <strong class="seat-score">{state.scores[index]}</strong> : null}
            {preparing && seat.ready ? (
              <ActionIcon name="check" />
            ) : selectable && seat.taken ? (
              <ActionIcon name="switch" />
            ) : null}
          </button>
        );
      })}
    </fieldset>
  );
}
