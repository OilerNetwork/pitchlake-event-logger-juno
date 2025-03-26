-- Create VaultLogs table to track event counts per vault
CREATE TABLE "VaultLogs" (
    id SERIAL PRIMARY KEY,
    vault_address character varying(255) NOT NULL,
    event_count BIGINT NOT NULL,
    block_number numeric(78,0) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indices for efficient querying
CREATE INDEX idx_vaultlogs_vault_address ON "VaultLogs" (vault_address);
CREATE INDEX idx_vaultlogs_block_number ON "VaultLogs" (block_number);

-- Create unique constraint to ensure one entry per vault per block
CREATE UNIQUE INDEX unique_vault_block ON "VaultLogs" (vault_address, block_number);