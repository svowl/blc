package main

// Helper utility to add/update users list
//
// Format:
// blcusers <file path> <login> <password>
//
// Example:
// blcusers ./users.json email@example.com 12345

import (
	"fmt"
	"os"

	"blc/pkg/auth"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Correct format: %s <file path> <login> <password>", os.Args[0])
		os.Exit(0)
	}
	a := auth.New(os.Args[1], 60)
	if err := a.AddUser(os.Args[2], os.Args[3]); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("User %s has been successfully created/updated", os.Args[2])
}
