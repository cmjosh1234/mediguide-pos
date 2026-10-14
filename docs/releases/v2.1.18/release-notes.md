# MediGuide v2.1.18

Changes since v2.1.17. Mobile version: 2.1.18+71.

## Mobile application

- Add readers tailored to document kinds, including manuals and frameworks.
- Redesign document overviews, contents navigation, reading settings and source-page access.
- Keep one bottom More menu with grouped reading, library, offline and source actions.
- Fix mobile CI analysis and update reader screenshot fixtures and baselines.

## API

- Apply migration 83 to add Manual and Framework document kinds.
- Add deterministic local reader fixtures for all document kinds with original PDFs.
- Filter content-hub assignments by document kind and support audited pillar lifecycle operations.

## AI worker

- No service-specific changes since v2.1.17.

## Dashboard

- Improve content-hub assignment filtering, pillar archive/save, unarchive, delete and template replacement.

## Public guidelines portal

- No service-specific changes since v2.1.17.

## Deployment and operations

- Synchronize platform version 2.1.18 and Android build 71.
- Apply migration 83 through the standard deployment workflow. Reader seed fixtures remain local-development data and must not be seeded in production.
- Distribute the production Android APK through Firebase and upload the production AAB as a Google Play draft using the existing release workflows.
