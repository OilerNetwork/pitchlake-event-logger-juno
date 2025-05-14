CREATE TABLE "starknet_blocks" (
    block_number numeric(78,0) NOT NULL PRIMARY KEY,
    block_hash character varying(66) NOT NULL,
    parent_hash character varying(66) NOT NULL,
    timestamp numeric(78,0) NOT NULL,
    status varchar(255) NOT NULL
);

CREATE INDEX idx_starknet_blocks_block_number ON "starknet_blocks" (block_number);
CREATE INDEX idx_starknet_blocks_parent_hash ON "starknet_blocks" (parent_hash);


CREATE OR REPLACE FUNCTION notify_insert_starknet_blocks()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('starknet_blocks_insert', NEW.block_number::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION notify_revert_starknet_blocks()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('starknet_blocks_revert', NEW.block_number::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_notify_insert_starknet_blocks
AFTER INSERT ON "starknet_blocks"
FOR EACH ROW EXECUTE FUNCTION notify_insert_starknet_blocks();

CREATE TRIGGER trigger_notify_revert_starknet_blocks
AFTER UPDATE ON "starknet_blocks"
FOR EACH ROW 
WHEN (NEW.status = 'revert')
EXECUTE FUNCTION notify_revert_starknet_blocks();