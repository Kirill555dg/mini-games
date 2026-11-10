import { type ComponentChildren, render } from "preact";
import { useEffect, useState } from "preact/hooks";
import { PROJECT } from "@/shared/config/project-info";
import "@/app/styles.css";
import "@/app/base.css";
import "@/app/layout.css";
import "@/app/components.css";
import "@/entities/game/game-board.css";
import "@/app/responsive.css";
import { RoomEntry } from "@/features/entry/room-entry";
import { HistoryPage } from "@/pages/history/history-page";
import { HomePage } from "@/pages/home/home-page";
import { RoomPage } from "@/pages/room/room-page";
import { GameRulesPage } from "@/pages/rules/game-rules-page";
import { StatusPage } from "@/pages/status/status-page";
import { api, type Lobby, type RoomView, unwrap } from "@/shared/api/game-api";
import { live } from "@/shared/api/live-connection";
import { language, setLanguage, t } from "@/shared/i18n/i18n";
import { normalizeCode, parseInvitation } from "@/shared/invitations/invitations";
import { ActionIcon } from "@/shared/ui/action-icon";
import { Button, Notice } from "@/shared/ui/ui-controls";

function routePath() {
  return location.hash.slice(1) || (location.pathname === "/status.html" ? "/status" : "/");
}
function initialTheme() {
  try {
    return (
      localStorage.getItem("mini-games-theme") ||
      (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light")
    );
  } catch {
    return "light";
  }
}
function App() {
  const [route, setRoute] = useState(routePath),
    [locale, setLocale] = useState(language),
    [theme, setTheme] = useState(initialTheme),
    [online, setOnline] = useState(false);
  const [lobby, setLobby] = useState<Lobby>(),
    [error, setError] = useState(""),
    [preset, setPreset] = useState<RoomView>(),
    [loadedRoute, setLoadedRoute] = useState("");
  const [path, queryString] = route.split("?"),
    [page = "home", id = ""] = path.split("/").filter(Boolean),
    query = new URLSearchParams(queryString);
  const room = page === "room";
  useEffect(() => {
    const change = () => setRoute(routePath());
    window.addEventListener("hashchange", change);
    return () => window.removeEventListener("hashchange", change);
  }, []);
  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);
  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    try {
      localStorage.setItem("mini-games-theme", theme);
    } catch {
      /* Storage is optional. */
    }
  }, [theme]);
  useEffect(() => {
    document.body.classList.toggle("in-room", room);
    document.title = `${t(["home", "history", "status"].includes(page) ? (page === "home" ? "games" : page) : room ? "room" : "app")} · ${t("app")}`;
  }, [page, room, locale]);
  useEffect(() => {
    let alive = true;
    const controller = new AbortController();
    setError("");
    setPreset(undefined);
    if (page === "status") return;
    const from = query.get("from");
    Promise.all([
      unwrap(api.GET("/game-api/lobby", { signal: controller.signal })),
      unwrap(api.POST("/game-api/visit", { signal: controller.signal })).catch(() => undefined),
      page === "new" && from
        ? unwrap(api.GET("/game-api/state", { params: { query: { room: from } }, signal: controller.signal }))
        : Promise.resolve(undefined),
    ])
      .then(([data, , previous]) => {
        if (alive) {
          setLobby(data);
          setPreset(previous);
          setLoadedRoute(route);
          setOnline(true);
        }
      })
      .catch((failure) => {
        if (alive) {
          setError(failure.message);
          setOnline(false);
        }
      });
    return () => {
      alive = false;
      controller.abort();
    };
  }, [route]);
  const invited = parseInvitation(location.href);
  const spec = lobby?.games.find((game) => game.id === id);
  let content: ComponentChildren = null;
  if (page === "status") content = <StatusPage connection={setOnline} />;
  else if (error)
    content = (
      <>
        <Notice error>{error}</Notice>
        <Button class="primary" onClick={() => location.reload()}>
          {t("retry")}
        </Button>{" "}
        <a class="button" href="/#/">
          {t("backGames")}
        </a>
      </>
    );
  else if (!lobby || loadedRoute !== route) content = <Notice>{t("loading")}</Notice>;
  else if (page === "home" || page === "join")
    content = (
      <HomeLive
        key={page === "join" ? route : "home"}
        lobby={lobby}
        connection={setOnline}
        invite={page === "join" ? invited : undefined}
      />
    );
  else if (page === "room" && normalizeCode(id))
    content = <RoomPage key={id} id={normalizeCode(id)} catalog={lobby.games} connection={setOnline} />;
  else if (page === "new" && spec)
    content = (
      <RoomEntry
        key={route}
        catalog={lobby.games}
        game={spec}
        from={query.get("from") || ""}
        preferredSeat={preset?.role}
        preset={preset ? { size: preset.room.size, players: preset.room.seats.length } : undefined}
      />
    );
  else if (page === "history") content = <HistoryPage catalog={lobby.games} />;
  else if (page === "rules" && spec) content = <GameRulesPage game={spec} />;
  else content = <Notice>{t("notFound")}</Notice>;
  const brand = (
    <>
      <img src="/favicon.svg" width="28" height="28" alt="" />
      {t("app")}
    </>
  );
  return (
    <>
      <button
        type="button"
        class="skip-link"
        onClick={(event) => {
          event.preventDefault();
          document.getElementById("content")?.focus();
        }}
      >
        {t("games")}
      </button>
      <header class="site-header">
        {room ? (
          <span class="brand">{brand}</span>
        ) : (
          <a href="/#/" class="brand">
            {brand}
          </a>
        )}
        {!room && (
          <nav aria-label={t("games")}>
            {["home", "history", "status"].map((item) => (
              <a
                key={item}
                class="nav-link"
                href={item === "home" ? "/#/" : `/#/${item}`}
                aria-current={(page === "join" ? "home" : page) === item ? "page" : undefined}
              >
                {t(item === "home" ? "games" : item)}
              </a>
            ))}
          </nav>
        )}
        <div class="preferences">
          <fieldset class="language-control" aria-label={t("language")}>
            {["ru", "en"].map((value) => (
              <button
                type="button"
                key={value}
                class="language-button"
                aria-pressed={locale === value}
                aria-label={value === "ru" ? "Русский" : "English"}
                onClick={() => {
                  setLanguage(value);
                  setLocale(language());
                }}
              >
                <img src={`/language-${value}.svg`} width="20" height="14" alt="" />
                <span>{value.toUpperCase()}</span>
              </button>
            ))}
          </fieldset>
          <button
            type="button"
            class="theme-toggle"
            aria-label={t(theme === "dark" ? "lightTheme" : "darkTheme")}
            title={t(theme === "dark" ? "lightTheme" : "darkTheme")}
            onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          >
            <ActionIcon name={theme === "dark" ? "sun" : "moon"} />
          </button>
        </div>
        <span class={`connection ${online ? "online" : "offline"}`} role="status">
          {t(online ? "online" : "offline")}
        </span>
      </header>
      <main id="content" tabIndex={-1} key={page}>
        {content}
      </main>
      <footer class="site-footer">
        {PROJECT.author} · {PROJECT.year} ·{" "}
        <a class="footer-link" href={PROJECT.repository}>
          GitHub
          <ActionIcon name="external" />
        </a>{" "}
        ·{" "}
        <a class="footer-link" href={`${PROJECT.repository}/blob/main/LICENSE`}>
          MIT
        </a>
      </footer>
    </>
  );
}
function HomeLive({
  lobby,
  connection,
  invite,
}: {
  lobby: Lobby;
  connection: (online: boolean) => void;
  invite?: ReturnType<typeof parseInvitation>;
}) {
  const [value, setValue] = useState(lobby);
  useEffect(
    () =>
      live<Lobby>(
        "",
        (next) => {
          if (next) setValue(next);
        },
        connection,
      ),
    [connection],
  );
  return <HomePage lobby={value} invite={invite} />;
}
const root = document.getElementById("app");
if (root) {
  root.replaceChildren();
  render(<App />, root);
}
