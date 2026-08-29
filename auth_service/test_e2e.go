package main
import (
"bytes"
"fmt"
"net/http"
"io"
)
func main() {
	reqBody1 := []byte(`{"email": "sasminiwanniarachchi@gmail.com"}`)
	resp1, _ := http.Post("http://localhost:8080/api/auth/recover", "application/json", bytes.NewBuffer(reqBody1))
	body1, _ := io.ReadAll(resp1.Body)
	fmt.Println("Recover Request:", string(body1))
}
