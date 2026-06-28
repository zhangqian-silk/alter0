import { useCallback, useEffect, useRef, type ComponentPropsWithoutRef, type PointerEvent, type TouchEvent } from "react";
import { runWithKeyboardDismissal } from "./runtimeKeyboardDismissal";

export type RuntimeMobilePressButtonProps = Omit<
  ComponentPropsWithoutRef<"button">,
  "type" | "className" | "children" | "onClick"
>;

type UseRuntimeMobilePressActionOptions = {
  suppressClickMs: number;
};

export function useRuntimeMobilePressAction({ suppressClickMs }: UseRuntimeMobilePressActionOptions) {
  const locksRef = useRef<Record<string, boolean>>({});
  const timersRef = useRef<Record<string, number | null>>({});

  const release = useCallback((key: string) => {
    locksRef.current[key] = false;
    const timer = timersRef.current[key];
    if (timer !== null && typeof timer !== "undefined") {
      window.clearTimeout(timer);
      timersRef.current[key] = null;
    }
  }, []);

  const triggerFromPress = useCallback((key: string, action: (() => void) | undefined) => {
    if (!action || locksRef.current[key]) {
      return;
    }
    locksRef.current[key] = true;
    const existingTimer = timersRef.current[key];
    if (existingTimer !== null && typeof existingTimer !== "undefined") {
      window.clearTimeout(existingTimer);
    }
    timersRef.current[key] = window.setTimeout(() => {
      release(key);
    }, suppressClickMs);
    runWithKeyboardDismissal(action);
  }, [release, suppressClickMs]);

  const triggerFromClick = useCallback((key: string, action: (() => void) | undefined) => {
    if (locksRef.current[key]) {
      release(key);
      return;
    }
    runWithKeyboardDismissal(action);
  }, [release]);

  const createPressHandlers = useCallback((
    key: string,
    action: (() => void) | undefined,
    props: RuntimeMobilePressButtonProps | undefined,
  ) => {
    const {
      onPointerDownCapture,
      onTouchStartCapture,
      ...restProps
    } = props || {};
    return {
      ...restProps,
      onPointerDownCapture: (event: PointerEvent<HTMLButtonElement>) => {
        onPointerDownCapture?.(event);
        if (event.defaultPrevented || event.pointerType === "mouse") {
          return;
        }
        event.preventDefault();
        triggerFromPress(key, action);
      },
      onTouchStartCapture: (event: TouchEvent<HTMLButtonElement>) => {
        onTouchStartCapture?.(event);
        if (event.defaultPrevented) {
          return;
        }
        event.preventDefault();
        triggerFromPress(key, action);
      },
    };
  }, [triggerFromPress]);

  useEffect(() => () => {
    for (const timer of Object.values(timersRef.current)) {
      if (timer !== null && typeof timer !== "undefined") {
        window.clearTimeout(timer);
      }
    }
  }, []);

  return {
    createPressHandlers,
    triggerFromClick,
  };
}
