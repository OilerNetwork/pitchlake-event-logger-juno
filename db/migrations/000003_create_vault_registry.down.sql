DROP TRIGGER IF EXISTS insert_vault_registry_trigger ON "vault_registry";
DROP FUNCTION IF EXISTS notify_vault_insert();

DROP FUNCTION IF EXISTS notify_insert_registry();
DROP TABLE IF EXISTS "driver_events";
DROP TABLE IF EXISTS "vault_registry";