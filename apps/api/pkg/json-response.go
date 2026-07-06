package pkg

import (
	"net/http"
	"encoding/json"
)

type JsonResponse struct{
	Status int `json:"status"`
	Message string `json:"message"`
	Data any `json:"data,omitempty"`
}

func JsonSuccessResponse(w http.ResponseWriter, status int, message string, data any){
	res := JsonResponse{
		Status: status,
		Message: message,
		Data: data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(res)

}

func JsonErrorResponse(w http.ResponseWriter, status int, message string){
	res := JsonResponse{
		Status: status,
		Message: message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(res)
}