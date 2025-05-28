// This file is deprecated. Use cmd/distributed-app/main.go instead.
// 
// To run the application:
// go run cmd/distributed-app/main.go
//
// Or build it:
// go build -o distributed-app cmd/distributed-app/main.go

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("This main.go is deprecated.")
	fmt.Println("Please use: go run cmd/distributed-app/main.go")
	fmt.Println("Or build with: go build -o distributed-app cmd/distributed-app/main.go")
	os.Exit(1)
}