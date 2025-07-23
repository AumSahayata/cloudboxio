package models

type User struct {
	ID       string
	Username string
	Password string
}

type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
}

type SignUp struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ResetPassword struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
