import { gameTitle } from "@/entities/game/game-board";
import type { Lobby } from "@/shared/api/game-api";
import { t } from "@/shared/i18n/i18n";
import { invitation } from "@/shared/invitations/invitations";

export function RoomList({ lobby }: { lobby: Lobby }) {
  return (
    <section class="room-list" aria-labelledby="rooms-heading">
      <div class="section-heading">
        <h2 id="rooms-heading">{t("rooms")}</h2>
        <span class="muted">{lobby.totalRooms}</span>
      </div>
      {!lobby.rooms.length && (
        <div class="empty-state">
          <strong>{t("quiet")}</strong>
          <p>{t("quietHint")}</p>
        </div>
      )}
      {lobby.rooms.map((room) => (
        <article class="room-row" key={room.id}>
          <div class="room-code-group">
            <span class="room-code">{room.id}</span>
            {!room.finished && room.seats.some((seat) => !seat.taken) && (
              <a class="button small primary" href={invitation(room.id)}>
                {t("play")}
              </a>
            )}
            <a class="button small" href={invitation(room.id, { watch: true })}>
              {t("watch")}
            </a>
          </div>
          <div class="room-description">
            <strong>{gameTitle(lobby.games, room.game)}</strong>
            <span class="muted">{room.seats.map((seat) => seat.name || t("free")).join(" · ")}</span>
          </div>
          <span class={`badge ${!room.started && !room.finished ? "available" : ""}`}>
            {room.finished
              ? t("ended")
              : room.started
                ? t("playing")
                : t("places", {
                    n: room.seats.filter((seat) => seat.taken).length,
                    total: room.seats.length,
                  })}
          </span>
        </article>
      ))}
    </section>
  );
}
