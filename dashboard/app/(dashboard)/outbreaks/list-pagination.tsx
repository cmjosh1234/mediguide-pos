"use client";

import * as React from "react";

import { Button } from "@/components/ui/button";

export const DEFAULT_PAGE_SIZE = 5;

export type Pagination<T> = {
  visible: T[];
  /** Index of the first visible item in the full list. */
  offset: number;
  page: number;
  pageCount: number;
  pageSize: number;
  total: number;
  setPage: (page: number) => void;
  /** Moves to the page that holds the item at this index of the full list. */
  showIndex: (index: number) => void;
};

/** Client-side paging; omit pageSize to show everything on one page. */
export function usePagination<T>(items: T[], pageSize?: number): Pagination<T> {
  const [page, setPage] = React.useState(1);
  const size = pageSize && pageSize > 0 ? pageSize : Math.max(items.length, 1);
  const pageCount = Math.max(1, Math.ceil(items.length / size));
  const current = Math.min(page, pageCount);
  const offset = (current - 1) * size;
  return {
    visible: items.slice(offset, offset + size),
    offset,
    page: current,
    pageCount,
    pageSize: size,
    total: items.length,
    setPage,
    showIndex: (index) => setPage(Math.floor(index / size) + 1),
  };
}

export function PaginationControls<T>({
  pagination,
  label,
}: {
  pagination: Pagination<T>;
  label: string;
}) {
  const { page, pageCount, offset, visible, total, setPage } = pagination;
  if (pageCount <= 1) return null;
  return (
    <nav className="flex flex-wrap items-center justify-between gap-2 pt-1" aria-label={`${label} pages`}>
      <p className="text-sm text-muted-foreground">
        Showing {offset + 1}–{offset + visible.length} of {total}
      </p>
      <div className="flex items-center gap-2">
        <Button size="sm" variant="outline" disabled={page === 1} onClick={() => setPage(page - 1)}>
          Previous
        </Button>
        <span className="text-sm">
          Page {page} of {pageCount}
        </span>
        <Button size="sm" variant="outline" disabled={page === pageCount} onClick={() => setPage(page + 1)}>
          Next
        </Button>
      </div>
    </nav>
  );
}
