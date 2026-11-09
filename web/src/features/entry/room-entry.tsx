import { useEffect, useRef, useState } from "preact/hooks";
import { api, type GameSpec, type RoomView, unwrap } from "@/shared/api/game-api";
import { live } from "@/shared/api/live-connection";
import { ROOM_ACTION, ROOM_CODE } from "@/shared/api/protocol.generated";
import { sideName, t, translated } from "@/shared/i18n/i18n";
import {
  INVITE_INPUT_MAX_LENGTH,
  invitation,
  normalizeCode,
  parseInvitation,
  savedName,
  saveName,
} from "@/shared/invitations/invitations";
import { Button, Field, Heading, Notice } from "@/shared/ui/ui-controls";

type Props = {
  catalog: GameSpec[];
  game?: GameSpec;
  code?: string;
  watch?: boolean;
  preferredSeat?: number;
  preset?: { size: number; players: number };
  from?: string;
};
export function RoomEntry({
  catalog,
  game,
  code = "",
  watch = false,
  preferredSeat,
  preset,
  from = "",
}: Props) {
  const [name, setName] = useState(savedName),
    [input, setInput] = useState(code),
    [watching, setWatching] = useState(watch);
  const [size, setSize] = useState(preset?.size || game?.sizes?.[0] || 0),
    [players, setPlayers] = useState(preset?.players || game?.minPlayers || 2);
  const [seat, setSeat] = useState(preferredSeat === undefined ? "" : String(preferredSeat));
  const [view, setView] = useState<RoomView | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const alive = useRef(true),
    submitController = useRef<AbortController | null>(null);
  const [nextRoom, setNextRoom] = useState("");
  useEffect(
    () => () => {
      alive.current = false;
      submitController.current?.abort();
    },
    [],
  );
  useEffect(() => {
    if (!from) return;
    return live<RoomView>(
      from,
      (value) => {
        if (alive.current && value?.room.nextRoom) setNextRoom(value.room.nextRoom);
      },
      () => {},
    );
  }, [from]);
  const creating = Boolean(game),
    normalized = normalizeCode(input);
  useEffect(() => {
    if (creating || !normalized) {
      setView(null);
      return;
    }
    let current = true;
    const controller = new AbortController();
    unwrap(api.GET("/game-api/state", { params: { query: { room: normalized } }, signal: controller.signal }))
      .then((value) => {
        if (current) setView(value);
      })
      .catch(() => {
        if (current) setView(null);
      });
    return () => {
      current = false;
      controller.abort();
    };
  }, [creating, normalized]);
  const spec = game || catalog.find((item) => item.id === view?.room.game);
  const capacity = creating ? players : view?.room.seats.length || 2;
  useEffect(() => {
    if (
      !creating &&
      seat !== "" &&
      view &&
      (Number(seat) >= view.room.seats.length || view.room.seats[Number(seat)].taken)
    )
      setSeat("");
  }, [view, seat, creating]);
  const title = game
    ? t("createTitle", { game: translated(game, "title") || "" })
    : t(watching ? "watchTitle" : "enterTitle");
  return (
    <section class="entry-page">
      <a href="/#/" class="breadcrumb">
        {t("games")}
      </a>
      <Heading title={title} />
      <form
        class="entry-form"
        onSubmit={async (event) => {
          event.preventDefault();
          if (busy) return;
          if (!creating && !normalized) {
            setError(t("codeInvalid"));
            return;
          }
          setBusy(true);
          setError("");
          try {
            const chosen = seat === "" ? undefined : Number(seat);
            const controller = new AbortController();
            submitController.current = controller;
            const value = await unwrap(
              api.POST("/game-api/action", {
                signal: controller.signal,
                body: game
                  ? { action: ROOM_ACTION.create, from, name, game: game.id, size, players, seat: chosen }
                  : { action: ROOM_ACTION.enter, name, room: normalized, watch: watching, seat: chosen },
              }),
            );
            if (!alive.current) return;
            saveName(name);
            location.hash = `/room/${value.room.id}`;
          } catch (failure) {
            if (!alive.current) return;
            if (from) {
              try {
                const previous = await unwrap(
                  api.GET("/game-api/state", { params: { query: { room: from } } }),
                );
                if (alive.current && previous.room.nextRoom) setNextRoom(previous.room.nextRoom);
              } catch {
                /* Preserve the original error. */
              }
            }
            if (!alive.current) return;
            setError(failure instanceof Error ? failure.message : t("error503"));
          } finally {
            if (alive.current) setBusy(false);
          }
        }}
      >
        <Field label={t("yourName")} id="entry-name" hint={t("nameHint")}>
          <input
            id="entry-name"
            name="name"
            value={name}
            required
            maxLength={ROOM_CODE.maxNameRunes}
            autoComplete="nickname"
            onInput={(event) => setName(event.currentTarget.value)}
          />
        </Field>
        {!creating && (
          <>
            <Field label={t("code")} id="entry-code" hint={t("codeHint")}>
              <input
                id="entry-code"
                name="code"
                value={input}
                required
                maxLength={INVITE_INPUT_MAX_LENGTH}
                autoComplete="off"
                autoCapitalize="characters"
                spellcheck={false}
                placeholder="K7M2QD"
                onInput={(event) => {
                  const value = event.currentTarget.value;
                  setInput(value);
                  const pasted = parseInvitation(value);
                  if (value.includes("/")) {
                    setWatching(pasted.watch || false);
                    setSeat(pasted.seat === undefined ? "" : String(pasted.seat));
                  } else setSeat("");
                }}
              />
            </Field>
            <fieldset class="segmented">
              <legend class="sr-only">{t("play")}</legend>
              {[false, true].map((value) => (
                <label key={String(value)}>
                  <input
                    type="radio"
                    name="mode"
                    checked={watching === value}
                    onChange={() => setWatching(value)}
                  />
                  <span>{t(value ? "watch" : "play")}</span>
                </label>
              ))}
            </fieldset>
          </>
        )}
        {game?.sizes && (
          <Field label={t("boardSize")} id="entry-size">
            <select
              id="entry-size"
              name="size"
              value={size}
              onChange={(event) => setSize(Number(event.currentTarget.value))}
            >
              {game.sizes.map((value) => (
                <option key={value} value={value}>
                  {value} × {value}
                </option>
              ))}
            </select>
          </Field>
        )}
        {game && game.maxPlayers > game.minPlayers && (
          <Field label={t("playerCount")} id="entry-players">
            <select
              id="entry-players"
              name="players"
              value={players}
              onChange={(event) => {
                setPlayers(Number(event.currentTarget.value));
                setSeat("");
              }}
            >
              {Array.from(
                { length: game.maxPlayers - game.minPlayers + 1 },
                (_, index) => index + game.minPlayers,
              ).map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
          </Field>
        )}
        {(creating || !watching) && (
          <Field label={t("seat")} id="entry-seat">
            <select
              id="entry-seat"
              name="seat"
              value={seat}
              onChange={(event) => setSeat(event.currentTarget.value)}
            >
              <option value="">{t("anySeat")}</option>
              {Array.from({ length: capacity }, (_, index) => (
                <option key={index} value={index} disabled={!creating && view?.room.seats[index].taken}>
                  {sideName(spec, index)}
                  {!creating && view?.room.seats[index].taken ? ` · ${t("occupied")}` : ""}
                </option>
              ))}
            </select>
          </Field>
        )}
        {game && (
          <>
            <p class="field-help">{t(from ? "newGameHint" : "settingsHint")}</p>
            <details class="entry-rules">
              <summary>{t("rules")}</summary>
              <p>{translated(game, "rules")}</p>
            </details>
          </>
        )}
        {error && <Notice error>{error}</Notice>}
        {nextRoom && (
          <Notice>
            <a class="text-link" href={invitation(nextRoom, { seat: preferredSeat })}>
              {t("continue")} · {nextRoom}
            </a>
          </Notice>
        )}
        <div class="form-actions">
          <Button type="submit" class="primary" disabled={busy}>
            {t(busy ? (creating ? "creating" : "entering") : creating ? "create" : "continue")}
          </Button>
          <a class="button" href={from ? `/#/room/${from}` : "/#/"}>
            {t("cancel")}
          </a>
        </div>
      </form>
    </section>
  );
}
