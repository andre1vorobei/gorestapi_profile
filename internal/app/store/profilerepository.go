package store

import (
	"fmt"
	"gorestapi/internal/app/model"
)

type ProfileRepository struct {
	store *Store
}

func (r *ProfileRepository) CreateProfile(p *model.Profile) (*model.Profile, error) {
	if err := r.store.db.QueryRow(
		"INSERT INTO user_profiles (user_id, username, birthday_date, description, avatar_url) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		p.User_ID, p.UserName, p.Birthday, p.Description, p.AvatarURL,
	).Scan(&p.ID); err != nil {
		return nil, err
	}

	return p, nil
}

func (r *ProfileRepository) GetFollowees(userName string) ([]model.ShortProfile, error) {

	p, err := r.FindProfileByUsername(userName)

	if err != nil {
		return nil, err
	}

	userID := p.User_ID

	rows, err := r.store.db.Query(
		"SELECT p.username, p.avatar_url FROM subscribtions s JOIN user_profiles p ON s.followee_id = p.user_id WHERE s.follower_id = $1", userID)

	if err != nil {
		return nil, err
	}

	var followees []model.ShortProfile
	for rows.Next() {
		var sp model.ShortProfile
		if err := rows.Scan(&sp.UserName, &sp.AvatarURL); err != nil {
			return nil, err
		}
		followees = append(followees, sp)
	}

	return followees, nil

}
func (r *ProfileRepository) GetFollowers(userName string) ([]model.ShortProfile, error) {

	p, err := r.FindProfileByUsername(userName)

	if err != nil {
		return nil, err
	}

	userID := p.User_ID

	rows, err := r.store.db.Query(
		"SELECT p.username, p.avatar_url FROM subscribtions s JOIN user_profiles p ON s.follower_id = p.user_id WHERE s.followee_id = $1", userID)

	if err != nil {
		return nil, err
	}

	var followers []model.ShortProfile
	for rows.Next() {
		var sp model.ShortProfile
		if err := rows.Scan(&sp.UserName, &sp.AvatarURL); err != nil {
			return nil, err
		}
		followers = append(followers, sp)
	}

	return followers, nil

}

func (r *ProfileRepository) DeleteProfileByID(profID int) error {
	if _, err := r.store.db.Exec("DELETE FROM user_profiles WHERE id = $1", profID); err != nil {
		return err
	}
	return nil
}

func (r *ProfileRepository) IsFollow(follower_id int, followee_id int) (bool, error) {
	var exists bool
	err := r.store.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM subscribtions WHERE follower_id = $1 AND followee_id = $2)",
		follower_id, followee_id,
	).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to check subscription: %w", err)
	}

	return exists, nil
}

func (r *ProfileRepository) FindProfileByPattern(pattern string) ([]model.ShortProfile, error) {

	pattern = "%" + pattern + "%"
	rows, err := r.store.db.Query(
		"SELECT username, avatar_url FROM user_profiles WHERE username LIKE $1", pattern)

	if err != nil {
		return nil, err
	}

	var profiles []model.ShortProfile
	for rows.Next() {
		var sp model.ShortProfile
		if err := rows.Scan(&sp.UserName, &sp.AvatarURL); err != nil {
			return nil, err
		}
		profiles = append(profiles, sp)
	}

	return profiles, nil
}

func (r *ProfileRepository) FindProfileByUsername(username string) (*model.Profile, error) {
	p := &model.Profile{}
	if err := r.store.db.QueryRow(
		"SELECT id, user_id, username, birthday_date, description, avatar_url, followers_count FROM user_profiles WHERE username = $1",
		username,
	).Scan(&p.ID, &p.User_ID, &p.UserName, &p.Birthday, &p.Description, &p.AvatarURL, &p.FollowersCount); err != nil {
		return nil, err
	}

	return p, nil
}

func (r *ProfileRepository) FindProfileByID(userID int) (*model.Profile, error) {
	p := &model.Profile{}
	if err := r.store.db.QueryRow(
		"SELECT id, user_id, username, birthday_date, description, avatar_url, followers_count FROM user_profiles WHERE user_id = $1",
		userID,
	).Scan(&p.ID, &p.User_ID, &p.UserName, &p.Birthday, &p.Description, &p.AvatarURL, &p.FollowersCount); err != nil {
		return nil, err
	}

	return p, nil
}
func (r *ProfileRepository) Subscribe(fwerID int, fweeUName string) error {
	var fweeID int

	if err := r.store.db.QueryRow(
		"SELECT user_id FROM user_profiles WHERE username = $1", fweeUName).Scan(&fweeID); err != nil {
		return err
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec(
		"INSERT INTO subscribtions (follower_id, followee_id) VALUES ($1, $2)",
		fwerID, fweeID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert subscription: %w", err)
	}

	_, err = tx.Exec(
		"UPDATE user_profiles SET followers_count = followers_count + 1 WHERE user_id = $1",
		fweeID,
	)
	if err != nil {
		return fmt.Errorf("failed to update followers count: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *ProfileRepository) Unsubscribe(fwerID int, fweeUName string) error {
	var fweeID int

	if err := r.store.db.QueryRow(
		"SELECT user_id FROM user_profiles WHERE username = $1", fweeUName).Scan(&fweeID); err != nil {
		return err
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec(
		"DELETE FROM subscribtions WHERE follower_id = $1 AND followee_id = $2",
		fwerID, fweeID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	_, err = tx.Exec(
		"UPDATE user_profiles SET followers_count = followers_count - 1 WHERE user_id = $1",
		fweeID,
	)
	if err != nil {
		return fmt.Errorf("failed to update followers count: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
