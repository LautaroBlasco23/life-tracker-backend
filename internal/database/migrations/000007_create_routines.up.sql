CREATE TABLE routines (
    id             BIGSERIAL    PRIMARY KEY,
    user_id        BIGINT       NOT NULL REFERENCES users(id),
    title          VARCHAR(255) NOT NULL,
    description    TEXT,
    privacy_status VARCHAR(20)  NOT NULL DEFAULT 'private',
    created_at     TIMESTAMPTZ,
    updated_at     TIMESTAMPTZ,
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX idx_routines_user_id    ON routines (user_id);
CREATE INDEX idx_routines_deleted_at ON routines (deleted_at);
