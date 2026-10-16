// Refresh an open library after returning from the dashboard, and while it
// remains visible. This also updates document-kind counts after withdrawal.
export function subscribeToPublicationRefresh(
  refresh: () => void,
  browser: Pick<Window, "addEventListener" | "removeEventListener" | "setInterval" | "clearInterval"> = window,
  page: Pick<Document, "visibilityState" | "addEventListener" | "removeEventListener"> = document,
) {
  const refreshVisible = () => {
    if (page.visibilityState === "visible") refresh();
  };
  browser.addEventListener("focus", refreshVisible);
  page.addEventListener("visibilitychange", refreshVisible);
  const interval = browser.setInterval(refreshVisible, 30_000);
  return () => {
    browser.removeEventListener("focus", refreshVisible);
    page.removeEventListener("visibilitychange", refreshVisible);
    browser.clearInterval(interval);
  };
}
