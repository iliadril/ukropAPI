CREATE TABLE IF NOT EXISTS recommendations
(
    id           bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    created_at   timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    user_id      bigint                      NOT NULL REFERENCES users ON DELETE CASCADE,
    artist       text                        NOT NULL,
    title        text                        NOT NULL,
    cover_url    text,
    yt_link      text,
    spotify_link text,
    comment      text,
    version      integer                     NOT NULL DEFAULT 1
);