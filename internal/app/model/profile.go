package model

type Profile struct {
	ID             int
	User_ID        int    `json:"userid"`
	UserName       string `json:"username"`
	Description    string `json:"description"`
	AvatarURL      string `json:"avatarurl"`
	Birthday       string `json:"birthday"`
	FollowersCount int    `json:"followerscount"`
	IsOwnProfile   bool   `json:"isownprofile"`
	IsFollowed     bool   `json:"isfollowed"`
}

type ShortProfile struct {
	UserName  string `json:"username"`
	AvatarURL string `json:"avatarurl"`
}
