
-- Trigger: lp_update

CREATE TRIGGER lp_update
    AFTER UPDATE 
    ON public."Liquidity_Providers"
    FOR EACH ROW
    EXECUTE FUNCTION public.notify_lp_update();


-- Trigger: or_update

CREATE TRIGGER or_update
    AFTER UPDATE 
    ON public."Option_Rounds"
    FOR EACH ROW
    EXECUTE FUNCTION public.notify_or_update();

    CREATE TRIGGER or_insert
    AFTER INSERT 
    ON public."Option_Rounds"
    FOR EACH ROW
    EXECUTE FUNCTION public.notify_or_update();

-- Trigger: vault_update


CREATE TRIGGER vault_update
    AFTER UPDATE 
    ON public."VaultStates"
    FOR EACH ROW
    EXECUTE FUNCTION public.notify_vault_update();

-- Trigger: ob_update

CREATE TRIGGER ob_update
    AFTER UPDATE 
    ON public."Option_Buyers"
    FOR EACH ROW
    EXECUTE FUNCTION public.notify_ob_update();

CREATE TRIGGER bids_insert_trigger
AFTER INSERT ON public."Bids"
FOR EACH ROW
EXECUTE FUNCTION public.notify_bids_channel();

CREATE TRIGGER bids_update_trigger
AFTER UPDATE ON public."Bids"
FOR EACH ROW
EXECUTE FUNCTION public.notify_bids_channel();