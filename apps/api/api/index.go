package handler

import (
	"net/http"
	"stock-forge/pkg"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	pkg.Root(w, r)
}
