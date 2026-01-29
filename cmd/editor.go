/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"os"
	"os/exec"
)

func getEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}
	return editor
}

func openEditor() (string, error) {
	return openEditorWithContent("")
}

func openEditorWithContent(initial string) (string, error) {
	editor := getEditor()

	tmpFile, err := os.CreateTemp("", "noted-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if initial != "" {
		if _, err := tmpFile.WriteString(initial); err != nil {
			tmpFile.Close()
			return "", err
		}
	}
	tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}

	return string(content), nil
}
