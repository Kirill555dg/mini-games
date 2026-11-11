import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "preact/hooks";
import { BOARD_LAYOUT } from "@/entities/game/board-layout";
import { GameBoard, gameTitle } from "@/entities/game/game-board";
import { GameRules } from "@/entities/game/game-rules";
import { RoomSeats } from "@/entities/room/room-seats";
import { api, type Command, type GameSpec, type RoomView, unwrap } from "@/shared/api/game-api";
import { live } from "@/shared/api/live-connection";
import { GAME_STATE, ROOM_ACTION } from "@/shared/api/protocol.generated";
import { sideName, t } from "@/shared/i18n/i18n";
import { invitation } from "@/shared/invitations/invitations";
import { ActionIcon } from "@/shared/ui/action-icon";
import { CopyControl } from "@/shared/ui/copy-control";
import { Button, ConfirmLeave, Notice } from "@/shared/ui/ui-controls";
import { NavalGame } from "@/widgets/naval/naval-game";
import "@/pages/room/room-page.css";

export function RoomPage({
  id,
  catalog,
  connection,
}: {
  id: string;
  catalog: GameSpec[];
  connection: (online: boolean) => void;
}) {
  const [current, setCurrent] = useState<RoomView>(),
    [closed, setClosed] = useState(false),
    [online, setOnline] = useState(false),
    [busy, setBusy] = useState(false);
  const [error, setError] = useState(""),
    [confirm, setConfirm] = useState(false);
  const inFlight = useRef(false),
    alive = useRef(true),
    stage = useRef<HTMLDivElement>(null),
    controls = useRef<HTMLDivElement>(null);
  const accept = useCallback((value: RoomView | null) => {
    if (!value) {
      setClosed(true);
      return;
    }
    setCurrent((previous) =>
      !previous || value.instance !== previous.instance || value.revision >= previous.revision
        ? value
        : previous,
    );
  }, []);
  useEffect(() => {
    alive.current = true;
    const stop = live<RoomView>(id, accept, (value) => {
      setOnline(value);
      connection(value);
    });
    return () => {
      alive.current = false;
      stop();
    };
  }, [id, accept, connection]);
  async function act(action: Command["action"], values: Partial<Command> = {}) {
    if (inFlight.current) return false;
    inFlight.current = true;
    setBusy(true);
    setError("");
    try {
      const value = await unwrap(
        api.POST("/game-api/action", { body: { action, room: id, round: current?.round, ...values } }),
      );
      if (alive.current) accept(value);
      return true;
    } catch (failure) {
      if (alive.current) setError(failure instanceof Error ? failure.message : t("error503"));
      try {
        const value = await unwrap(api.GET("/game-api/state", { params: { query: { room: id } } }));
        if (alive.current) accept(value);
      } catch {
        /* Live connection reports recovery. */
      }
      return false;
    } finally {
      inFlight.current = false;
      if (alive.current) setBusy(false);
    }
  }
  const fit = useCallback(() => {
    if (!stage.current || !current) return;
    const board = [...stage.current.querySelectorAll<HTMLElement>(".board, .fleet-grid")].find(
      (element) => element.getClientRects().length,
    );
    if (!board) return;
    const ratio = current.state.width / current.state.height;
    const top = board.getBoundingClientRect().top + scrollY;
    const stageRect = stage.current.getBoundingClientRect();
    const controlsRect = controls.current?.getBoundingClientRect();
    const sameColumn =
      controlsRect && controlsRect.right > stageRect.left && controlsRect.left < stageRect.right;
    const reserve = sameColumn
      ? Math.min(controls.current?.offsetHeight || 0, BOARD_LAYOUT.controlsReserve)
      : 0;
    const height = Math.max(
      BOARD_LAYOUT.minSide,
      (window.visualViewport?.height || innerHeight) - top - reserve - BOARD_LAYOUT.viewportGap,
    );
    stage.current.style.setProperty(
      "--board-size",
      `${Math.floor(Math.min(BOARD_LAYOUT.maxWidth, stage.current.clientWidth, height * ratio))}px`,
    );
  }, [current]);
  useLayoutEffect(fit);
  useEffect(() => {
    const resize = () => requestAnimationFrame(fit);
    window.addEventListener("resize", resize);
    window.visualViewport?.addEventListener("resize", resize);
    return () => {
      window.removeEventListener("resize", resize);
      window.visualViewport?.removeEventListener("resize", resize);
    };
  }, [fit]);
  async function leave() {
    setConfirm(false);
    const left = !current?.name || (await act(ROOM_ACTION.leave));
    if (alive.current && left) location.hash = "/";
  }
  function requestLeave() {
    if (current && current.role >= 0 && current.room.started && !current.room.finished) setConfirm(true);
    else void leave();
  }
  if (closed)
    return (
      <section class="room-page">
        <Notice>{t("roomClosed")}</Notice>
        {current?.room.nextRoom && (
          <a
            class="button primary"
            href={invitation(current.room.nextRoom, { watch: current.role < 0, seat: current.role })}
          >
            {t("continue")}
          </a>
        )}
        <a class="button" href="/#/">
          <ActionIcon name="back" />
          {t("backGames")}
        </a>
      </section>
    );
  if (!current)
    return (
      <section class="room-page">
        <Button
          onClick={() => {
            location.hash = "/";
          }}
        >
          <ActionIcon name="back" />
          {t("backGames")}
        </Button>
        <Notice>{t("loading")}</Notice>
      </section>
    );
  const { room, state, role } = current,
    player = role >= 0,
    spec = catalog.find((game) => game.id === room.game),
    naval = room.game === "battleship";
  let text: string;
  if (room.finished)
    text =
      current.reason && state.winner === GAME_STATE.draw
        ? t("interrupted")
        : state.winner === GAME_STATE.draw
          ? t("draw")
          : state.winner === role
            ? t("victory")
            : t("winner", { name: room.seats[state.winner]?.name || t("player", { n: state.winner + 1 }) });
  else if (!room.started)
    text = room.seats.every((seat) => seat.taken)
      ? t("waitingReady")
      : t("waitingPlayers", { n: room.seats.filter((seat) => seat.taken).length, total: room.seats.length });
  else if (state.phase === GAME_STATE.setup)
    text = player ? t(state.ready?.[role] ? "fleetReady" : "placeFleet") : t("fleetsSetup");
  else text = t(role === state.turn ? "yourTurn" : "turn", { name: room.seats[state.turn]?.name || "" });
  const canMove =
    online &&
    !busy &&
    player &&
    room.started &&
    !room.finished &&
    (state.phase === GAME_STATE.setup || state.turn === role);
  const goal =
    room.game === "ultimate"
      ? t("ultimateGoal")
      : state.winLength
        ? t("lineGoal", { width: state.width, height: state.height, n: state.winLength })
        : t(room.game === "dotsboxes" ? "dotsGoal" : "navalGoal");
  return (
    <section class={`room-page ${naval ? "naval-room" : ""}`}>
      <header class="room-toolbar">
        <Button
          class="room-back icon-button"
          disabled={busy}
          onClick={requestLeave}
          aria-label={t("backGames")}
          title={t("backGames")}
        >
          <ActionIcon name="back" />
        </Button>
        <h1 title={gameTitle(catalog, room.game)}>{gameTitle(catalog, room.game)}</h1>
        <span class="room-goal" title={goal}>
          {goal}
        </span>
      </header>
      <fieldset class="room-meta" aria-label={t("roomInvitations")}>
        <CopyControl value={id} label={t("copyCode")} className="room-code-copy">
          <span class="room-code" translate={false}>
            {id}
          </span>
        </CopyControl>
        <CopyControl value={invitation(id)} label={t("copyLink")} icon="link">
          {t("invite")}
        </CopyControl>
        <CopyControl
          value={invitation(id, { watch: true })}
          label={t("inviteWatch")}
          className="spectator-invite"
          icon="eye"
        >
          <span title={t("spectators", { n: room.spectators })}>{room.spectators}</span>
        </CopyControl>
      </fieldset>
      <div class="room-matchbar">
        <p
          class={`game-status ${online && player && role === state.turn && room.started && !room.finished ? "your-turn" : ""} ${room.finished ? "finished" : ""}`}
          role="status"
        >
          <span class="turn-dot" />
          <span>
            {text}
            {!online && ` · ${t("reconnecting")}`}
          </span>
        </p>
        {!room.started && !room.finished && player && (
          <Button
            class={`small ${room.seats[role].ready ? "" : "primary"}`}
            disabled={busy || !online || Boolean(room.swap)}
            onClick={() => void act(ROOM_ACTION.ready)}
          >
            <ActionIcon name={room.seats[role].ready ? "close" : "check"} />
            {t(room.seats[role].ready ? "notReady" : "ready")}
          </Button>
        )}
      </div>
      {error && <Notice error>{error}</Notice>}
      {room.nextRoom && (
        <Notice>
          <a
            class="text-link"
            href={invitation(room.nextRoom, { watch: !player, seat: player ? role : undefined })}
          >
            {t("continue")} · {room.nextRoom}
          </a>
        </Notice>
      )}
      <div ref={stage} class="game-stage">
        {naval ? (
          <NavalGame
            key={`${id}:${current.round}:${role}`}
            prepared={room.started || room.finished}
            state={state}
            actor={role}
            enabled={canMove}
            playerNames={room.seats.map((seat) => seat.name)}
            onAction={(values) => act(ROOM_ACTION.move, values)}
          />
        ) : (
          <GameBoard
            game={room.game}
            state={state}
            actor={role}
            enabled={canMove}
            onMove={(move) => void act(ROOM_ACTION.move, move)}
          />
        )}
      </div>
      <div ref={controls} class="game-controls">
        <RoomSeats
          room={room}
          state={state}
          actor={role}
          spec={spec}
          enabled={online && !busy && !room.swap}
          choose={(seat) => void act(ROOM_ACTION.seat, { seat })}
        />
        {room.swap && (
          <div class="swap-request" role="status">
            <ActionIcon name="switch" />
            <p>
              {t("swapRequest", { from: sideName(spec, room.swap.from), to: sideName(spec, room.swap.to) })}
            </p>
            {player && (role === room.swap.from || role === room.swap.to) && (
              <span class="actions">
                {role === room.swap.to && (
                  <Button
                    class="small primary"
                    disabled={busy || !online}
                    onClick={() => void act(ROOM_ACTION.acceptSwap)}
                  >
                    <ActionIcon name="check" />
                    {t("acceptSwap")}
                  </Button>
                )}
                <Button
                  class="icon-button small"
                  aria-label={t("declineSwap")}
                  title={t("declineSwap")}
                  disabled={busy || !online}
                  onClick={() => void act(ROOM_ACTION.cancelSwap)}
                >
                  <ActionIcon name="close" />
                </Button>
              </span>
            )}
          </div>
        )}
        {room.finished && (
          <div class="finished-actions">
            {player && (
              <a class="button primary" href={`/#/new/${room.game}?from=${id}`}>
                <ActionIcon name="plus" />
                {t("newGame")}
              </a>
            )}
            <a class="button" href={`/#/history?code=${id}`}>
              <ActionIcon name="clock" />
              {t("history")}
            </a>
          </div>
        )}
        {!player && !current.name && (
          <a class="button primary" href={invitation(id, { watch: true })}>
            {t("continue")}
          </a>
        )}
        {!player &&
          current.name &&
          !room.started &&
          !room.finished &&
          room.seats.some((seat) => !seat.taken) && (
            <Button
              class="primary"
              disabled={busy || !online}
              onClick={() => void act(ROOM_ACTION.enter, { name: current.name })}
            >
              {t("play")}
            </Button>
          )}
      </div>
      <div class="room-secondary">
        <GameRules game={spec} />
      </div>
      {confirm && <ConfirmLeave code={id} cancel={() => setConfirm(false)} confirm={() => void leave()} />}
    </section>
  );
}
