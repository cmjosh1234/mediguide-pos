# MediGuide v2.1.16

Changes since v2.1.15. Mobile version: 2.1.16+69.

## Mobile application

- Keep cached/default Remote Config values active when network requests or real-time updates fail.
- Contain Crashlytics reporting failures and include x86_64 alongside ARM Android release architectures.

## API

- No service-specific changes since v2.1.15.

## AI worker

- No service-specific changes since v2.1.15.

## Dashboard

- No service-specific changes since v2.1.15.

## Public guidelines portal

- Display smaller publication tiles and open the reader from the entire tile while preserving separate downloads.

## Deployment and operations

- Include all three Flutter-supported Android release architectures and verify native ELF libraries before distribution.
- Provide a complete production Firebase Remote Config template aligned with the mobile app defaults.
- Retain the explicit manual override for the public storage network probe.
