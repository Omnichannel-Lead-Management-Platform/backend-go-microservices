package main
import (
"fmt"
"github.com/aarondl/authboss/v3"
)
func main() {
	ab := authboss.New()
	fmt.Printf("Default TokenSize: %d\n", ab.Core.OneTimeTokenGenerator.TokenSize())
}
