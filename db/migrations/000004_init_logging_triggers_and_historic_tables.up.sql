
-- Create trigger function for logging Liquidity_Providers updates
CREATE OR REPLACE FUNCTION public.log_lp_update()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO "Liquidity_Providers_Historic" (
        address, vault_address, stashed_balance, locked_balance, unlocked_balance, block_number
    )
    VALUES (
        NEW.address, NEW.vault_address, NEW.stashed_balance, NEW.locked_balance, NEW.unlocked_balance, NEW.latest_block
    )
    ON CONFLICT (address,vault_address, block_number)
    DO UPDATE SET
        stashed_balance = EXCLUDED.stashed_balance,
        locked_balance = EXCLUDED.locked_balance,
        unlocked_balance = EXCLUDED.unlocked_balance;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for Liquidity_Providers
CREATE TRIGGER lp_log_update
AFTER UPDATE
ON public."Liquidity_Providers"
FOR EACH ROW
EXECUTE FUNCTION public.log_lp_update();

-- Create trigger function for logging VaultStates updates
CREATE OR REPLACE FUNCTION public.log_vault_update()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO "Vault_Historic" (
        address, unlocked_balance, locked_balance, stashed_balance, block_number
    )
    VALUES (
        NEW.address, NEW.unlocked_balance, NEW.locked_balance, NEW.stashed_balance, NEW.latest_block
    )
    ON CONFLICT (address, block_number)
    DO UPDATE SET
        unlocked_balance = EXCLUDED.unlocked_balance,
        locked_balance = EXCLUDED.locked_balance,
        stashed_balance = EXCLUDED.stashed_balance;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- Create trigger for VaultStates
CREATE TRIGGER vault_log_update
AFTER UPDATE
ON public."VaultStates"
FOR EACH ROW
EXECUTE FUNCTION public.log_vault_update();