package notify

import (
	"encoding/base64"
	"encoding/binary"
	"os/exec"
	"strings"
	"unicode/utf16"
)

const appID = `{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe`

func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func Script(title, body string) string {
	return strings.Join([]string{
		"[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null",
		"$t = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)",
		"$x = $t.GetElementsByTagName('text')",
		"$x.Item(0).AppendChild($t.CreateTextNode(" + quote(title) + ")) | Out-Null",
		"$x.Item(1).AppendChild($t.CreateTextNode(" + quote(body) + ")) | Out-Null",
		"[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier(" + quote(appID) + ").Show([Windows.UI.Notifications.ToastNotification]::new($t))",
	}, "\n")
}

func Encode(script string) string {
	u := utf16.Encode([]rune(script))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	return base64.StdEncoding.EncodeToString(b)
}

type Toaster struct {
	Run func(name string, args ...string) error
}

func NewToaster() Toaster {
	return Toaster{Run: func(name string, args ...string) error {
		return exec.Command(name, args...).Run()
	}}
}

func (t Toaster) Show(title, body string) error {
	return t.Run("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", Encode(Script(title, body)))
}
