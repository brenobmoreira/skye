package notify

import (
	"encoding/base64"
	"os/exec"
	"strings"
)

const appID = `{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe`

func text(s string) string {
	return "[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('" + base64.StdEncoding.EncodeToString([]byte(s)) + "'))"
}

func Script(title, body string) string {
	return strings.Join([]string{
		"[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null",
		"$t = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)",
		"$x = $t.GetElementsByTagName('text')",
		"$x.Item(0).AppendChild($t.CreateTextNode(" + text(title) + ")) | Out-Null",
		"$x.Item(1).AppendChild($t.CreateTextNode(" + text(body) + ")) | Out-Null",
		"[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('" + appID + "').Show([Windows.UI.Notifications.ToastNotification]::new($t))",
	}, "\n")
}

// Toaster shows the toast by piping the script into powershell.exe. The script goes on stdin, not
// on the command line: an encoded command (-EncodedCommand) is what endpoint security tools flag
// as obfuscated PowerShell, and the texts are already base64 inside the script.
type Toaster struct {
	Run func(stdin, name string, args ...string) error
}

func NewToaster() Toaster {
	return Toaster{Run: func(stdin, name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Stdin = strings.NewReader(stdin)
		return cmd.Run()
	}}
}

func (t Toaster) Show(title, body string) error {
	return t.Run(Script(title, body)+"\n", "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "-")
}
