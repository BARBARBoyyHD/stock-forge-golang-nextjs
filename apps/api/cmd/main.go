package main

import (
	"log"
	"net/http"
	"os"
	"stock-forge/pkg"

	"github.com/joho/godotenv"
)

func main(){

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", pkg.Root)
	mux.HandleFunc("/api/test", pkg.Test)

	port:= os.Getenv("PORT")
	addr := ":" + port

	log.Printf("http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
	
}