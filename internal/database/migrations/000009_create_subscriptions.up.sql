CREATE TABLE subscriptions (
    id                 BIGSERIAL    PRIMARY KEY,
    subscriber_user_id BIGINT       NOT NULL REFERENCES users(id),
    source_user_id     BIGINT       NOT NULL REFERENCES users(id),
    source_type        VARCHAR(20)  NOT NULL,
    source_id          BIGINT       NOT NULL,
    cloned_id          BIGINT       NOT NULL,
    created_at         TIMESTAMPTZ,
    deleted_at         TIMESTAMPTZ
);

CREATE INDEX idx_subscriptions_subscriber ON subscriptions (subscriber_user_id);
CREATE UNIQUE INDEX idx_subscriptions_no_dupe ON subscriptions (subscriber_user_id, source_type, source_id) WHERE deleted_at IS NULL;
