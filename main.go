// Idea: We use sqlite to track down deleted file, to provide ability to
// recover them to their respective places.
package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"recycle/utils"

	_ "github.com/glebarez/go-sqlite"
	"github.com/jmoiron/sqlx"
)

const REPO_DIR = "C:/users/thed4/.recycle"
const VAR_DIR = REPO_DIR + "/var"
const DB_PATH = VAR_DIR + "/main.db"
const STORAGE_DIR = VAR_DIR + "/storage"

type PostponedFileMove struct {
	from string
	to   string
}

func store(targets []string) {
	// Queue used to postpone actual file movement to the point where all db
	// entries are correctly created.
	postponedFileMoveQueue := []PostponedFileMove{}

	cwd, e := os.Getwd()
	utils.Unwrap(e)
	tx := db.MustBegin()
	for _, target := range targets {
		// Original path must be absolute, or we won't able to restore it.
		originalPath := path.Join(cwd, target)

		deletionTime := utils.TimeNow()
		uuid := utils.Uuid4()
		destinationPath := fmt.Sprintf("%s/%s", STORAGE_DIR, uuid)
		tx.MustExec("INSERT INTO entry (uuid, original_path, deletion_time) VALUES ($1, $2, $3)", uuid, originalPath, deletionTime)
		postponedFileMoveQueue = append(
			postponedFileMoveQueue,
			PostponedFileMove{
				from: target,
				to:   destinationPath,
			},
		)
	}
	for _, postpone := range postponedFileMoveQueue {
		e = moveFile(postpone.from, postpone.to)
		utils.Unwrap(e)
	}
	tx.Commit()
}

func recover(targets []string) {
}

var schema = `
CREATE TABLE entry (
	uuid CHAR(32) NOT NULL UNIQUE,
	original_path TEXT NOT NULL,
	deletion_time INTEGER NOT NULL
)
`

var db *sqlx.DB

// https://stackoverflow.com/a/50741908/14748231
func moveFile(from string, to string) error {
	inputFile, e := os.Open(from)
	if e != nil {
		return fmt.Errorf("couldn't open source file: %v", e)
	}
	defer inputFile.Close()

	outputFile, e := os.Create(to)
	if e != nil {
		return fmt.Errorf("couldn't open dest file: %v", e)
	}
	defer outputFile.Close()

	_, e = io.Copy(outputFile, inputFile)
	if e != nil {
		return fmt.Errorf("couldn't copy to dest from source: %v", e)
	}

	inputFile.Close() // for Windows, close before trying to remove: https://stackoverflow.com/a/64943554/246801

	e = os.Remove(from)
	if e != nil {
		return fmt.Errorf("couldn't remove source file: %v", e)
	}
	return nil
}

func setupDb() {
	var e error
	db, e = sqlx.Connect(
		"sqlite",
		DB_PATH,
	)
	utils.Unwrap(e)
	db.MustExec(schema)
}

func main() {
	setupDb()

	args := os.Args
	if len(args) < 2 {
		panic("Must define command to execute.")
	}
	command := args[1]
	switch command {
	case "store":
		if len(args) < 3 {
			panic("`store` command requires at least one argument.")
		}
		targets := args[2:]
		store(targets)
	case "recover":
		if len(args) < 3 {
			panic("`store` command requires at least one argument.")
		}
		targets := args[2:]
		recover(targets)
	default:
		panic(fmt.Sprintf("Unrecognized command: %s\n", command))
	}
}
