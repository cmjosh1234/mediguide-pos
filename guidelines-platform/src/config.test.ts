import { afterEach, describe, expect, it, vi } from "vitest";

import { normalizeDashboardBaseUrl } from "./config";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.resetModules();
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
