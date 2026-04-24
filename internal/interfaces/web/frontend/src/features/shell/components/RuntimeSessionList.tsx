import type { ComponentPropsWithoutRef, ReactNode } from "react";

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

export type RuntimeSessionListGroup<TItem> = {
  key: string;
  label: string;
  items: TItem[];
};

type RuntimeSessionListProps<TItem> = {
  groups: Array<RuntimeSessionListGroup<TItem>>;
  emptyState?: ReactNode;
  listClassName?: string;
  listProps?: Omit<ComponentPropsWithoutRef<"div">, "children" | "className"> & {
    className?: string;
  };
  renderItem: (item: TItem, group: RuntimeSessionListGroup<TItem>) => ReactNode;
};

export function RuntimeSessionList<TItem>({
  groups,
  emptyState,
  listClassName,
  listProps,
  renderItem,
}: RuntimeSessionListProps<TItem>) {
  const {
    className: listPropsClassName,
    ...listRestProps
  } = listProps || {};

  return (
    <div
      className={joinClassNames(
        "runtime-session-list",
        "menu",
        listClassName,
        listPropsClassName,
      )}
      role="list"
      {...listRestProps}
    >
      {emptyState}
      {groups.map((group) => (
        <section
          key={group.key}
          className="runtime-session-group menu-group"
          aria-label={group.label}
        >
          <h2 className="runtime-session-group-label">{group.label}</h2>
          <div className="runtime-session-group-items">
            {group.items.map((item) => renderItem(item, group))}
          </div>
        </section>
      ))}
    </div>
  );
}
