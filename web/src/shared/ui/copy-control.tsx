import type { ComponentChildren } from "preact";
import { useRef, useState } from "preact/hooks";
import { t } from "@/shared/i18n/i18n";
import { ActionIcon } from "@/shared/ui/action-icon";
import { Button } from "@/shared/ui/ui-controls";

export function CopyControl({
  value,
  label,
  className = "",
  children,
}: {
  value: string;
  label: string;
  className?: string;
  children?: ComponentChildren;
}) {
  const [copied, setCopied] = useState(false),
    [fallback, setFallback] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  return (
    <span class={`copy-control ${className}`}>
      <Button
        class={copied ? "copy-success small" : "small"}
        aria-label={copied ? t("copyDone") : label}
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(value);
            setCopied(true);
            setFallback(false);
          } catch {
            setCopied(false);
            setFallback(true);
            requestAnimationFrame(() => {
              input.current?.focus();
              input.current?.select();
            });
          }
        }}
      >
        <ActionIcon name={copied ? "check" : "copy"} />
        {children || (copied ? t("copyDone") : label)}
      </Button>
      {copied && (
        <span class="sr-only" role="status">
          {t("copyDone")}
        </span>
      )}
      {fallback && (
        <span class="copy-fallback">
          <label>
            {t("copyManually")}
            <input ref={input} value={value} readOnly onFocus={(event) => event.currentTarget.select()} />
          </label>
          <Button class="small" aria-label={t("close")} onClick={() => setFallback(false)}>
            <ActionIcon name="close" />
          </Button>
        </span>
      )}
    </span>
  );
}
