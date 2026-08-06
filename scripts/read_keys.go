package main

import (
"database/sql"
"fmt"
_ "modernc.org/sqlite"
)

func main() {
db, err := sql.Open("sqlite", "/home/oem/Desktop/voidllm/voidllm.db")
if err != nil {
tln("Error opening DB:", err)

}
defer db.Close()

rows, err := db.Query("SELECT id, key_hint, key_type, name, user_id, created_by FROM api_keys WHERE deleted_at IS NULL")
if err != nil {
tln("Error querying keys:", err)

}
defer rows.Close()

for rows.Next() {
t, keyType, name, userID, createdBy string
(&id, &keyHint, &keyType, &name, &userID, &createdBy); err != nil {
tln("Error scanning:", err)

tf("ID: %s | Hint: %s | Type: %s | Name: %s | UserID: %s | CreatedBy: %s\n", id, keyHint, keyType, name, userID, createdBy)
}
}
