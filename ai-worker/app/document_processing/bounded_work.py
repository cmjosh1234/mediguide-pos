"""Ordered execution with at most `workers` submitted tasks, not an eager map."""
from collections import deque
from concurrent.futures import ThreadPoolExecutor, ProcessPoolExecutor
from multiprocessing import get_context


def ordered_work(function, items, workers=1, processes=False):
    if workers <= 1:
        yield from map(function, items)
        return
    factory = ProcessPoolExecutor if processes else ThreadPoolExecutor
    options = {"mp_context": get_context("spawn")} if processes else {}
    iterator = iter(items)
    pending = deque()
    with factory(max_workers=workers, **options) as pool:
        try:
            for _ in range(workers):
                item = next(iterator, None)
                if item is None:
                    break
                pending.append(pool.submit(function, item))
            while pending:
                yield pending.popleft().result()
                item = next(iterator, None)
                if item is not None:
                    pending.append(pool.submit(function, item))
        finally:
            for future in pending:
                future.cancel()
