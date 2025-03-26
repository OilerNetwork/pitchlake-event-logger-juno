-- CreateFunction: public.notify_new_block()

CREATE OR REPLACE FUNCTION public.notify_new_block(block_num bigint)
    RETURNS void
    LANGUAGE 'plpgsql'
AS $BODY$
BEGIN
    -- Send notification with the block number
    PERFORM pg_notify('new_block', block_num::text);
END;
$BODY$;

-- CreateFunction: public.notify_revert_block()

CREATE OR REPLACE FUNCTION public.notify_revert_block(block_num bigint)
    RETURNS void
    LANGUAGE 'plpgsql'
AS $BODY$
BEGIN
    -- Send notification with the block number
    PERFORM pg_notify('revert_block', block_num::text);
END;
$BODY$; 