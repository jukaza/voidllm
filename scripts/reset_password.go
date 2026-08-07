package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: go run scripts/reset_password.go <email> <new-password>")
		os.Exit(1)
	}
	email := os.Args[1]
	password := os.Args[2]

	db, err := sql.Open("sqlite", "/home/oem/Desktop/voidllm/voidllm.db")
	if err != nil {
		fmt.Println("Error opening DB:", err)
		os.Exit(1)
	}
	defer db.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error hashing password:", err)
		os.Exit(1)
	}

	res, err := db.Exec("UPDATE users SET password_hash = ?, auth_provider = 'local', updated_at = CURRENT_TIMESTAMP WHERE email = ? AND deleted_at IS NULL", string(hash), email)
	if err != nil {
		fmt.Println("Error updating password:", err)
		os.Exit(1)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		fmt.Printf("No user found with email %q\n", email)
		os.Exit(1)
	}
	fmt.Printf("Password reset for %q\n", email)
}
