CREATE TABLE working_hours (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    master_id BIGINT NOT NULL,

    day_of_week INTEGER NOT NULL,

    start_time TIME NOT NULL,

    end_time TIME NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT working_hours_master_fk
        FOREIGN KEY (master_id)
        REFERENCES master_profiles(id)
        ON DELETE CASCADE,

    CONSTRAINT working_hours_day_check
        CHECK (day_of_week BETWEEN 1 AND 7),

    CONSTRAINT working_hours_time_check
        CHECK (start_time < end_time),

    CONSTRAINT working_hours_master_day_unique
        UNIQUE (master_id, day_of_week)
);