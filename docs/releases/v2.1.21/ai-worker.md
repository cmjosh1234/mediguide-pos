# MediGuide v2.1.21

Changes since v2.1.20. Mobile version: 2.1.21+74.

## AI worker

- Recover slow Ollama embedding batches by splitting timed-out requests while preserving full inputs and order.
- Honor OLLAMA_EMBEDDING_TIMEOUT_SECONDS and retain successful embedding checkpoints across retries.
