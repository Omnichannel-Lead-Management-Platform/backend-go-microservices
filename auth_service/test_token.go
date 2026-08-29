package main
import (
"fmt"
"github.com/aarondl/authboss/v3/crypto"
)
func main() {
	s, v, t, err := crypto.GenerateToken()
	fmt.Printf("Selector: %s\nVerifier: %s\nToken: %s\nError: %v\n", s, v, t, err)
	p1, p2, err := crypto.ParseToken(t)
	fmt.Printf("Parsed: %s, %s, %v\n", p1, p2, err)
}
