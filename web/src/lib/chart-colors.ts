import { useMemo, useSyncExternalStore } from "react";

// Subscribe to theme changes by watching the <html> class attribute
function subscribeToTheme(callback: () => void) {
  const observer = new MutationObserver(callback);
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["class"],
  });
  return () => observer.disconnect();
}

function getThemeSnapshot() {
  return document.documentElement.classList.contains("dark") ? "dark" : "light";
}

/**
 * Returns an array of chart colors derived from CSS theme variables.
 * Recomputes when the theme changes (light/dark toggle).
 */
export function useChartColors(count = 8): string[] {
  const theme = useSyncExternalStore(subscribeToTheme, getThemeSnapshot);

  return useMemo(() => {
    const style = getComputedStyle(document.documentElement);
    const colors: string[] = [];
    for (let i = 1; i <= 5; i++) {
      const c = style.getPropertyValue(`--color-chart-${i}`).trim();
      if (c) colors.push(c);
    }
    // Extend with additional distinct colors if we need more than 5
    if (colors.length < count) {
      const extras = ["#8884D8", "#82CA9D", "#FFC658", "#D0ED57"];
      for (const e of extras) {
        if (colors.length >= count) break;
        colors.push(e);
      }
    }
    return colors;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [theme, count]);
}
