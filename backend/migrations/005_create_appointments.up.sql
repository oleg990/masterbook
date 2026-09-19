CREATE TABLE appointments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    client_id BIGINT NOT NULL,

    master_id BIGINT NOT NULL,

    service_id BIGINT NOT NULL,

    start_time TIMESTAMPTZ NOT NULL,

    end_time TIMESTAMPTZ NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT appointments_client_fk
        FOREIGN KEY (client_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT appointments_master_fk
        FOREIGN KEY (master_id)
        REFERENCES master_profiles(id)
        ON DELETE CASCADE,

    CONSTRAINT appointments_service_fk
        FOREIGN KEY (service_id)
        REFERENCES services(id)
        ON DELETE CASCADE,

    CONSTRAINT appointments_time_check
        CHECK (start_time < end_time),

    CONSTRAINT appointments_status_check
        CHECK (
            status IN (
                'pending',
                'confirmed',
                'cancelled',
                'completed'
            )
        )
);