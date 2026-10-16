import { afterEach, describe, expect, it, vi } from "vitest";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";

import { normalizeDashboardBaseUrl } from "./config";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  vi.resetModules();
});

describe("public API configuration", () => {
  it("uses the configured API after the development config script loads", async () => {
    const browser = { __APP_CONFIG__: {} };
    const script = readFileSync(new URL("../public/config.js", import.meta.url), "utf8");
    runInNewContext(script, { window: browser });
    vi.stubGlobal("window", browser);
    vi.stubEnv("VITE_MEDIGUIDE_API_URL", "http://127.0.0.1:8080");
    vi.resetModules();
    const config = await import("./config");
    expect(config.publicApiBaseUrl).toBe("http://127.0.0.1:8080");
  });

  it("preserves production runtime API configuration over build-time defaults", async () => {
    vi.stubGlobal("window", { __APP_CONFIG__: { mediguideApiUrl: "https://api.example.org/" } });
    vi.stubEnv("VITE_MEDIGUIDE_API_URL", "http://127.0.0.1:8080");
    vi.resetModules();
    const config = await import("./config");
    expect(config.publicApiBaseUrl).toBe("https://api.example.org");
  });
});

describe("normalizeDashboardBaseUrl", () => {
  it("adds the dashboard base path to a public origin", () => {
    expect(normalizeDashboardBaseUrl("https://mediguide.health.go.ug")).toBe(
      "https://mediguide.health.go.ug/admin",
    );
  });

  it("does not duplicate an existing dashboard base path", () => {
    expect(
      normalizeDashboardBaseUrl("https://mediguide.health.go.ug/admin/"),
    ).toBe("https://mediguide.health.go.ug/admin");
  });

  it.each([
    "https://mediguide.health.go.ug/admin/login",
    "https://mediguide.health.go.ug/admin/login/",
    "https://mediguide.health.go.ug/login",
  ])("accepts a configured login URL: %s", (value) => {
    expect(normalizeDashboardBaseUrl(value)).toBe(
      "https://mediguide.health.go.ug/admin",
    );
  });

  it("builds the public login and recovery links from production runtime configuration", async () => {
    vi.stubGlobal("window", {
      __APP_CONFIG__: {
        mediguidePosUrl: "https://mediguide.health.go.ug/admin/login",
      },
    });
    vi.resetModules();
    const config = await import("./config");
    expect(config.dashboardLoginUrl).toBe(
      "https://mediguide.health.go.ug/admin/login",
    );
    expect(config.mediguidePosLoginUrl).toBe(config.dashboardLoginUrl);
    expect(`${config.dashboardBaseUrl}/forgot-password`).toBe(
      "https://mediguide.health.go.ug/admin/forgot-password",
    );
  });
});
