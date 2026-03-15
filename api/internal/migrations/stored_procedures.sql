-- ============================================================
-- FluxMesh DEX – PostgreSQL stored functions
-- Executed via CREATE OR REPLACE so the migration is idempotent.
-- ============================================================

-- ===================  ORDERS  ================================

CREATE OR REPLACE FUNCTION fn_find_order_by_idempotency_key(p_key TEXT)
RETURNS SETOF orders
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM orders
    WHERE idempotency_key = p_key AND deleted_at IS NULL
    LIMIT 1;
END;
$$;

CREATE OR REPLACE FUNCTION fn_create_order(
    p_idempotency_key TEXT,
    p_user_id         TEXT,
    p_market_id       TEXT,
    p_side            TEXT,
    p_type            TEXT,
    p_price           TEXT,
    p_size            TEXT,
    p_remaining       TEXT
) RETURNS SETOF orders
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    INSERT INTO orders (
        id, idempotency_key, user_id, market_id,
        side, "type", price, size, remaining,
        status, created_at, updated_at
    ) VALUES (
        gen_random_uuid(),
        NULLIF(p_idempotency_key, ''),
        p_user_id, p_market_id,
        p_side, p_type,
        p_price::NUMERIC, p_size::NUMERIC, p_remaining::NUMERIC,
        'pending', NOW(), NOW()
    )
    RETURNING *;
END;
$$;

CREATE OR REPLACE FUNCTION fn_list_orders(
    p_user_id   TEXT DEFAULT NULL,
    p_market_id TEXT DEFAULT NULL,
    p_status    TEXT DEFAULT NULL
) RETURNS SETOF orders
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM orders
    WHERE deleted_at IS NULL
      AND (p_user_id   IS NULL OR user_id   = p_user_id)
      AND (p_market_id IS NULL OR market_id = p_market_id)
      AND (p_status    IS NULL OR status    = p_status)
    ORDER BY created_at DESC;
END;
$$;

CREATE OR REPLACE FUNCTION fn_order_depth(
    p_market_id TEXT,
    p_side      TEXT,
    p_limit     INT DEFAULT 20
) RETURNS TABLE(price TEXT, total_size TEXT, count BIGINT)
LANGUAGE plpgsql AS $$
BEGIN
    IF p_limit <= 0 OR p_limit > 50 THEN
        p_limit := 20;
    END IF;

    IF p_side = 'sell' THEN
        RETURN QUERY
        SELECT o.price::TEXT,
               SUM(o.remaining)::TEXT,
               COUNT(*)
        FROM orders o
        WHERE o.market_id = p_market_id
          AND o.side      = p_side
          AND o.status    = 'pending'
          AND o.deleted_at IS NULL
        GROUP BY o.price
        ORDER BY o.price ASC
        LIMIT p_limit;
    ELSE
        RETURN QUERY
        SELECT o.price::TEXT,
               SUM(o.remaining)::TEXT,
               COUNT(*)
        FROM orders o
        WHERE o.market_id = p_market_id
          AND o.side      = p_side
          AND o.status    = 'pending'
          AND o.deleted_at IS NULL
        GROUP BY o.price
        ORDER BY o.price DESC
        LIMIT p_limit;
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION fn_get_order_by_id(p_id TEXT)
RETURNS SETOF orders
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM orders
    WHERE id = p_id::UUID AND deleted_at IS NULL
    LIMIT 1;
END;
$$;

CREATE OR REPLACE FUNCTION fn_update_order(
    p_id        TEXT,
    p_user_id   TEXT,
    p_market_id TEXT,
    p_side      TEXT,
    p_type      TEXT,
    p_price     TEXT,
    p_size      TEXT,
    p_remaining TEXT,
    p_status    TEXT
) RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE orders SET
        user_id   = p_user_id,
        market_id = p_market_id,
        side      = p_side,
        "type"    = p_type,
        price     = p_price::NUMERIC,
        size      = p_size::NUMERIC,
        remaining = p_remaining::NUMERIC,
        status    = p_status,
        updated_at = NOW()
    WHERE id = p_id::UUID AND deleted_at IS NULL;
END;
$$;

-- Guarded cancel — enforces cancellation rules and computes fee:
--
--   Status      | Cancellable? | Reason
--   ------------|-------------|-----------------------------------
--   pending     | YES         | Not yet matched
--   partial     | YES         | Cancels remaining qty only
--   matched     | NO          | Trade already executed
--   rejected    | NO          | Already terminal
--   cancelled   | NO          | Already cancelled
--   market type | NO          | Executes instantly at market price
--
-- Fee = remaining × price × market.cancel_fee_rate
-- Fee is capped at the user's available balance (never goes negative).
-- Fee asset: quote asset for buy orders, base asset for sell orders.
-- Raises ORDER_NOT_FOUND or ORDER_NOT_CANCELLABLE on failure.
-- Returns the updated order row on success.
CREATE OR REPLACE FUNCTION fn_cancel_order(p_id TEXT, p_user_id TEXT)
RETURNS SETOF orders
LANGUAGE plpgsql AS $$
DECLARE
    v_order     orders%ROWTYPE;
    v_fee_rate  NUMERIC;
    v_base      TEXT;
    v_quote     TEXT;
    v_fee       NUMERIC;
    v_fee_asset TEXT;
    v_available NUMERIC;
BEGIN
    SELECT * INTO v_order
    FROM orders
    WHERE id = p_id::UUID
      AND user_id = p_user_id
      AND deleted_at IS NULL
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'ORDER_NOT_FOUND';
    END IF;

    IF v_order."type" = 'market' THEN
        RAISE EXCEPTION 'ORDER_NOT_CANCELLABLE:market orders execute instantly';
    END IF;

    IF v_order.status NOT IN ('pending', 'partial') THEN
        RAISE EXCEPTION 'ORDER_NOT_CANCELLABLE:order is already %', v_order.status;
    END IF;

    -- Look up market fee configuration
    SELECT COALESCE(m.cancel_fee_rate, 0), m.base_asset, m.quote_asset
    INTO v_fee_rate, v_base, v_quote
    FROM markets m
    WHERE m.id = v_order.market_id AND m.deleted_at IS NULL;

    IF v_order.side = 'buy' THEN
        v_fee_asset := v_quote;
    ELSE
        v_fee_asset := v_base;
    END IF;

    v_fee := v_order.remaining * v_order.price * v_fee_rate;

    SELECT COALESCE(b.available, 0)::NUMERIC INTO v_available
    FROM balances b
    WHERE b.user_id = p_user_id AND b.asset = v_fee_asset;

    IF v_available IS NULL THEN
        v_available := 0;
    END IF;

    IF v_fee > v_available THEN
        v_fee := v_available;
    END IF;

    -- Deduct fee from balance
    IF v_fee > 0 THEN
        UPDATE balances
        SET available  = (available::NUMERIC - v_fee)::TEXT,
            updated_at = NOW()
        WHERE user_id = p_user_id AND asset = v_fee_asset;
    END IF;

    -- Cancel the order and record the fee
    UPDATE orders
    SET status     = 'cancelled',
        cancel_fee = v_fee,
        updated_at = NOW()
    WHERE id = p_id::UUID;

    RETURN QUERY
    SELECT * FROM orders WHERE id = p_id::UUID;
END;
$$;

CREATE OR REPLACE FUNCTION fn_update_order_status(
    p_order_id  TEXT,
    p_status    TEXT,
    p_remaining TEXT DEFAULT NULL
) RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    IF p_remaining IS NOT NULL AND p_remaining <> '' THEN
        UPDATE orders
        SET status    = p_status,
            remaining = p_remaining::NUMERIC,
            updated_at = NOW()
        WHERE id = p_order_id::UUID;
    ELSE
        UPDATE orders
        SET status     = p_status,
            updated_at = NOW()
        WHERE id = p_order_id::UUID;
    END IF;
END;
$$;

-- Used by the matching engine on startup to restore resting orders
-- into the in-memory order book after a restart.
CREATE OR REPLACE FUNCTION fn_get_resting_orders()
RETURNS TABLE(
    id         UUID,
    user_id    TEXT,
    market_id  TEXT,
    side       TEXT,
    price      NUMERIC,
    remaining  NUMERIC,
    created_at TIMESTAMPTZ
)
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    SELECT o.id, o.user_id, o.market_id, o.side, o.price, o.remaining, o.created_at
    FROM orders o
    WHERE o.status IN ('pending', 'partial')
      AND o.deleted_at IS NULL
    ORDER BY o.created_at ASC;
END;
$$;

-- ===================  USERS  =================================

CREATE OR REPLACE FUNCTION fn_create_user(
    p_email         TEXT,
    p_name          TEXT,
    p_avatar_url    TEXT,
    p_password_hash TEXT,
    p_role          TEXT
) RETURNS SETOF users
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    INSERT INTO users (id, email, name, avatar_url, password_hash, role, created_at, updated_at)
    VALUES (
        gen_random_uuid(), p_email,
        COALESCE(p_name, ''), COALESCE(p_avatar_url, ''),
        p_password_hash, p_role, NOW(), NOW()
    )
    RETURNING *;
END;
$$;

CREATE OR REPLACE FUNCTION fn_find_user_by_email(p_email TEXT)
RETURNS SETOF users
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM users
    WHERE email = p_email AND deleted_at IS NULL
    LIMIT 1;
END;
$$;

CREATE OR REPLACE FUNCTION fn_find_user_by_id(p_id TEXT)
RETURNS SETOF users
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM users
    WHERE id = p_id::UUID AND deleted_at IS NULL
    LIMIT 1;
END;
$$;

CREATE OR REPLACE FUNCTION fn_update_user(
    p_id            TEXT,
    p_email         TEXT,
    p_name          TEXT,
    p_avatar_url    TEXT,
    p_password_hash TEXT,
    p_role          TEXT
) RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE users SET
        email         = p_email,
        name          = p_name,
        avatar_url    = p_avatar_url,
        password_hash = p_password_hash,
        role          = p_role,
        updated_at    = NOW()
    WHERE id = p_id::UUID AND deleted_at IS NULL;
END;
$$;

CREATE OR REPLACE FUNCTION fn_soft_delete_user(p_id TEXT)
RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE users
    SET deleted_at = NOW(), updated_at = NOW()
    WHERE id = p_id::UUID AND deleted_at IS NULL;
END;
$$;

-- ===================  MARKETS  ===============================

CREATE OR REPLACE FUNCTION fn_list_enabled_markets()
RETURNS SETOF markets
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM markets
    WHERE enabled = TRUE AND deleted_at IS NULL;
END;
$$;

CREATE OR REPLACE FUNCTION fn_get_market_by_id(p_id TEXT)
RETURNS SETOF markets
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM markets
    WHERE id = p_id AND deleted_at IS NULL
    LIMIT 1;
END;
$$;

-- ===================  BALANCES  ==============================

CREATE OR REPLACE FUNCTION fn_list_balances_by_user(p_user_id TEXT)
RETURNS SETOF balances
LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM balances
    WHERE user_id = p_user_id AND deleted_at IS NULL;
END;
$$;

CREATE OR REPLACE FUNCTION fn_upsert_balance(
    p_user_id   TEXT,
    p_asset     TEXT,
    p_available TEXT,
    p_locked    TEXT
) RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO balances (user_id, asset, available, locked, updated_at)
    VALUES (p_user_id, p_asset, p_available::NUMERIC, p_locked::NUMERIC, NOW())
    ON CONFLICT (user_id, asset)
    DO UPDATE SET
        available  = EXCLUDED.available,
        locked     = EXCLUDED.locked,
        updated_at = NOW();
END;
$$;

-- ===================  COMPOSITE / ATOMIC  ====================

-- Atomically updates both order statuses and creates the trade record.
-- PL/pgSQL functions execute within a single transaction; if any
-- statement fails the entire function rolls back.
CREATE OR REPLACE FUNCTION fn_process_order_matched(
    p_maker_order_id TEXT,
    p_taker_order_id TEXT,
    p_maker_remaining TEXT,
    p_taker_remaining TEXT,
    p_trade_id       TEXT,
    p_market_id      TEXT,
    p_price          TEXT,
    p_size           TEXT,
    p_maker_side     TEXT
) RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    -- 1. Update maker order
    UPDATE orders
    SET status    = CASE WHEN p_maker_remaining::NUMERIC = 0 THEN 'matched' ELSE 'partial' END,
        remaining = p_maker_remaining::NUMERIC,
        updated_at = NOW()
    WHERE id = p_maker_order_id::UUID;

    -- 2. Update taker order
    UPDATE orders
    SET status    = CASE WHEN p_taker_remaining::NUMERIC = 0 THEN 'matched' ELSE 'partial' END,
        remaining = p_taker_remaining::NUMERIC,
        updated_at = NOW()
    WHERE id = p_taker_order_id::UUID;

    -- 3. Insert trade (idempotent via ON CONFLICT)
    INSERT INTO trades (
        id, market_id, maker_order_id, taker_order_id,
        price, size, maker_side, created_at, updated_at
    ) VALUES (
        p_trade_id, p_market_id, p_maker_order_id, p_taker_order_id,
        p_price::NUMERIC, p_size::NUMERIC, p_maker_side,
        NOW(), NOW()
    )
    ON CONFLICT (id) DO NOTHING;
END;
$$;

-- Atomically check idempotency key and insert an order.
-- Returns the row and a boolean `is_duplicate`.
-- Drop first so we can change return type (PostgreSQL does not allow changing it with CREATE OR REPLACE).
DROP FUNCTION IF EXISTS fn_create_order_atomic(TEXT, TEXT, TEXT, TEXT, TEXT, TEXT, TEXT, TEXT);
CREATE OR REPLACE FUNCTION fn_create_order_atomic(
    p_idempotency_key TEXT,
    p_user_id         TEXT,
    p_market_id       TEXT,
    p_side            TEXT,
    p_type            TEXT,
    p_price           TEXT,
    p_size            TEXT,
    p_remaining       TEXT
) RETURNS TABLE(
    id              UUID,
    idempotency_key TEXT,
    user_id         TEXT,
    market_id       TEXT,
    side            TEXT,
    "type"          TEXT,
    price           TEXT,
    size            TEXT,
    remaining       TEXT,
    status          TEXT,
    cancel_fee      TEXT,
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,
    is_duplicate    BOOLEAN
)
LANGUAGE plpgsql AS $$
DECLARE
    ins_row orders%ROWTYPE;
BEGIN
    -- Fast-path: idempotency check. Use subquery with aliased columns to avoid 42702 (id ambiguous with RETURNS TABLE).
    IF p_idempotency_key IS NOT NULL AND p_idempotency_key <> '' THEN
        RETURN QUERY
            SELECT sub.r_id::UUID,
                   sub.r_idempotency_key::TEXT,
                   sub.r_user_id::TEXT,
                   sub.r_market_id::TEXT,
                   sub.r_side::TEXT,
                   sub.r_type::TEXT,
                   sub.r_price::TEXT,
                   sub.r_size::TEXT,
                   sub.r_remaining::TEXT,
                   sub.r_status::TEXT,
                   sub.r_cancel_fee::TEXT,
                   sub.r_created_at::TIMESTAMPTZ,
                   sub.r_updated_at::TIMESTAMPTZ,
                   sub.r_deleted_at::TIMESTAMPTZ,
                   TRUE::BOOLEAN
            FROM (
                SELECT o.id AS r_id, o.idempotency_key AS r_idempotency_key,
                       o.user_id AS r_user_id, o.market_id AS r_market_id,
                       o.side AS r_side, o."type" AS r_type,
                       o.price AS r_price, o.size AS r_size, o.remaining AS r_remaining,
                       o.status AS r_status, COALESCE(o.cancel_fee, 0) AS r_cancel_fee,
                       o.created_at AS r_created_at, o.updated_at AS r_updated_at,
                       o.deleted_at AS r_deleted_at
                FROM orders o
                WHERE o.idempotency_key = p_idempotency_key
                  AND o.deleted_at IS NULL
                LIMIT 1
            ) sub;
        IF FOUND THEN RETURN; END IF;
    END IF;

    -- Insert; on unique-violation (race), return existing row.
    -- Use a record variable so RETURNING * has no column names in scope (avoids 42702).
    BEGIN
        INSERT INTO orders (
            id, idempotency_key, user_id, market_id,
            side, "type", price, size, remaining,
            status, created_at, updated_at
        ) VALUES (
            gen_random_uuid(),
            NULLIF(p_idempotency_key, ''),
            p_user_id, p_market_id,
            p_side, p_type,
            p_price::NUMERIC, p_size::NUMERIC, p_remaining::NUMERIC,
            'pending', NOW(), NOW()
        )
        RETURNING * INTO ins_row;

        RETURN QUERY SELECT
            ins_row.id,
            ins_row.idempotency_key::TEXT,
            ins_row.user_id::TEXT,
            ins_row.market_id::TEXT,
            ins_row.side::TEXT,
            ins_row."type"::TEXT,
            ins_row.price::TEXT,
            ins_row.size::TEXT,
            ins_row.remaining::TEXT,
            ins_row.status::TEXT,
            COALESCE(ins_row.cancel_fee, 0)::TEXT,
            ins_row.created_at::TIMESTAMPTZ,
            ins_row.updated_at::TIMESTAMPTZ,
            ins_row.deleted_at::TIMESTAMPTZ,
            FALSE::BOOLEAN;
    EXCEPTION WHEN unique_violation THEN
        RETURN QUERY
            SELECT sub.r_id::UUID,
                   sub.r_idempotency_key::TEXT,
                   sub.r_user_id::TEXT,
                   sub.r_market_id::TEXT,
                   sub.r_side::TEXT,
                   sub.r_type::TEXT,
                   sub.r_price::TEXT,
                   sub.r_size::TEXT,
                   sub.r_remaining::TEXT,
                   sub.r_status::TEXT,
                   sub.r_cancel_fee::TEXT,
                   sub.r_created_at::TIMESTAMPTZ,
                   sub.r_updated_at::TIMESTAMPTZ,
                   sub.r_deleted_at::TIMESTAMPTZ,
                   TRUE::BOOLEAN
            FROM (
                SELECT o.id AS r_id, o.idempotency_key AS r_idempotency_key,
                       o.user_id AS r_user_id, o.market_id AS r_market_id,
                       o.side AS r_side, o."type" AS r_type,
                       o.price AS r_price, o.size AS r_size, o.remaining AS r_remaining,
                       o.status AS r_status, COALESCE(o.cancel_fee, 0) AS r_cancel_fee,
                       o.created_at AS r_created_at, o.updated_at AS r_updated_at,
                       o.deleted_at AS r_deleted_at
                FROM orders o
                WHERE o.idempotency_key = p_idempotency_key
                  AND o.deleted_at IS NULL
                LIMIT 1
            ) sub;
    END;
END;
$$;

-- Atomically check email uniqueness and insert a user.
-- Raises 'EMAIL_TAKEN' on duplicate so the Go layer can
-- distinguish from other constraint errors.
CREATE OR REPLACE FUNCTION fn_register_user_atomic(
    p_email         TEXT,
    p_name          TEXT,
    p_avatar_url    TEXT,
    p_password_hash TEXT,
    p_role          TEXT
) RETURNS SETOF users
LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE email = p_email AND deleted_at IS NULL) THEN
        RAISE EXCEPTION 'EMAIL_TAKEN';
    END IF;

    BEGIN
        RETURN QUERY
        INSERT INTO users (id, email, name, avatar_url, password_hash, role, created_at, updated_at)
        VALUES (
            gen_random_uuid(), p_email,
            COALESCE(p_name, ''), COALESCE(p_avatar_url, ''),
            p_password_hash, p_role, NOW(), NOW()
        )
        RETURNING *;
    EXCEPTION WHEN unique_violation THEN
        RAISE EXCEPTION 'EMAIL_TAKEN';
    END;
END;
$$;

-- Atomically find a user by ID, check email uniqueness if changed,
-- and update the row. Returns the updated user row.
CREATE OR REPLACE FUNCTION fn_update_profile_atomic(
    p_id         TEXT,
    p_email      TEXT,
    p_name       TEXT,
    p_avatar_url TEXT
) RETURNS SETOF users
LANGUAGE plpgsql AS $$
DECLARE
    v_user users%ROWTYPE;
BEGIN
    SELECT * INTO v_user FROM users
    WHERE id = p_id::UUID AND deleted_at IS NULL
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'USER_NOT_FOUND';
    END IF;

    -- Email uniqueness check (skip if unchanged)
    IF p_email IS DISTINCT FROM v_user.email THEN
        IF EXISTS (
            SELECT 1 FROM users
            WHERE email = p_email AND deleted_at IS NULL AND id <> p_id::UUID
        ) THEN
            RAISE EXCEPTION 'EMAIL_TAKEN';
        END IF;
    END IF;

    UPDATE users SET
        email      = COALESCE(p_email, v_user.email),
        name       = COALESCE(p_name, v_user.name),
        avatar_url = COALESCE(p_avatar_url, v_user.avatar_url),
        updated_at = NOW()
    WHERE id = p_id::UUID AND deleted_at IS NULL;

    RETURN QUERY SELECT * FROM users WHERE id = p_id::UUID;
END;
$$;

-- ===================  TRADES  ================================

CREATE OR REPLACE FUNCTION fn_create_trade_if_not_exists(
    p_id             TEXT,
    p_market_id      TEXT,
    p_maker_order_id TEXT,
    p_taker_order_id TEXT,
    p_price          TEXT,
    p_size           TEXT,
    p_maker_side     TEXT
) RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO trades (
        id, market_id, maker_order_id, taker_order_id,
        price, size, maker_side, created_at, updated_at
    ) VALUES (
        p_id, p_market_id, p_maker_order_id, p_taker_order_id,
        p_price::NUMERIC, p_size::NUMERIC, p_maker_side,
        NOW(), NOW()
    )
    ON CONFLICT (id) DO NOTHING;
END;
$$;

CREATE OR REPLACE FUNCTION fn_mark_trade_settled(p_trade_id TEXT)
RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE trades
    SET settled_at = NOW(), updated_at = NOW()
    WHERE id = p_trade_id;
END;
$$;
