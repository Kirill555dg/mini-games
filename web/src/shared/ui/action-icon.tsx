export type IconName =
  | "back"
  | "copy"
  | "link"
  | "book"
  | "plus"
  | "arrow"
  | "check"
  | "users"
  | "search"
  | "rotate"
  | "shuffle"
  | "trash"
  | "close"
  | "external"
  | "clock"
  | "chevron"
  | "sun"
  | "moon"
  | "eye"
  | "grid"
  | "ship"
  | "switch";

export function ActionIcon({ name, className = "" }: { name: IconName; className?: string }) {
  return (
    <span
      class={`action-icon ${className}`}
      aria-hidden="true"
      style={{ maskImage: `url("/icons/${name}.svg")` }}
    />
  );
}
