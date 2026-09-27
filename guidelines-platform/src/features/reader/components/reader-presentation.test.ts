import { describe, expect, it } from "vitest";

import { readerSectionTitle } from "./reader-presentation";

describe("readerSectionTitle", () => {
  it("uses the publication's explicit chapter number instead of root position", () => {
    expect(readerSectionTitle("Chapter 1: Epidemiology of Diabetes")).toEqual({
      label: "Chapter 1",
      chapterNumber: "1",
      title: "Epidemiology of Diabetes",
    });
    expect(readerSectionTitle("Chapter 2. Definition and Classification of Diabetes")).toEqual({
      label: "Chapter 2",
      chapterNumber: "2",
      title: "Definition and Classification of Diabetes",
    });
  });

  it("does not invent chapter labels for front matter", () => {
    expect(readerSectionTitle("Introduction and Preface")).toEqual({ title: "Introduction and Preface" });
    expect(readerSectionTitle("Executive Summary")).toEqual({ title: "Executive Summary" });
    expect(readerSectionTitle("Abbreviations")).toEqual({ title: "Abbreviations" });
  });
});
