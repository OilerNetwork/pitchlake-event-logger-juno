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

-- Driver Events Table for Issue #3 and Issue #6
-- Handles both DriverEvent and VaultCatchupEvent types
CREATE TABLE "driver_events"
(
    "id" SERIAL PRIMARY KEY,
    "type" VARCHAR(50) NOT NULL,
    -- DriverEvent fields (NULL for VaultCatchupEvent)
    "block_number" NUMERIC(78,0),
    "block_hash" VARCHAR(66),
    -- VaultCatchupEvent fields (NULL for DriverEvent)
    "vault_address" VARCHAR(66),
    "start_block" NUMERIC(78,0),
    "end_block" NUMERIC(78,0),
    -- Common fields
    "timestamp" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_driver_events_type ON "driver_events" (type);
CREATE INDEX idx_driver_events_block_number ON "driver_events" (block_number);
CREATE INDEX idx_driver_events_vault_address ON "driver_events" (vault_address);
CREATE INDEX idx_driver_events_timestamp ON "driver_events" (timestamp);