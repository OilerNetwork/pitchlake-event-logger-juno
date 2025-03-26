CREATE TABLE "VaultRegistry" (
    "id" SERIAL PRIMARY KEY,
    "vault_address" VARCHAR(255) NOT NULL,
    "deployed_at" BIGINT NOT NULL,   
)

CREATE FUNCTION public.notify_vault_insert()
    RETURNS trigger
    LANGUAGE 'plpgsql'
    COST 100
    VOLATILE NOT LEAKPROOF
AS $BODY$
    BEGIN
        PERFORM pg_notify('vault_insert', row_to_json(NEW)::text);
        RETURN NEW;
    END;
$BODY$;


CREATE TRIGGER insert_vault_registry_trigger
AFTER INSERT ON "VaultRegistry"
FOR EACH ROW
EXECUTE FUNCTION notify_insert_registry();