import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test } from "vitest";

import { PaginationControls, usePagination, type Pagination } from "./list-pagination";

afterEach(cleanup);

let latest: Pagination<string> | undefined;

const capture = (pagination: Pagination<string>) => {
  latest = pagination;
};

function List({ items, pageSize, onRender = capture }: { items: string[]; pageSize?: number; onRender?: typeof capture }) {
  const pagination = usePagination(items, pageSize);
  onRender(pagination);
  return (
    <div>
      {pagination.visible.map((item) => <p key={item}>{item}</p>)}
      <PaginationControls pagination={pagination} label="Items" />
    </div>
  );
}

const twelve = Array.from({ length: 12 }, (_, index) => `Item ${index + 1}`);

test("shows five items per page and moves between pages", () => {
  render(<List items={twelve} pageSize={5} />);
  expect(screen.getByText("Showing 1–5 of 12")).toBeTruthy();
  expect(screen.queryByText("Item 6")).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Next" }));
  fireEvent.click(screen.getByRole("button", { name: "Next" }));
  expect(screen.getByText("Page 3 of 3")).toBeTruthy();
  expect(screen.getByText("Showing 11–12 of 12")).toBeTruthy();
});

test("jumps to the page holding a given item", () => {
  render(<List items={twelve} pageSize={5} />);
  act(() => latest!.showIndex(7));
  expect(screen.getByText("Item 8")).toBeTruthy();
  expect(screen.getByText("Page 2 of 3")).toBeTruthy();
});

test("stays on a valid page when items are removed", () => {
  const { rerender } = render(<List items={twelve} pageSize={5} />);
  act(() => latest!.setPage(3));
  rerender(<List items={twelve.slice(0, 6)} pageSize={5} />);
  expect(screen.getByText("Page 2 of 2")).toBeTruthy();
  expect(screen.getByText("Item 6")).toBeTruthy();
});

test("hides the controls when everything fits on one page", () => {
  render(<List items={twelve.slice(0, 3)} pageSize={5} />);
  expect(screen.queryByRole("navigation")).toBeNull();
  render(<List items={twelve} />);
  expect(screen.getAllByText("Item 12")).toHaveLength(1);
});
