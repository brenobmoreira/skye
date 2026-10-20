package app

import (
	"fmt"

	"github.com/brenobmoreira/skye/internal/files"
)

// fallbackBases are tried, in order, after the [[repo]] base when a terminal's branch is compared
// with where it started.
var fallbackBases = []string{"origin/develop", "develop", "origin/main", "main", "origin/master", "master"}

// root is the folder the files panel shows for a terminal: the git work tree it is in, or its
// folder.
func (a *App) root(id string) (string, error) {
	t, err := a.get(id)
	if err != nil {
		return "", err
	}
	cwd := t.Cwd
	if cwd == "" {
		cwd = a.o.Home
	}
	root, _ := files.Root(cwd)
	return root, nil
}

// bases puts first the base configured for the repo the root belongs to, worktrees included.
func (a *App) bases(root string) []string {
	var list []string
	if common, ok := files.CommonDir(root); ok {
		for _, r := range a.o.Config.Repos {
			if r.Base == "" || r.Base == "HEAD" {
				continue
			}
			if c, ok := files.CommonDir(r.Resolved(a.o.Home).Path); ok && c == common {
				list = append(list, r.Base)
			}
		}
	}
	return append(list, fallbackBases...)
}

func mode(m string) (files.Mode, error) {
	switch files.Mode(m) {
	case files.Work, files.Branch:
		return files.Mode(m), nil
	}
	return "", fmt.Errorf("modo de comparação desconhecido: %q", m)
}

func (a *App) Changes(id, m string) (files.Changes, error) {
	md, err := mode(m)
	if err != nil {
		return files.Changes{}, err
	}
	root, err := a.root(id)
	if err != nil {
		return files.Changes{}, err
	}
	return files.ListChanges(root, md, a.bases(root))
}

func (a *App) Diff(id, path, m string) (files.Diff, error) {
	md, err := mode(m)
	if err != nil {
		return files.Diff{}, err
	}
	root, err := a.root(id)
	if err != nil {
		return files.Diff{}, err
	}
	return files.DiffOf(root, path, md, a.bases(root))
}

func (a *App) ListDir(id, path string) ([]files.Entry, error) {
	root, err := a.root(id)
	if err != nil {
		return nil, err
	}
	return files.List(root, path)
}

func (a *App) ReadFile(id, path string) (files.File, error) {
	root, err := a.root(id)
	if err != nil {
		return files.File{}, err
	}
	return files.Read(root, path)
}

func (a *App) WriteFile(id, path, content, version string) (files.File, error) {
	root, err := a.root(id)
	if err != nil {
		return files.File{}, err
	}
	return files.Write(root, path, content, version)
}

// OpenInEditor opens the terminal's folder in VS Code, on the file when path is set.
func (a *App) OpenInEditor(id, path string) error {
	root, err := a.root(id)
	if err != nil {
		return err
	}
	open := a.o.OpenEditor
	if open == nil {
		open = files.OpenInEditor
	}
	return open(root, path)
}
