import { afterEach, expect, it, vi } from "vitest";
import { subscribeToPublicationRefresh } from "./publication-refresh";

afterEach(() => vi.useRealTimers());

it("refreshes on returning to the library and every 30 seconds while visible", () => {
  vi.useFakeTimers();
  const browserEvents = new EventTarget();
  const pageEvents = new EventTarget();
  let visibility: DocumentVisibilityState = "visible";
  const browser = {
    addEventListener: browserEvents.addEventListener.bind(browserEvents),
    removeEventListener: browserEvents.removeEventListener.bind(browserEvents),
    setInterval: ((callback: () => void, delay: number) => setInterval(callback, delay)) as unknown as Window["setInterval"],
    clearInterval: ((id: number) => clearInterval(id)) as Window["clearInterval"],
  };
  const page = {
    get visibilityState() { return visibility; },
    addEventListener: pageEvents.addEventListener.bind(pageEvents),
    removeEventListener: pageEvents.removeEventListener.bind(pageEvents),
  };
  const refresh = vi.fn();
  const stop = subscribeToPublicationRefresh(refresh, browser, page);
  browserEvents.dispatchEvent(new Event("focus"));
  vi.advanceTimersByTime(30_000);
  expect(refresh).toHaveBeenCalledTimes(2);
  visibility = "hidden";
  pageEvents.dispatchEvent(new Event("visibilitychange"));
  vi.advanceTimersByTime(30_000);
  expect(refresh).toHaveBeenCalledTimes(2);
  visibility = "visible";
  pageEvents.dispatchEvent(new Event("visibilitychange"));
  expect(refresh).toHaveBeenCalledTimes(3);
  stop();
  browserEvents.dispatchEvent(new Event("focus"));
  pageEvents.dispatchEvent(new Event("visibilitychange"));
  vi.advanceTimersByTime(60_000);
  expect(refresh).toHaveBeenCalledTimes(3);
});
