package utils

import (
	"os"

	"github.com/creack/pty"
)

func SetPTYWinSize(master *os.File, rows, cols uint16) error {
	return pty.Setsize(master, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}