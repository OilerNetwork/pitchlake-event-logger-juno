-- Drop the unique constraint
DROP INDEX IF EXISTS unique_vault_block;

-- Drop the indices
DROP INDEX IF EXISTS idx_vaultlogs_vault_address;
DROP INDEX IF EXISTS idx_vaultlogs_block_number;

-- Drop the table
DROP TABLE IF EXISTS "VaultLogs";
