package bitwarden

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Entry struct {
	Data    Data      `json:"data"`
	Fields  []Field   `json:"fields"`
	Folder  string    `json:"folder"`
	History []History `json:"history"`
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Notes   string    `json:"notes"`
}

type Data struct {
	Password string `json:"password"`
	Totp     string `json:"totp"`
	Uris     []URI  `json:"uris"`
	Username string `json:"username"`
}

type URI struct {
	MatchType int    `json:"match_type"`
	URI       string `json:"uri"`
}

type Field struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type History struct {
	LastUsedDate string `json:"last_used_date"`
	Password     string `json:"password"`
}

func (e *Entry) Format(format string, name ...string) string {
	formatArgs := make([]any, len(name))
	for i, n := range name {
		var v any
		switch n {
		case "folder":
			v = e.Folder
		case "id":
			v = e.ID
		case "name":
			v = e.Name
		case "notes":
			v = e.Notes
		case "password":
			v = e.Data.Password
		case "username":
			v = e.Data.Username
		}
		formatArgs[i] = v
	}

	return fmt.Sprintf(format, formatArgs...)
}

func getJSON(id string) ([]byte, error) {
	cmd := exec.Command("rbw", "get", id, "--raw")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		errStr := stderr.String()
		if strings.Contains(strings.ToLower(errStr), "failed to read password from pinentry") {
			return nil, errors.New("rbw agent not active, run `rbw unlock` and try again")
		}
		return nil, fmt.Errorf("run rbw (%s): %w", stderr.String(), err)
	}

	return stdout.Bytes(), nil
}

func GetJSON(id string) (string, error) {
	data, err := getJSON(id)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func GetFormat(id string, format string, name ...string) (string, error) {
	data, err := getJSON(id)
	if err != nil {
		return "", err
	}

	var entry Entry
	err = json.Unmarshal(data, &entry)
	if err != nil {
		return "", fmt.Errorf("unmarshal rbw entry: %w", err)
	}

	formatArgs := make([]any, len(name))
	for i, n := range name {
		var v any
		switch n {
		case "folder":
			v = entry.Folder
		case "id":
			v = entry.ID
		case "name":
			v = entry.Name
		case "notes":
			v = entry.Notes
		case "password":
			v = entry.Data.Password
		case "username":
			v = entry.Data.Username
		}
		formatArgs[i] = v
	}

	return fmt.Sprintf(format, formatArgs...), nil
}
