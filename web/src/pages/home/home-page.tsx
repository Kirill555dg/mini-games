import { GameRules } from "@/entities/game/game-rules";
import { RoomEntry } from "@/features/entry/room-entry";
import type { Lobby } from "@/shared/api/game-api";
import { t, translated } from "@/shared/i18n/i18n";
import { ActionIcon } from "@/shared/ui/action-icon";
import { Heading } from "@/shared/ui/ui-controls";
import { RoomList } from "@/widgets/rooms/room-list";

export function HomePage({
  lobby,
  invite,
}: {
  lobby: Lobby;
  invite?: { code: string; watch?: boolean; seat?: number };
}) {
  return (
    <section class="home-page">
      <div class="home-intro">
        <Heading title={t("chooseGame")} hint={t("homeHint")} />
        <span class="home-note">
          <ActionIcon name="users" />
          {t("noAccount")}
        </span>
      </div>
      <RoomEntry
        key={invite?.code || "home-join"}
        embedded
        catalog={lobby.games}
        code={invite?.code}
        watch={invite?.watch}
        preferredSeat={invite?.seat}
      />
      <div class="game-grid">
        {lobby.games.map((game) => (
          <article class={`game-card game-${game.id}`} key={game.id}>
            <div class="game-card-main">
              <img class="game-illustration" src={`/${game.id}.svg`} alt="" width="160" height="132" />
              <div class="game-card-content">
                <div class="game-card-title">
                  <h2>{translated(game, "title")}</h2>
                  <span class="player-count">
                    <ActionIcon name="users" />
                    {game.minPlayers === game.maxPlayers
                      ? game.minPlayers
                      : `${game.minPlayers}-${game.maxPlayers}`}
                  </span>
                </div>
                <p>{translated(game, "description")}</p>
                <div class="actions">
                  <a class="button primary" href={`/#/new/${game.id}`}>
                    <ActionIcon name="plus" />
                    {t("createShort")}
                  </a>
                  <GameRules game={game} />
                </div>
              </div>
            </div>
          </article>
        ))}
      </div>
      <RoomList lobby={lobby} />
    </section>
  );
}
