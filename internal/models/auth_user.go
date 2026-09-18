package models

// TelegramInfo stores nested Telegram account information.
type TelegramInfo struct {
	ID       int64  `bson:"id,omitempty" json:"id,omitempty"`
	Username string `bson:"username,omitempty" json:"username,omitempty"`
}

// User represents a user record in the users collection.
type User struct {
	ID         int64         `bson:"_id" json:"id"`
	UserID     int64         `bson:"user_id,omitempty" json:"user_id,omitempty"`
	FirstName  string        `bson:"first_name,omitempty" json:"first_name,omitempty"`
	Username   string        `bson:"username,omitempty" json:"username,omitempty"`
	PhotoURL   string        `bson:"photo_url,omitempty" json:"photo_url,omitempty"`
	ProfileURL string        `bson:"profile_url,omitempty" json:"profile_url,omitempty"`
	Status     string        `bson:"status,omitempty" json:"status,omitempty"`
	Telegram   *TelegramInfo `bson:"telegram,omitempty" json:"telegram,omitempty"`
	CreatedAt  float64       `bson:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt  float64       `bson:"updated_at,omitempty" json:"updated_at,omitempty"`
}


// AuthResponse returns token and user payload upon successful authentication.
type AuthResponse struct {
	OK    bool   `json:"ok"`
	Token string `json:"token"`
	User  *User  `json:"user"`
}

// TelegramWebAppRequest contains raw initData string sent from Telegram WebApp.
type TelegramWebAppRequest struct {
	InitData string `json:"init_data"`
}

// TelegramWidgetRequest contains callback parameters from Telegram Login Widget.
type TelegramWidgetRequest struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}
