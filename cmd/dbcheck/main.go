package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=127.0.0.1 port=15432 user=posuser password=pospassword dbname=posdb sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping: %v", err)
	}
	fmt.Println("=== DB PING OK ===")

	// Check active/waiting sessions
	fmt.Println("\n=== ACTIVE SESSIONS ===")
	rows, err := db.Query(`SELECT pid, state, wait_event_type, wait_event, LEFT(query,80) FROM pg_stat_activity WHERE pid != pg_backend_pid() ORDER BY state`)
	if err != nil {
		log.Printf("pg_stat_activity: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var pid int
			var state, wet, we, q sql.NullString
			rows.Scan(&pid, &state, &wet, &we, &q)
			fmt.Printf("  pid=%-6d state=%-20s wait=%s/%s q=%s\n", pid, state.String, wet.String, we.String, q.String)
		}
	}

	// Check lock waits
	fmt.Println("\n=== LOCK WAITS ===")
	lockRows, err := db.Query(`SELECT blocked.pid, blocked.query, blocking.pid, blocking.query FROM pg_stat_activity blocked JOIN pg_stat_activity blocking ON blocking.pid = ANY(pg_blocking_pids(blocked.pid)) WHERE cardinality(pg_blocking_pids(blocked.pid)) > 0`)
	if err != nil {
		log.Printf("lock check: %v", err)
	} else {
		defer lockRows.Close()
		count := 0
		for lockRows.Next() {
			count++
			var bpid, kpid int
			var bq, kq string
			lockRows.Scan(&bpid, &bq, &kpid, &kq)
			fmt.Printf("  BLOCKED pid=%d q=%s\n  BY pid=%d q=%s\n", bpid, bq, kpid, kq)
		}
		if count == 0 {
			fmt.Println("  (none)")
		}
	}

	// Check schema_migrations
	fmt.Println("\n=== SCHEMA_MIGRATIONS (last 5) ===")
	mrows, err := db.Query(`SELECT filename, applied_at FROM schema_migrations ORDER BY applied_at DESC LIMIT 5`)
	if err != nil {
		log.Printf("schema_migrations: %v", err)
	} else {
		defer mrows.Close()
		for mrows.Next() {
			var fn string
			var at sql.NullString
			mrows.Scan(&fn, &at)
			fmt.Printf("  %s  %s\n", fn, at.String)
		}
	}
}
