# MediGuide v2.1.19

Changes since v2.1.18. Mobile version: 2.1.19+72.

## API

- Add audited document withdrawal with a required reason while retaining files, approved versions and review history.
- Republish through existing validation checks while preserving the original approval and recording republication.
- Require public-response revalidation so withdrawn publications are not reused from browser caches.
- Expand development fixtures to cover workflow history tables and preserve withdrawals when reseeding.
