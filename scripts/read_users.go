package main

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "/home/oem/Desktop/voidllm/voidllm.db")
	if err != nil {
		fmt.Println("Error opening DB:", err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, email, display_name, role, status, auth_provider FROM users WHERE deleted_at IS NULL")
	if err != nil {
		fmt.Println("Error querying users:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, email, displayName, role, status, authProvider string
		if err := rows.Scan(&id, &email, &displayName, &role, &status, &authProvider); err != nil {
			fmt.Println("Error scanning:", err)
			return
		}
		fmt.Printf("ID: %s | Email: %s | Name: %s | Role: %s | Status: %s | Provider: %s\n", id, email, displayName, role, status, authProvider)
	}
}