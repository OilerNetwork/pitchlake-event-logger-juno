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