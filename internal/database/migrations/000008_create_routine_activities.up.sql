CREATE TABLE routine_activities (
    routine_id  BIGINT NOT NULL REFERENCES routines(id)    ON DELETE CASCADE,
    activity_id BIGINT NOT NULL REFERENCES activities(id)  ON DELETE CASCADE,
    position    INT    NOT NULL,
    PRIMARY KEY (routine_id, activity_id)
);

CREATE INDEX idx_routine_activities_routine_position ON routine_activities (routine_id, position);
