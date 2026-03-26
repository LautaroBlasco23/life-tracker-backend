CREATE TABLE users (
    id          bigserial PRIMARY KEY,
    first_name  varchar(255) NOT NULL,
    last_name   varchar(255) NOT NULL,
    profile_pic_url text,
    thumbnail_url   text,
    timezone    varchar(64),
    created_at  timestamptz,
    updated_at  timestamptz,
    deleted_at  timestamptz
);

CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- ---

CREATE TABLE auths (
    id            bigserial PRIMARY KEY,
    user_id       bigint NOT NULL,
    email         varchar(255) NOT NULL,
    password_hash text NOT NULL,
    created_at    timestamptz,
    updated_at    timestamptz,
    deleted_at    timestamptz
);

CREATE UNIQUE INDEX idx_auths_email ON auths(email);
CREATE INDEX idx_auths_deleted_at ON auths(deleted_at);

-- ---

CREATE TABLE activities (
    id                bigserial PRIMARY KEY,
    user_id           bigint NOT NULL,
    title             varchar(255) NOT NULL,
    description       text,
    frequency         varchar(50) NOT NULL,
    day_frequency     text,
    day_time          varchar(50) NOT NULL DEFAULT 'notSpecified',
    completion_amount bigint NOT NULL DEFAULT 1,
    is_active         boolean NOT NULL DEFAULT false,
    created_at        timestamptz,
    updated_at        timestamptz,
    deleted_at        timestamptz
);

CREATE INDEX idx_activities_user_id ON activities(user_id);
CREATE INDEX idx_activities_deleted_at ON activities(deleted_at);

-- ---

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

-- ---

CREATE TABLE time_records (
    id               bigserial PRIMARY KEY,
    user_id          bigint NOT NULL,
    category         varchar(100) NOT NULL,
    description      text,
    duration_minutes bigint NOT NULL,
    created_at       timestamptz,
    updated_at       timestamptz,
    deleted_at       timestamptz
);

CREATE INDEX idx_time_records_user_id ON time_records(user_id);
CREATE INDEX idx_time_records_deleted_at ON time_records(deleted_at);

-- ---

CREATE TABLE notes (
    id         bigserial PRIMARY KEY,
    user_id    bigint NOT NULL,
    title      varchar(255) NOT NULL,
    content    text,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);

CREATE INDEX idx_notes_user_id ON notes(user_id);
CREATE INDEX idx_notes_deleted_at ON notes(deleted_at);
