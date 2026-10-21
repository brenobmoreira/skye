package resume

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SetTitle names a claude session the way /rename does: a custom-title line appended to its
// transcript, which claude --resume reads last-wins. A session without a transcript is skipped.
func SetTitle(projects, sessionID, title string) error {
	if !ValidSessionID(sessionID) {
		return fmt.Errorf("invalid session id %q", sessionID)
	}
	matches, err := filepath.Glob(filepath.Join(projects, "*", sessionID+".jsonl"))
	if err != nil || len(matches) == 0 {
		return err
	}
	line, err := json.Marshal(struct {
		Type        string `json:"type"`
		CustomTitle string `json:"customTitle"`
		SessionID   string `json:"sessionId"`
	}{"custom-title", title, sessionID})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(matches[0], os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
