package jwt

import "github.com/golang-jwt/jwt/v5"


type UserClaims struct{
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Username string `json:"username"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}