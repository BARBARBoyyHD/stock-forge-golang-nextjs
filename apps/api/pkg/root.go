package pkg

import (
	"fmt"
	"net/http"	
)

func Root (w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func Test (w http.ResponseWriter, r *http.Request) {
	JsonSuccessResponse(w,200,"Hello this is Test",nil)
}