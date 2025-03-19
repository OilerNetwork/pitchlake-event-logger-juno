-- Drop triggers
DROP TRIGGER IF EXISTS lp_log_update ON public."Liquidity_Providers";
DROP TRIGGER IF EXISTS vault_log_update ON public."VaultStates";

-- Drop trigger functions
DROP FUNCTION IF EXISTS public.log_lp_update;
DROP FUNCTION IF EXISTS public.log_vault_update;

-- Drop tables