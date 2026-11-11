package pomodoro

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/brenobmoreira/skye/internal/atomicfile"
)

// Settings are the lengths in minutes, and how many focuses come before a long break.
type Settings struct {
	Focus int `json:"focus"`
	Break int `json:"break"`
	Long  int `json:"long"`
	Every int `json:"every"`
}

var Default = Settings{Focus: 25, Break: 5, Long: 15, Every: 4}

func (s Settings) Validate() error {
	for _, m := range []struct {
		name string
		v    int
	}{{"foco", s.Focus}, {"pausa", s.Break}, {"pausa longa", s.Long}} {
		if m.v < 1 || m.v > 180 {
			return fmt.Errorf("%s: de 1 a 180 minutos", m.name)
		}
	}
	if s.Every < 1 || s.Every > 12 {
		return errors.New("pausa longa a cada: de 1 a 12 focos")
	}
	return nil
}

// LoadSettings reads the file; missing is Default, broken or invalid is Default and an error.
func LoadSettings(file string) (Settings, error) {
	data, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return Default, nil
	}
	if err != nil {
		return Default, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Default, fmt.Errorf("%s: %w", file, err)
	}
	if err := s.Validate(); err != nil {
		return Default, fmt.Errorf("%s: %w", file, err)
	}
	return s, nil
}

func SaveSettings(file string, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Write(file, data, 0o600)
}
