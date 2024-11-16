package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Id = int64

// Basic time defined in milliseconds.
type Time = int64

// We consider standard time.Time as a Date, other mentions of Time are ms
// timestamps.
type Date = time.Time
type Dict = map[string]any

func Unwrap(err error) {
	if err != nil {
		panic(err)
	}
}

func Print(obj ...any) {
	fmt.Println(obj...)
}

// Reference: https://stackoverflow.com/a/13295158/14748231
func TimeToDate(t Time) Date {
	return time.Unix(0, t*int64(time.Millisecond))
}

func TimeNow() Time {
	return DateNow().UnixMilli()
}

func DateNow() Date {
	return time.Now()
}

func Uuid4() string {
	r := uuid.New()
	return strings.ReplaceAll(r.String(), "-", "")
}
