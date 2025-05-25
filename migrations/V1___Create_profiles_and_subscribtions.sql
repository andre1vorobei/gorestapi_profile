CREATE TABLE user_profiles (
    id bigserial not null primary key,
    user_id bigserial not null unique,
    username varchar not null  unique,
    birthday_date date,
    description text,
    avatar_url text
);

CREATE TABLE subscribtions (
    follower_id bigserial not null,
    followee_id bigserial not null, 
    primary key (follower_id, followee_id)
);
