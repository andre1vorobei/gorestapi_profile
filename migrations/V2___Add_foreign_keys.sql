ALTER TABLE subscribtions
ADD CONSTRAINT fk_follower
FOREIGN KEY (follower_id) REFERENCES user_profiles(user_id)
ON DELETE CASCADE;
