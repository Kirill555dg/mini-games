import type { ComponentChildren, JSX } from "preact";
import { useEffect, useRef } from "preact/hooks";
import { t } from "@/shared/i18n/i18n";

export function Button(props: JSX.IntrinsicElements["button"]) {
  return (
    <button type="button" {...props} class={`button ${props.class || ""}`}>
      {props.children}
    </button>
  );
}
export function Notice({ children, error = false }: { children: ComponentChildren; error?: boolean }) {
  return (
    <p class={`notice ${error ? "error" : ""}`} role={error ? "alert" : "status"}>
      {children}
    </p>
  );
}
export function Heading({ title, hint }: { title: string; hint?: string }) {
  return (
    <div class="page-heading">
      <h1>{title}</h1>
      {hint && <p class="muted">{hint}</p>}
    </div>
  );
}
export function Field({
  label,
  id,
  children,
  hint,
}: {
  label: string;
  id: string;
  children: ComponentChildren;
  hint?: string;
}) {
  return (
    <div class="field">
      <label for={id}>{label}</label>
      {children}
      {hint && <p class="field-help">{hint}</p>}
    </div>
  );
}
export function Mark({ player }: { player: number }) {
  if (!player) return null;
  return (
    <svg viewBox="0 0 40 40" class={`mark player-${player}`} aria-hidden="true">
      {player === 1 ? <path d="M8 8 32 32M32 8 8 32" /> : <circle cx="20" cy="20" r="12" />}
    </svg>
  );
}
export function ConfirmLeave({
  code,
  cancel,
  confirm,
}: {
  code: string;
  cancel: () => void;
  confirm: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    ref.current?.showModal();
    return () => ref.current?.close();
  }, []);
  return (
    <dialog
      ref={ref}
      class="confirm-dialog"
      aria-labelledby="leave-title"
      onCancel={(event) => {
        event.preventDefault();
        cancel();
      }}
    >
      <button type="button" class="dialog-close" aria-label={t("close")} onClick={cancel}>
        ×
      </button>
      <h2 id="leave-title">{t("leaveTitle", { code })}</h2>
      <p>{t("leaveHint")}</p>
      <div class="actions">
        <Button onClick={cancel} autoFocus>
          {t("stay")}
        </Button>
        <Button class="danger" onClick={confirm}>
          {t("leave")}
        </Button>
      </div>
    </dialog>
  );
}
