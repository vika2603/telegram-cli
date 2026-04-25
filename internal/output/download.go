package output

import "github.com/vika2603/telegram-cli/internal/ui"

// DownloadRow is emitted by `msg download`.
type DownloadRow struct {
	MessageRef string `json:"message_ref"`
	Path       string `json:"path"`
	Name       string `json:"name,omitempty"`
	MIMEType   string `json:"mime_type,omitempty"`
	Bytes      int64  `json:"bytes,omitempty"`
}

// RenderDownload prints the saved media path and metadata.
func RenderDownload(io *ui.IOStreams, row DownloadRow) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("MESSAGE", "PATH", "TYPE", "BYTES")
	tp.AddRow(row.MessageRef, row.Path, row.MIMEType, i64toaOrBlank(row.Bytes))
	return tp.Render()
}
