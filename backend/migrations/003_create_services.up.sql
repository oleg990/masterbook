CREATE TABLE services (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    master_id BIGINT NOT NULL,

    name VARCHAR(150) NOT NULL,

    description TEXT NOT NULL DEFAULT '',

    price NUMERIC(10, 2) NOT NULL,

    duration_minutes INTEGER NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT services_master_fk
        FOREIGN KEY (master_id)
        REFERENCES master_profiles(id)
        ON DELETE CASCADE,

    CONSTRAINT services_price_check
        CHECK (price >= 0),

    CONSTRAINT services_duration_check
        CHECK (duration_minutes > 0)
);