package main

	import "database/sql"

	type Database struct {
		db *sql.DB
	}
	func (d *Database) InitDB(db *sql.DB) {
		d.db = db
	}
	var DB = Database{}
