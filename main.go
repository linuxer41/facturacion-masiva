package main

import (
	"app/db"
	"app/ui"
	"flag"
	"fmt"
)

func main() {
	// Define the default connection string
	defaultConnString := "server=localhost;database=EMPSAAT;user id=sa;password=Anarkia41?!"
	// connString := "sqlserver://sa:Anarkia41%3F%21@localhost:1433?database=EMPSAAT&connection+timeout=30"

	// Define a flag for the connection string
	connStringPtr := flag.String("connString", defaultConnString, "SQL Server connection string")

	// Parse the command-line flags
	flag.Parse()

	// Use the provided connection string or the default one
	connString := *connStringPtr

	fmt.Printf("Using connection string: %s\n", connString)

	// Initialize the database connection
	db.InitDB(connString)

	// Set up the UI
	ui.SetupUI()
}