ALTER TABLE subscribtions
ADD CONSTRAINT fk_followee
FOREIGN KEY (followee_id) REFERENCES user_profiles(user_id)
ON DELETE CASCADE;
