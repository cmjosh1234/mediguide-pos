# Guideline collections

## Purpose and delivery status

Guideline collections let an authenticated reader group published clinical
guidelines into a private, synchronized library. The feature is complete across
the PostgreSQL model, authenticated API, generated contracts, Flutter
repository/cache, Riverpod state, collection screens, and the published
guideline reader.

This document records phases 7 through 10 of the mobile collections delivery:

- Phase 7: the collection detail screen, including real published content,
  empty/add, rename, delete, remove, pagination, and error workflows.
- Phase 8: the public-guideline and reader **Save to collection** workflow,
  including authentication redirect, inline collection creation, idempotent
  membership detection, provider refresh, and user feedback.
- Phase 9: canonical collection communication through `AppMessage`, with safe,
  actionable wording for authentication, conflicts, connectivity, server
  failures, successful mutations, and already-saved membership.
- Phase 10: owner-scoped cache reconciliation, complete-versus-partial snapshot
  tracking, deletion cleanup, rename replacement, and mutation count updates.

Collections are not bookmarks, reading history, downloads, or shared clinical
workspaces. Those remain separate capabilities. A collection stores references
to canonical guideline documents and always resolves their current published
version when read.

| Capability | Purpose | Authentication and storage |
|---|---|---|
| Collection | User-named group of multiple published guidelines | Authenticated, owner-scoped, synchronized |
| Bookmark | Quick marker on one guideline for reading later | Authenticated reading-progress data |
| Download | Device copy of an approved PDF/offline package | Authenticated history plus device storage |

Saving to a collection does not bookmark or download a guideline. Removing a
collection membership does not change either of those independent states.

## User workflow

### Open and manage collections

1. Sign in.
2. Open **My Library**.
3. Select **Collections**.
4. Select an existing collection or choose **New collection**.
5. Enter a name and an optional description.
6. Use the collection action menu to edit or delete it.

Deleting a collection deletes its membership records, not the referenced
guidelines. The UI asks for confirmation before deletion.

### Save a published guideline

1. Open a published guideline.
2. Select the app-bar save action or **More > Save to collection**.
3. Select an existing collection, or select **New** to create one inline.
4. The app confirms the result through `AppMessage`.

Saving the same guideline to the same collection more than once is idempotent.
The existing membership is restored or updated rather than duplicated.

### Remove a saved guideline

1. Open **My Library > Collections** and select the collection.
2. Select **Remove from collection** on the guideline.
3. Confirm the action.

The guideline remains published and available elsewhere. Only the user's
collection membership is removed.

### Guest behavior

Collections are private and require authentication. Protected router paths send
guests to sign-in. The collection pages also render an explicit **Sign in
required** state if they are opened without a valid authenticated state. A guest
who selects **Save to collection** from a public guideline is sent through the
normal sign-in flow instead of creating local anonymous data.

## Architecture and data flow

The implementation follows this path:

```text
Flutter screen or save sheet
  -> Riverpod collection controller
  -> GuidelineLibraryRepository
  -> authenticated /api/v2/library endpoints
  -> GuidelineLibraryService
  -> PostgreSQL guideline_collections / guideline_collection_items
```

Read requests also update an owner-scoped local snapshot. If a transport or
server availability failure occurs, the repository may return that snapshot.
Writes are server-authoritative and are never queued silently.

Important implementation locations:

| Area | Location |
|---|---|
| Database schema | `backend/migrations/00017_guideline_library_and_asset_review.sql` |
| Backend model | `backend/internal/models/guideline_library.go` |
| Service rules | `backend/internal/services/guideline_library_service.go` |
| HTTP contract | `backend/internal/handlers/guideline_library_handler.go` |
| Mobile models and repository | `user_app/lib/features/library/data/` |
| Mobile state | `user_app/lib/features/library/presentation/controllers/` |
| Collection screens and sheets | `user_app/lib/features/library/presentation/` |
| Routes | `user_app/lib/app/router/route_names.dart` and `app_router.dart` |

## Data and security rules

- The server derives the owner from authenticated JWT claims. A client cannot
  provide or override `user_id`.
- Every collection read and mutation includes the owner in its database query.
  Another user's record is returned as not found rather than disclosed.
- Collection names are trimmed, required, and limited to 120 Unicode code
  points. Descriptions are trimmed and limited to 1,000 Unicode code points.
- Names are unique per owner, case-insensitively, among active collections.
- Collection IDs and guideline IDs must be valid UUIDs at the HTTP boundary.
- `sort_order` cannot be negative.
- Only a guideline document with a current published version can be added or
  returned. Draft and review-required content is never exposed through a
  collection.
- Deletion is soft deletion. Re-adding an earlier membership restores it.
- Local collection data uses a `user:<user-id>` cache scope. Logout clears
  private cache through the application authentication lifecycle.
- The cache is not used to turn `400`, `401`, `403`, `404`, or `409` responses
  into apparent success. This prevents stale data from hiding authorization,
  validation, ownership, or conflict decisions.

Do not put patient-identifiable information in collection names or
descriptions. Collections are a personal organizational aid, not a patient
record.

## API contract

All routes below are under `/api/v2`, use bearer authentication, and operate on
the authenticated owner.

| Method | Route | Result |
|---|---|---|
| `GET` | `/library/collections` | Paginated collection summaries |
| `POST` | `/library/collections` | Create a collection |
| `GET` | `/library/collections/{id}` | Get one owned collection |
| `PATCH` | `/library/collections/{id}` | Rename or redescribe a collection |
| `DELETE` | `/library/collections/{id}` | Delete a collection and memberships |
| `GET` | `/library/collections/{id}/items` | Paginated published guidelines |
| `POST` | `/library/collections/{id}/items` | Idempotently add a published guideline |
| `DELETE` | `/library/collections/{id}/items/{guidelineId}` | Remove one membership |

List parameters are `page` and `per_page`. Collection lists additionally allow
`sort=name|created_at|updated_at` and `order=asc|desc`. The server caps page size
at 100. Collection items are ordered by `sort_order`, then creation time.

Expected response classes:

| Status | Meaning |
|---|---|
| `200` / `201` / `204` | Read, create, or mutation succeeded |
| `400` | Invalid JSON, UUID, pagination, text length, sort, or order |
| `401` | Missing, invalid, or expired session |
| `404` | Collection is absent/not owned, guideline is not published, or item is absent |
| `409` | The owner already has a collection with that name |
| `500` | Unexpected persistence failure |

Generated backend, TypeScript, and Dart contracts should be regenerated when a
DTO or route shape changes. Do not hand-maintain a second mobile wire model.

## Offline behavior

Successful collection and item reads are cached for seven days under the
authenticated owner's scope. Empty server snapshots are authoritative and are
cached too. Pagination metadata is retained so an offline page does not invent
items or totals.

Every snapshot records whether it is complete:

- A page-one response is complete only when `total_pages <= 1`. Complete
  collection and item snapshots replace their scoped dataset and tombstone
  records that disappeared from the server.
- Any response from a multi-page dataset is partial. It updates the entities
  and authoritative total that were fetched, but never deletes cached entities
  from pages outside that response.
- Creating or deleting a collection updates its cached total. A rename replaces
  the cached entity with the same ID. Deleting a collection tombstones its
  summary and clears both its item entities and item snapshot.
- Removing a guideline tombstones the cached membership and decrements the
  item and collection totals. Adding a guideline updates a known-complete count;
  because the API returns `204` without the new item entity, the item snapshot
  is marked partial until the next successful item refresh.

This completeness marker is essential: treating page one of a multi-page
response as a full refresh could silently delete valid offline pages, while
treating a genuinely complete empty response as partial could resurrect remote
deletions.

The app falls back only for connectivity failures and eligible `5xx` failures.
It does not provide offline create, edit, delete, save, or remove. When a write
cannot reach the server, the UI keeps the canonical state and displays an
`AppMessage` error. This avoids showing a clinical library organization that
was never synchronized.

## Validation and accessibility

The mobile form mirrors server limits and requires a nonblank name. Server
validation remains authoritative. Long collection names and descriptions are
constrained in list cards, and actions have semantic tooltips.

The regression suite verifies the list at a 320 logical-pixel viewport with a
200% text scale. Any layout change to collection cards, sheets, menus, or
actions must preserve that baseline, keyboard-safe form scrolling, screen
reader labels, and a touch target suitable for mobile use.

All transient success and failure communication must use `AppMessage`; do not
introduce direct `SnackBar`, toast, or package-specific messaging calls.
Collection wording is centralized in
`features/library/presentation/utils/collection_messages.dart`. UI code must
map the operation and caught error through this utility rather than display a
raw Dio message, database exception, stack trace, or server response. Expected
messages cover created/updated/deleted, saved/removed/already saved,
duplicate-name conflict, sign-in required, offline mutation unavailable, and
retryable network/server failure.

## Test and verification runbook

From the repository root, run focused backend service and HTTP handler tests:

```bash
cd backend
go test ./internal/services ./internal/handlers -run GuidelineLibrary
```

Run the mobile model, repository, controller, workflow, and route tests:

```bash
cd user_app
fvm flutter test \
  test/guideline_library_models_test.dart \
  test/guideline_library_repository_controller_test.dart \
  test/route_names_test.dart
fvm flutter analyze
```

Before a release, run the full validation matrix from the implementation brief:

```bash
cd backend
go test ./...
go vet ./...
go build ./cmd/api

cd ../user_app
dart format --set-exit-if-changed .
flutter analyze
flutter test
flutter build apk --release --split-per-abi \
  --flavor production \
  --dart-define=APP_ENV=production

cd ..
make contracts-check
git diff --check
bash scripts/check-release-readiness.sh v2.1.0 --metadata-only
```

The focused tests include HTTP status/envelope assertions, ownership and
published-only service rules, complete and partial cache reconciliation,
offline mutation messages, create/edit/delete interactions, and duplicate-tap
prevention. Full Flutter golden failures must be compared with the repository's
committed golden baseline; do not approve a pixel change as part of Collections
unless the UI change was intentional.

Before release, manually verify with two different accounts:

1. Account A creates, edits, opens, and deletes a collection.
2. Account A saves a published guideline twice and sees one membership.
3. Account B cannot open or mutate Account A's collection ID.
4. A draft or review-required guideline cannot be inserted through the API.
5. Account A removes a guideline after the confirmation dialog.
6. With the network unavailable, an already loaded collection can be read but a
   mutation fails visibly and does not alter canonical state.
7. After logout and login as Account B, Account A's cached collections are not
   displayed.
8. Test a narrow Android device with system font size at 200%.
9. Cache at least two collection pages, refresh only page one, go offline, and
   confirm the second cached page is still available.
10. Delete and rename collections, then simulate an offline read and confirm the
    deleted collection stays absent and the renamed value remains current.

## Rollout and rollback

Deploy in this order:

1. Apply migration `00017_guideline_library_and_asset_review.sql` through the
   normal backend migration workflow.
2. Deploy the backend and verify authenticated collection routes.
3. Release the mobile build to alpha testers.
4. Complete the two-account and offline checks above.
5. Promote the same tested artifact through beta and production using the
   mobile release workflow.

The API is additive, so an older mobile client can continue operating while the
new backend is deployed. If the mobile workflow is defective, stop promotion
and roll back the client artifact; collection rows can remain safely in the
database. Do not run the migration's down section in production merely to hide
the client feature because it drops collection and download history. Database
rollback requires a separately approved backup and data-recovery plan.

## Troubleshooting

### Collections do not appear

- Confirm the user is signed in and the bearer token is accepted.
- Inspect `GET /api/v2/library/collections?page=1&per_page=100`.
- A `401` is an authentication problem; do not use cached data as a workaround.
- Confirm the mobile request and cache scope use the same authenticated user ID.
- If the server returns an empty successful page, the empty list is canonical.

### Creating or renaming returns 409

The owner already has an active collection with the same name, ignoring case.
Choose a different name. Do not remove the unique index or retry automatically.

### Saving a guideline returns 404

Check both possibilities:

- the collection does not exist for the authenticated owner; or
- the guideline document has no current version whose status is `published`.

The endpoint intentionally does not add draft or review-required content.

### A saved guideline disappears from a collection

Collection reads join through the guideline document's current published
version. Confirm that the document still has a valid current published version
and was not soft-deleted. The membership may still exist while unpublished
content remains intentionally hidden.

### Offline content belongs to the wrong account

Treat this as a privacy defect. Confirm that calls pass the authenticated user
ID, cache keys begin with `user:<id>`, and logout invoked the private cache
cleaner. Reproduce with two accounts and do not ship until isolation passes.

### A mutation failed but the screen changed

Controllers must restore their prior state when repository writes throw. Check
that the screen calls the Riverpod controller rather than mutating a local list,
and that the failure is displayed through `AppMessage`.

### A deleted or renamed collection reappears offline

- Confirm the client stores a complete snapshot only when page one reports
  `total_pages <= 1`.
- Confirm successful deletes tombstone the collection, remove its ID from the
  collection snapshot, and clear its item snapshot and item entities.
- Confirm successful renames replace the cached entity under the existing ID.
- For multi-page data, fetch all affected pages before diagnosing a missing
  entity. A partial snapshot intentionally preserves unfetched cached pages.
- If the cache predates completeness metadata, perform one successful online
  refresh before retesting offline behavior. Avoid clearing another user's or
  the public cache scope as a workaround.

## Acceptance criteria

The collections feature is releasable when all of the following are true:

- migration `00017` is applied and backend health checks pass;
- owner isolation and published-only service tests pass;
- mobile model, repository, controller, route, and interaction tests pass;
- Flutter analysis is clean;
- create, edit, delete, save, idempotent save, remove, guest, offline, and
  account-switch workflows pass manual verification;
- 320-pixel/200%-text accessibility verification passes; and
- the alpha artifact uses the intended production API base URL and is promoted
  according to `docs/mobile-release-workflow.md`.
