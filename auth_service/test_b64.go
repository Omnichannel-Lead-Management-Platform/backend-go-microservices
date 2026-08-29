package main
import (
"fmt"
"encoding/base64"
)
func main() {
	token := "1h5IBwiIo4m2tmmFITlD_3ye0-m-XbeRygpzjATbBvpBQM8cY8aAMT8WU-sup2IfV2z538-4Bk--jmg7Le09ig=="
	rawToken, _ := base64.URLEncoding.DecodeString(token)
	selectorBytes := rawToken[:32]
	verifierBytes := rawToken[32:]
	selector := base64.StdEncoding.EncodeToString(selectorBytes)
	verifier := base64.StdEncoding.EncodeToString(verifierBytes)
	fmt.Printf("Selector string: %s\nVerifier string: %s\n", selector, verifier)
}
