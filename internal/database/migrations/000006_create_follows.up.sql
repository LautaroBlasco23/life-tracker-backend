CREATE TABLE follows (
    follower_id BIGINT      NOT NULL REFERENCES users(id),
    followee_id BIGINT      NOT NULL REFERENCES users(id),
    status      VARCHAR(20) NOT NULL,
    created_at  TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    PRIMARY KEY (follower_id, followee_id),
    CHECK (follower_id <> followee_id)
);

CREATE INDEX idx_follows_followee_status ON follows (followee_id, status);
CREATE INDEX idx_follows_follower_status ON follows (follower_id, status);
