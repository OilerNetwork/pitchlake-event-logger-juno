DROP TRIGGER IF EXISTS lp_update ON public."Liquidity_Providers";
DROP TRIGGER IF EXISTS or_update ON public."Option_Rounds";
DROP TRIGGER IF EXISTS vault_update ON public."VaultStates";
DROP TRIGGER IF EXISTS ob_update ON public."Option_Buyers";

DROP TRIGGER IF EXISTS bids_insert_trigger ON public."Bids";
DROP TRIGGER IF EXISTS bids_update_trigger ON public."Bids";
DROP TRIGGER IF EXISTS or_insert ON public."Option_Rounds";