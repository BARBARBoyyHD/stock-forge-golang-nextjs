package handler

import (
	"encoding/json"
	"net/http"
	"stock-forge/internal/jwt"
	"stock-forge/pkg"
)

type AuthHandler struct {
	TokenService *jwt.TokenService
}

type Credentials struct{
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	creds := Credentials{
		Username:"admin",
		Password:"admin",
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		pkg.JsonErrorResponse(w, 400, "invalid request body")
		return
	}

	if creds.Username != "admin" || creds.Password != "admin" {
		pkg.JsonErrorResponse(w, 401, "invalid credentials")
		return
	}

	token, err := h.TokenService.GenerateAccessToken(1, "admin@test.com", creds.Username)
	if err != nil {
		pkg.JsonErrorResponse(w, 500, "failed to generate token")
		return
	}

	expiresIn := int(h.TokenService.Config.AccessExpiry.Seconds())

	setTokenCookie(w, token, expiresIn)

	pkg.JsonSuccessResponse(w, 200, "login success", map[string]string{
		"token":    token,
		"username": creds.Username,
	})
}

func setTokenCookie(w http.ResponseWriter, accessToken string, expiresIn int ){
	http.SetCookie(w, &http.Cookie{
		Name : "sf_access_token",
		Value: accessToken,
		Path: "/",
		HttpOnly: true,
		Secure: false,
		SameSite: http.SameSiteLaxMode,
		MaxAge: expiresIn,
	})
}

func clearTokenCookie(w http.ResponseWriter){
	http.SetCookie(w, &http.Cookie{
		Name:"sf_access_token",
		Value : "",
		Path:"/",
		HttpOnly: true,
		MaxAge: -1,
	})
}