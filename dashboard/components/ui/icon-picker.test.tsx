import * as React from "react"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"
import { IconPicker } from "./icon-picker"
import { isValidStoredIcon, StoredIcon } from "./stored-icon"

const fallback = <span data-testid="fallback" />

afterEach(cleanup)

describe("StoredIcon", () => {
  it("never renders an icon name as text", () => {
    const { container } = render(<StoredIcon icon="heart-pulse" fallback={fallback} />)
    expect(container.querySelector("svg")).not.toBeNull()
    expect(container.textContent).not.toContain("heart-pulse")
  })

  it("renders emoji icons as text", () => {
    render(<StoredIcon icon="🧮" fallback={fallback} />)
    expect(screen.getByText("🧮")).toBeInTheDocument()
  })

  it("falls back for empty or unknown identifiers", () => {
    const { rerender } = render(<StoredIcon icon="not-a-real-icon" fallback={fallback} />)
    expect(screen.getByTestId("fallback")).toBeInTheDocument()
    expect(screen.queryByText("not-a-real-icon")).toBeNull()
    rerender(<StoredIcon fallback={fallback} />)
    expect(screen.getByTestId("fallback")).toBeInTheDocument()
  })
})

describe("isValidStoredIcon", () => {
  it("accepts empty values, Lucide names and emoji, and rejects typos", () => {
    expect(isValidStoredIcon("")).toBe(true)
    expect(isValidStoredIcon("heart-pulse")).toBe(true)
    expect(isValidStoredIcon("🩺")).toBe(true)
    expect(isValidStoredIcon("HeartPulse")).toBe(false)
    expect(isValidStoredIcon("heart pulse")).toBe(false)
  })
})

describe("IconPicker", () => {
  function setup(value = "") {
    const onChange = vi.fn()
    render(<IconPicker value={value} onChange={onChange} />)
    fireEvent.click(screen.getByRole("combobox"))
    return onChange
  }

  it("shows the selected icon's label", () => {
    render(<IconPicker value="baby" onChange={() => {}} />)
    expect(screen.getByRole("combobox")).toHaveTextContent("Baby")
  })

  it("selects a curated icon by name", () => {
    const onChange = setup()
    fireEvent.click(screen.getByRole("option", { name: "Thermometer" }))
    expect(onChange).toHaveBeenCalledWith("thermometer")
  })

  it("filters icons by keyword", () => {
    setup()
    fireEvent.change(screen.getByLabelText("Search icons"), { target: { value: "apgar" } })
    expect(screen.getAllByRole("option").map((option) => option.getAttribute("aria-label"))).toEqual(["Baby"])
  })

  it("accepts a pasted emoji and rejects plain text", () => {
    const onChange = setup()
    const input = screen.getByLabelText("Emoji icon")
    const use = screen.getByRole("button", { name: "Use" })
    fireEvent.change(input, { target: { value: "abc" } })
    expect(use).toBeDisabled()
    fireEvent.change(input, { target: { value: "🩺" } })
    fireEvent.click(use)
    expect(onChange).toHaveBeenCalledWith("🩺")
  })

  it("clears the icon", () => {
    const onChange = setup("baby")
    fireEvent.click(screen.getByRole("button", { name: "Clear icon" }))
    expect(onChange).toHaveBeenCalledWith("")
  })
})
