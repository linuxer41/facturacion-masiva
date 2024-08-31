package main

import (
	"app/db"
	"app/ui"
)

func main() {
	connString := "server=localhost;database=mydb;user id=myuser;password=mypassword"
	db.InitDB(connString)
	ui.SetupUI()
}