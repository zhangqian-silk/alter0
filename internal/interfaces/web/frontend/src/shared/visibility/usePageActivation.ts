import { useEffect, useRef } from "react";

type UsePageActivationOptions = {
  debounceMs?: number;
  onActive?: () => void | Promise<void>;
  onVisibilityChange?: (hidden: boolean) => void;
};

export function usePageActivation(options: UsePageActivationOptions) {
  const onActiveRef = useRef(options.onActive);
  const onVisibilityChangeRef = useRef(options.onVisibilityChange);
  const lastActiveAtRef = useRef(0);

  onActiveRef.current = options.onActive;
  onVisibilityChangeRef.current = options.onVisibilityChange;

  useEffect(() => {
    if (typeof document === "undefined" || typeof window === "undefined") {
      return;
    }

    const debounceMs = Math.max(0, Number(options.debounceMs) || 0);
    const triggerActive = () => {
      if (document.visibilityState === "hidden") {
        return;
      }
      const now = Date.now();
      if (now - lastActiveAtRef.current < debounceMs) {
        return;
      }
      lastActiveAtRef.current = now;
      void onActiveRef.current?.();
    };

    const handleVisibilityChange = () => {
      onVisibilityChangeRef.current?.(document.hidden);
      if (document.visibilityState !== "visible") {
        return;
      }
      triggerActive();
    };

    window.addEventListener("focus", triggerActive);
    window.addEventListener("pageshow", triggerActive);
    window.addEventListener("online", triggerActive);
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      window.removeEventListener("focus", triggerActive);
      window.removeEventListener("pageshow", triggerActive);
      window.removeEventListener("online", triggerActive);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [options.debounceMs]);
}
