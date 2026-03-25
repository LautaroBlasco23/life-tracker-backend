CREATE TABLE categories (
    id                 bigserial PRIMARY KEY,
    name               varchar(100) NOT NULL,
    type               varchar(50) NOT NULL,
    icon               varchar(50),
    applicable_to_freq varchar(20) NOT NULL DEFAULT 'variable',
    created_at         timestamptz,
    updated_at         timestamptz,
    deleted_at         timestamptz
);

CREATE UNIQUE INDEX idx_categories_name ON categories(name);
CREATE INDEX idx_categories_deleted_at ON categories(deleted_at);
