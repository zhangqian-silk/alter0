import type { ComponentPropsWithoutRef, ReactNode, Ref } from "react";

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

type RuntimeWorkspaceScreenProps = {
  panelClassName?: string;
  panelProps?: Omit<ComponentPropsWithoutRef<"section">, "children" | "className"> & {
    className?: string;
  };
  screenClassName?: string;
  screenProps?: Omit<ComponentPropsWithoutRef<"div">, "children" | "className"> & {
    className?: string;
  };
  screenRef?: Ref<HTMLDivElement>;
  children: ReactNode;
};

export function RuntimeWorkspaceScreen({
  panelClassName,
  panelProps,
  screenClassName,
  screenProps,
  screenRef,
  children,
}: RuntimeWorkspaceScreenProps) {
  const {
    className: panelPropsClassName,
    ...panelRestProps
  } = panelProps || {};
  const {
    className: screenPropsClassName,
    ...screenRestProps
  } = screenProps || {};

  return (
    <section
      className={joinClassNames("runtime-workspace-panel", panelClassName, panelPropsClassName)}
      {...panelRestProps}
    >
      <div
        ref={screenRef}
        className={joinClassNames("runtime-workspace-screen", screenClassName, screenPropsClassName)}
        {...screenRestProps}
      >
        {children}
      </div>
    </section>
  );
}
