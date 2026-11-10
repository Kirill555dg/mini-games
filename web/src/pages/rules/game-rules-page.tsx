import type { GameSpec } from "@/shared/api/game-api";
import { t, translated } from "@/shared/i18n/i18n";
import { ActionIcon } from "@/shared/ui/action-icon";
import { Heading } from "@/shared/ui/ui-controls";

export function GameRulesPage({ game }: { game: GameSpec }) {
  return (
    <section class="reading-page">
      <a href="/#/" class="button back-button">
        <ActionIcon name="back" />
        {t("backGames")}
      </a>
      <Heading title={`${translated(game, "title")} · ${t("rules")}`} />
      <section class="rule-section">
        <p>{translated(game, "rules")}</p>
      </section>
      <section class="rule-section">
        <h2>{t("generalRulesTitle")}</h2>
        <p>{t("generalRules")}</p>
        <p>{t("connectionRules")}</p>
        <p>{t("privacyRules")}</p>
      </section>
    </section>
  );
}
