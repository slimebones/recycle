// Idea: We use sqlite to track down deleted file, to provide ability to
// recover them to their respective places.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"recycle/utils"
	"regexp"
	"strings"

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
		originalPath := strings.ReplaceAll(path.Join(cwd, target), "\\", "/")

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
		cmd := exec.Command("mv", postpone.from, postpone.to)
		_, e := cmd.Output()
		utils.Unwrap(e)
	}
	tx.Commit()
}

func recover(targets []string) {
}

type Entry struct {
	Id           int        `db:"id"`
	Uuid         string     `db:"uuid"`
	OriginalPath string     `db:"original_path"`
	DeletionTime utils.Time `db:"deletion_time"`
}

// Since path.Join completely truncates structures like `../../`, they are
// not supported for now.
func list(target string) {
	// path.IsAbs incorrectly recognizes windows disks `E:/`. To address that,
	// we use an additional regex check.
	isStartedWithDisk, e := regexp.MatchString(`^[A-Z]:/.+`, target)
	utils.Unwrap(e)
	if !path.IsAbs(target) && !isStartedWithDisk {
		cwd, e := os.Getwd()
		utils.Unwrap(e)
		target = path.Join(cwd, target)
	}
	target = strings.ReplaceAll(target, "\\", "/")

	entries := []Entry{}
	isAnythingPrinted := false
	if target != "" {
		// Show all files for directory target, or the exact file for file target.
		sqlTarget := target + "%"
		e = db.Select(
			&entries, "SELECT * FROM entry WHERE original_path LIKE $1", sqlTarget)
		utils.Unwrap(e)
		for i, entry := range entries {
			fmt.Printf("%d. %s\n", i, entry.OriginalPath)
			isAnythingPrinted = true
		}
	}
	if !isAnythingPrinted {
		fmt.Printf("No entries for %s", target)
	}
}

var schema = `
CREATE TABLE entry (
	id INTEGER PRIMARY KEY,
	uuid CHAR(32) NOT NULL UNIQUE,
	original_path TEXT NOT NULL,
	deletion_time INTEGER NOT NULL
)
`

var db *sqlx.DB

func setupDb() {
	var e error
	db, e = sqlx.Connect(
		"sqlite",
		DB_PATH,
	)
	utils.Unwrap(e)
	db.Exec(schema)
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
	case "list":
		// Listing without arguments will show current directory.
		target := ""
		if len(args) == 3 {
			target = args[2]
		}
		list(target)
	default:
		panic(fmt.Sprintf("Unrecognized command: %s\n", command))
	}
}
