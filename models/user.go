package models

// type User struct {
// 	ID string `json:"id"`
// 	Username string `json:"username"`
// 	Email string `json:"email"`
// 	Password string `json:"password"`
// }

type SignUp struct {
	Username string `json:"username"`
	Email string `json:"email"`
	Password string `json:"password"`
}

type Login struct {
	Email string `json:"email"`
	Password string `json:"password"`
}