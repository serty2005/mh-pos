package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

type maintenanceOptions struct {
	dbPath      string
	vacuum      bool
	vacuumInto  string
	optimize    bool
	checkpoint  bool
	forceVacuum bool
}

func main() {
	var options maintenanceOptions
	flag.StringVar(&options.dbPath, "db", "", "путь к SQLite базе данных")
	flag.BoolVar(&options.vacuum, "vacuum", false, "выполнить VACUUM")
	flag.StringVar(&options.vacuumInto, "vacuum-into", "", "выполнить VACUUM INTO в целевой файл")
	flag.BoolVar(&options.optimize, "optimize", false, "выполнить PRAGMA optimize")
	flag.BoolVar(&options.checkpoint, "wal-checkpoint", false, "выполнить PRAGMA wal_checkpoint(TRUNCATE)")
	flag.BoolVar(&options.forceVacuum, "force", false, "подтвердить тяжелую операцию VACUUM/VACUUM INTO")
	flag.Parse()

	if err := runMaintenance(context.Background(), options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runMaintenance(ctx context.Context, options maintenanceOptions) error {
	options.dbPath = strings.TrimSpace(options.dbPath)
	options.vacuumInto = strings.TrimSpace(options.vacuumInto)
	if options.dbPath == "" {
		return fmt.Errorf("sqlite maintenance: обязателен параметр -db")
	}
	if (options.vacuum || options.vacuumInto != "") && !options.forceVacuum {
		return fmt.Errorf("sqlite maintenance: VACUUM/VACUUM INTO требует явный параметр -force")
	}
	if !options.vacuum && options.vacuumInto == "" && !options.optimize && !options.checkpoint {
		return fmt.Errorf("sqlite maintenance: выберите минимум одну операцию")
	}

	db, err := sql.Open("sqlite", options.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if options.checkpoint {
		if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			return fmt.Errorf("sqlite maintenance: не удалось выполнить wal checkpoint: %w", err)
		}
	}
	if options.optimize {
		if _, err := db.ExecContext(ctx, `PRAGMA optimize`); err != nil {
			return fmt.Errorf("sqlite maintenance: не удалось выполнить optimize: %w", err)
		}
	}
	if options.vacuum {
		if _, err := db.ExecContext(ctx, `VACUUM`); err != nil {
			return fmt.Errorf("sqlite maintenance: не удалось выполнить vacuum: %w", err)
		}
	}
	if options.vacuumInto != "" {
		if _, err := db.ExecContext(ctx, `VACUUM INTO ?`, options.vacuumInto); err != nil {
			return fmt.Errorf("sqlite maintenance: не удалось выполнить vacuum into: %w", err)
		}
	}
	return nil
}
