// Idea: We use sqlite to track down deleted file, to provide ability to
// recover them to their respective places.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"recycle/utils"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/glebarez/go-sqlite"
	"github.com/jmoiron/sqlx"
)

var REPO_DIR = mustGetenv("HOME") + "/.recycle"
var VAR_DIR = REPO_DIR + "/var"
var DB_PATH = VAR_DIR + "/main.db"
var STORAGE_DIR = VAR_DIR + "/storage"

type PostponedFileMove struct {
	from string
	to   string
}

var cwd string

func mustGetenv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		panic(fmt.Sprintf("No env %s", key))
	}
	return value
}

func store(targets []string) {
	// Queue used to postpone actual file movement to the point where all db
	// entries are correctly created.
	postponedFileMoveQueue := []PostponedFileMove{}

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
		movePath(postpone.from, postpone.to)
	}
	tx.Commit()
}

func movePath(from string, to string) {
	cmd := exec.Command("mv", from, to)
	_, e := cmd.Output()
	utils.Unwrap(e)
}

// https://stackoverflow.com/a/12527546/14748231
func isPathExists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}

func recover(id int) {
	// Recover only works in the cwd.
	entries := list("", false)
	for _, entry := range entries {
		if entry.Id == id {
			if isPathExists(entry.OriginalPath) {
				panic(fmt.Sprintf(
					"Cannot recover path %s, already exists.",
					entry.OriginalPath))
			}
			tx := db.MustBegin()
			tx.MustExec("DELETE FROM entry WHERE id = $1", id)
			movePath(STORAGE_DIR+"/"+entry.Uuid, entry.OriginalPath)
			tx.Commit()
			return
		}
	}
	panic(fmt.Sprintf("Cannot find entry with id %d in directory %s", id, cwd))
}

type Entry struct {
	Id           int        `db:"id"`
	Uuid         string     `db:"uuid"`
	OriginalPath string     `db:"original_path"`
	DeletionTime utils.Time `db:"deletion_time"`
}

// Since path.Join completely truncates structures like `../../`, they are
// not supported for now.
func list(target string, shouldPrint bool) []Entry {
	// path.IsAbs incorrectly recognizes windows disks `E:/`. To address that,
	// we use an additional regex check.
	isStartedWithDisk, e := regexp.MatchString(`[A-Z]:/.*`, target)
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
		if shouldPrint {
			for _, entry := range entries {
				fmt.Printf("%d. %s (%s)\n", entry.Id, entry.OriginalPath, utils.TimeToDate(entry.DeletionTime))
				isAnythingPrinted = true
			}
		}
	}
	if !isAnythingPrinted && shouldPrint {
		fmt.Printf("No entries for %s", target)
	}
	return entries
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
	var e error
	cwd, e = os.Getwd()
	utils.Unwrap(e)
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
		if len(args) != 3 {
			panic("`recover` command requires one argument: id. To find out id of item you want to recover, call `list` command for the directory.")
		}
		id, e := strconv.Atoi(args[2])
		utils.Unwrap(e)
		recover(id)
	case "list":
		// Listing without arguments will show current directory.
		target := ""
		if len(args) == 3 {
			target = args[2]
		}
		list(target, true)
	default:
		panic(fmt.Sprintf("Unrecognized command: %s\n", command))
	}
}
