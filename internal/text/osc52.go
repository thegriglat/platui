package text

import "encoding/base64"

// OSC52 возвращает escape-последовательность записи text в буфер обмена;
// tmux — оборачивает её для прохода через tmux.
func OSC52(text string, tmux bool) string {
	seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
	if tmux {
		return "\x1bPtmux;\x1b" + seq + "\x1b\\"
	}
	return seq
}
