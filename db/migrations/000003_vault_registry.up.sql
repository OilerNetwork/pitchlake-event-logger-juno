CREATE TABLE "vault_registry"
(
    "id" SERIAL PRIMARY KEY,
    "vault_address" VARCHAR(66) NOT NULL,
    "deployed_at" VARCHAR(66) NOT NULL,   
    "last_block_indexed" VARCHAR(66),
    "last_block_processed" VARCHAR(66)    
);

CREATE FUNCTION public.notify_insert_registry()
    RETURNS trigger AS $$
    BEGIN 
        PERFORM pg_notify('vault_insert', row_to_json(NEW)::text);
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER insert_vault_registry_trigger
AFTER INSERT ON "vault_registry"
FOR EACH ROW
EXECUTE FUNCTION notify_insert_registry();

-- Unified driver events table for all notification types
CREATE TABLE "driver_events"
(
    "id" SERIAL PRIMARY KEY,
    "sequence_index" BIGINT NOT NULL, -- Sequential counter for ordering
    "type" VARCHAR(50) NOT NULL, -- "StartBlock", "RevertBlock", "CatchupVault"
    "timestamp" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "is_processed" BOOLEAN DEFAULT FALSE,
    
    -- Basic driver event fields (NULL for CatchupVault)
    "block_hash" VARCHAR(66),
    
    -- Vault catchup event fields (NULL for basic driver events)
    "vault_address" VARCHAR(66),
    "start_block_hash" VARCHAR(66), -- Changed from start_block to start_block_hash
    "end_block_hash" VARCHAR(66)    -- Changed from end_block to end_block_hash
);

-- Create sequence for sequential ordering
CREATE SEQUENCE driver_events_sequence START 1;

-- Indexes for efficient querying
CREATE INDEX idx_driver_events_sequence ON "driver_events" (sequence_index);
CREATE INDEX idx_driver_events_type ON "driver_events" (type);
CREATE INDEX idx_driver_events_is_processed ON "driver_events" (is_processed);
CREATE INDEX idx_driver_events_timestamp ON "driver_events" (timestamp);
CREATE INDEX idx_driver_events_block_hash ON "driver_events" (block_hash);
CREATE INDEX idx_driver_events_vault_address ON "driver_events" (vault_address);

-- PostgreSQL NOTIFY trigger for unified driver events
CREATE OR REPLACE FUNCTION notify_driver_event()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('driver_events', 
        json_build_object(
            'id', NEW.id,
            'sequence_index', NEW.sequence_index,
            'type', NEW.type,
            'timestamp', NEW.timestamp,
            'is_processed', NEW.is_processed,
            'block_hash', NEW.block_hash,
            'vault_address', NEW.vault_address,
            'start_block_hash', NEW.start_block_hash,
            'end_block_hash', NEW.end_block_hash
        )::text
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER driver_event_notify_trigger
    AFTER INSERT ON "driver_events"
    FOR EACH ROW
    EXECUTE FUNCTION notify_driver_event();