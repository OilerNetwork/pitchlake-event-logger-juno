-- Table: public.Events

CREATE TABLE "Events"
(
    event_count BIGINT NOT NULL,
    block_number numeric(78,0) NOT NULL,
    vault_address character varying(255) NOT NULL,
    timestamp numeric(78,0) NOT NULL,
    event_name character varying(255) COLLATE pg_catalog."default" NOT NULL,
    event_keys character varying(256)[] NOT NULL,
    event_data character varying(256)[] NOT NULL,
    transaction_hash character varying(66) NOT NULL
);

CREATE INDEX idx_events_block_number ON "Events" (block_number);
CREATE INDEX idx_events_event_name ON "Events" (event_name);
CREATE INDEX idx_events_vault_address ON "Events" (vault_address);
CREATE INDEX idx_events_transaction_hash ON "Events" (transaction_hash); 

