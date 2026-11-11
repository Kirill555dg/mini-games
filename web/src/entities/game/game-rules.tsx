import { useId, useRef, useState } from "preact/hooks";
import type { GameSpec } from "@/shared/api/game-api";
import { t, translated } from "@/shared/i18n/i18n";
import { ActionIcon } from "@/shared/ui/action-icon";
import { Button } from "@/shared/ui/ui-controls";
import "@/entities/game/game-rules.css";

export function GameRules({ game, className = "" }: { game?: GameSpec; className?: string }) {
  const [open, setOpen] = useState(false),
    id = useId(),
    container = useRef<HTMLDivElement>(null);
  return (
    <div ref={container} class={`game-rules ${className}`}>
      <Button class="rules-button" aria-expanded={open} aria-controls={id} onClick={() => setOpen(!open)}>
        <ActionIcon name="book" />
        {t("rules")}
      </Button>
      <div id={id} class="rules-disclosure" hidden={!open}>
        <h3>{translated(game, "title")}</h3>
        <p>{translated(game, "rules")}</p>
        <Button
          class="icon-button rules-close"
          aria-label={t("closeRules")}
          onClick={() => {
            setOpen(false);
            container.current?.querySelector("button")?.focus();
          }}
        >
          <ActionIcon name="close" />
        </Button>
      </div>
    </div>
  );
}
