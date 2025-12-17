CREATE TABLE IF NOT EXISTS recommendations
(
    id           bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    created_at   timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    created_by   text                        NOT NULL,
    title        text                        NOT NULL,
    yt_link      text,
    spotify_link text,
    comment      text,
    version      integer                     NOT NULL DEFAULT 1
);