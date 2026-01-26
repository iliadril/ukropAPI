CREATE TABLE IF NOT EXISTS reservations
(
    id                    bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    created_at            timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    user_id               bigint                      NOT NULL REFERENCES users ON DELETE CASCADE,
    title                 text                        NOT NULL,
    description           text,
    start_time            timestamp(0) with time zone NOT NULL,
    end_time              timestamp(0) with time zone NOT NULL,
    color                 text,
    parent_reservation_id bigint REFERENCES reservations ON DELETE CASCADE,
    version               integer                     NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS reservations_user_id_idx ON reservations (user_id);
