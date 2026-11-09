import type { Lobby } from "@/shared/api/game-api";
import { t, translated } from "@/shared/i18n/i18n";
import { Heading } from "@/shared/ui/ui-controls";
import { RoomList } from "@/widgets/rooms/room-list";

export function HomePage({ lobby }: { lobby: Lobby }) {
  return (
    <section class="home-page">
      <div class="home-heading">
        <Heading title={t("chooseGame")} hint={t("homeHint")} />
        <a class="button primary join-primary" href="/#/join">
          {t("joinCode")}
        </a>
      </div>
      <div class="game-grid">
        {lobby.games.map((game) => (
          <article class="game-card" key={game.id}>
            <img class="game-illustration" src={`/${game.id}.svg`} alt="" width="160" height="132" />
            <div class="game-card-content">
              <div class="game-card-title">
                <h2>{translated(game, "title")}</h2>
                <span class="player-count">
                  {t("players", {
                    n:
                      game.minPlayers === game.maxPlayers
                        ? game.minPlayers
                        : `${game.minPlayers}-${game.maxPlayers}`,
                  })}
                </span>
              </div>
              <p>{translated(game, "description")}</p>
              <div class="actions">
                <a class="button" href={`/#/new/${game.id}`}>
                  {t("create")}
                </a>
                <a class="text-link" href={`/#/rules/${game.id}`}>
                  {t("rules")}
                </a>
              </div>
            </div>
          </article>
        ))}
      </div>
      <RoomList lobby={lobby} />
    </section>
  );
}
