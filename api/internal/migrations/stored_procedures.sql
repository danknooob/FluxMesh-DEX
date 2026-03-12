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

CREATE OR REPLACE FUNCTION fn_cancel_order(p_id TEXT, p_user_id TEXT)
RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE orders
    SET status = 'cancelled', updated_at = NOW()
    WHERE id = p_id::UUID
      AND user_id = p_user_id
      AND deleted_at IS NULL;
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
