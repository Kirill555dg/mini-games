import { t } from "@/shared/i18n/i18n";
import { Button } from "@/shared/ui/ui-controls";

export function MoveReview({
  index,
  total,
  change,
}: {
  index: number;
  total: number;
  change: (index: number) => void;
}) {
  return (
    <fieldset class="replay-controls" aria-label={t("review")}>
      <Button
        class="icon-button"
        disabled={index === 0}
        onClick={() => change(index - 1)}
        aria-label={t("previous")}
      >
        ←
      </Button>
      <span class="replay-position" aria-live="polite">
        {t("moveCount", { n: index, total })}
      </span>
      <Button
        class="icon-button"
        disabled={index === total}
        onClick={() => change(index + 1)}
        aria-label={t("next")}
      >
        →
      </Button>
    </fieldset>
  );
}
