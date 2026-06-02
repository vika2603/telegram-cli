package output

import (
	"strconv"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// PollOption is one answer of a poll, with current tallies when available.
type PollOption struct {
	Text    string `json:"text"`
	Voters  int    `json:"voters,omitempty"`
	Chosen  bool   `json:"chosen,omitempty"`  // the account voted for this option
	Correct bool   `json:"correct,omitempty"` // quiz: this is the correct answer
}

// PollInfo describes a poll message: its question, options, and tallies.
type PollInfo struct {
	Question    string       `json:"question"`
	Options     []PollOption `json:"options"`
	TotalVoters int          `json:"total_voters,omitempty"`
	Multiple    bool         `json:"multiple,omitempty"`
	Quiz        bool         `json:"quiz,omitempty"`
	Public      bool         `json:"public,omitempty"`
	Closed      bool         `json:"closed,omitempty"`
}

// RenderPoll prints a poll as a numbered list with tallies.
func RenderPoll(io *ui.IOStreams, p PollInfo) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("#", "OPTION", "VOTES", "FLAGS")
	for i, o := range p.Options {
		flags := ""
		if o.Chosen {
			flags += "voted"
		}
		if o.Correct {
			if flags != "" {
				flags += ","
			}
			flags += "correct"
		}
		tp.AddRow(strconv.Itoa(i+1), o.Text, strconv.Itoa(o.Voters), flags)
	}
	return tp.Render()
}
