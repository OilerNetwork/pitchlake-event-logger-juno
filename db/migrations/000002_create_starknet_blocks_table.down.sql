DROP TRIGGER IF EXISTS trigger_notify_insert_starknet_blocks ON "starknet_blocks";
DROP TRIGGER IF EXISTS trigger_notify_revert_starknet_blocks ON "starknet_blocks";
DROP FUNCTION IF EXISTS notify_insert_starknet_blocks();
DROP FUNCTION IF EXISTS notify_revert_starknet_blocks();
DROP INDEX IF EXISTS idx_starknet_blocks_block_number;
DROP INDEX IF EXISTS idx_starknet_blocks_parent_hash;
DROP INDEX IF EXISTS idx_starknet_blocks_timestamp;
DROP TABLE IF EXISTS "starknet_blocks";
