package main

import (
	"log"
	"net/http"
	"os"
	"stock-forge/internal/handler"
	"stock-forge/internal/jwt"
	"stock-forge/internal/middleware"
	"stock-forge/pkg"

	"github.com/joho/godotenv"
)

func main(){

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal("Error loading .env file: ", err)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "test-secret"
	}
	tokenSvc := jwt.New(jwt.NewDefaultConfig(secret))
	authHandler := &handler.AuthHandler{TokenService: tokenSvc}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", pkg.Root)
	mux.HandleFunc("POST /api/auth", authHandler.HandleAuth)

	authMW := middleware.JWTMiddleware(tokenSvc)
	mux.Handle("GET /api/test", authMW(http.HandlerFunc(pkg.Test)))
	mux.Handle("GET /api/morning", authMW(http.HandlerFunc(pkg.Morning)))

	port:= os.Getenv("PORT")
	addr := ":" + port

	log.Printf("http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
	
}