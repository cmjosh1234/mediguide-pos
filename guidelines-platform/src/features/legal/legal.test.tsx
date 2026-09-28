import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { AccountDeletionPage } from "./AccountDeletionPage";
import { PrivacyPage } from "./PrivacyPage";

function render(element: ReactNode) {
  return renderToStaticMarkup(
    <MemoryRouter>{element}</MemoryRouter>,
  );
}

describe("public legal pages", () => {
  it("provides account deletion outside the installed application", () => {
    const html = render(<AccountDeletionPage />);

    expect(html).toContain("Request account deletion");
    expect(html).toContain('name="email"');
    expect(html).toContain('name="password"');
    expect(html).toContain('name="confirm"');
    expect(html).toContain('href="/privacy"');
  });

  it("keeps the privacy notice visibly blocked pending publisher review", () => {
    const html = render(<PrivacyPage />);

    expect(html).toContain("Draft for publisher review.");
    expect(html).toContain('href="/delete-account"');
  });
});
