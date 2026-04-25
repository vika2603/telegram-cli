// Command tg is the Telegram CLI entry point. The real Main lives in
// internal/program so it can be unit-tested.
package main

import (
	"os"

	"github.com/vika2603/telegram-cli/internal/program"
)

func main() { os.Exit(program.Main()) }
