package models

import "time"

type User struct {
	Id        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	HashPass  string    `json:"hash_pass"`
	Api_key   string    `json:"api_key"`
	Credits   int       `json:"credits"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserRequestData struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Pass    string `json:"pass"`
	Api_key string `json:"api_key"`
}

type UserResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// LoginBody login body
type LoginBody struct {
	Email string `json:"email"`
	Pass  string `json:"pass"`
}

// Claims : type for jwt body
