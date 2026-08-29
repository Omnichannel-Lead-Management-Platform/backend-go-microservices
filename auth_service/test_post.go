package main
import (
"bytes"
"fmt"
"net/http"
"io"
)
func main() {
	reqBody := []byte(`{"token": "1h5IBwiIo4m2tmmFITlD_3ye0-m-XbeRygpzjATbBvpBQM8cY8aAMT8WU-sup2IfV2z538-4Bk--jmg7Le09ig==", "password": "NewSuperSecretPassword123/", "confirm_password": "NewSuperSecretPassword123/"}`)
	resp, err := http.Post("http://localhost:8080/api/auth/recover/end", "application/json", bytes.NewBuffer(reqBody))
	if err != nil { fmt.Println(err); return }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}
