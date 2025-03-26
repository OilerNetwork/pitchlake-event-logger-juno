CREATE TABLE "StarknetBlocks" (
    block_number numeric(78,0) NOT NULL PRIMARY KEY,
    block_hash character varying(66) NOT NULL,
    parent_hash character varying(66) NOT NULL,
    status varchar(255) NOT NULL,
    is_processed boolean NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_starknet_blocks_block_number ON "StarknetBlocks" (block_number);
CREATE INDEX idx_starknet_blocks_parent_hash ON "StarknetBlocks" (parent_hash);
CREATE INDEX idx_starknet_blocks_timestamp ON "StarknetBlocks" (timestamp);
