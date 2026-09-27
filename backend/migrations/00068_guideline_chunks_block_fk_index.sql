-- +goose NO TRANSACTION

-- +goose Up
-- PostgreSQL's ON DELETE SET NULL action for guideline_content_blocks probes
-- guideline_chunks by block_id without a deleted_at predicate. The historical
-- partial index cannot support that foreign-key lookup, which makes replacing
-- large structured projections repeatedly scan the complete chunks table.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_guideline_chunks_block_id_fk
  ON guideline_chunks(block_id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_guideline_chunks_block_id_fk;
