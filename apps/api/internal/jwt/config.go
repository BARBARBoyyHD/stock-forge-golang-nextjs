package jwt

import "time"

type TokenConfig struct{
	Secret 		  string
	AccessExpiry  time.Duration	
	RefreshExpiry time.Duration
	Issuer 		  string
}

func NewDefaultConfig(secret string) TokenConfig{
	return TokenConfig{
		Secret: secret,
		AccessExpiry: 15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
		Issuer: "api",
	}
}