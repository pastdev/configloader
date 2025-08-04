package bitwarden

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pastdev/configloader/pkg/log"
)

type Client struct {
	Lookup func(id string) ([]byte, error)
}

type Data struct {
	Password string `json:"password"`
	Totp     string `json:"totp"`
	Uris     []URI  `json:"uris"`
	Username string `json:"username"`
}

type Entry struct {
	Data    Data      `json:"data"`
	Fields  []Field   `json:"fields"`
	Folder  string    `json:"folder"`
	History []History `json:"history"`
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Notes   string    `json:"notes"`
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

type URI struct {
	MatchType int    `json:"match_type"`
	URI       string `json:"uri"`
}

func (c Client) AddFuncs(funcs map[string]any) {
	funcs["bitwardenField"] = c.GetField
	funcs["bitwardenFormat"] = c.GetFormat
	funcs["bitwardenJSON"] = c.GetJSON
}

func (c Client) GetJSON(id string) (string, error) {
	data, err := c.Lookup(id)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (c Client) GetField(id string, name string) (string, error) {
	entry, err := c.unmarshal(id)
	if err != nil {
		return "", err
	}

	for _, field := range entry.Fields {
		if field.Name == name {
			return field.Value, nil
		}
	}

	return "", nil
}

func (c Client) GetFormat(id string, format string, name ...string) (string, error) {
	entry, err := c.unmarshal(id)
	if err != nil {
		return "", err
	}

	return entry.Format(format, name...), nil
}

func (c Client) unmarshal(id string) (*Entry, error) {
	data, err := c.Lookup(id)
	if err != nil {
		return nil, err
	}

	var entry Entry
	err = json.Unmarshal(data, &entry)
	if err != nil {
		return nil, fmt.Errorf("unmarshal rbw entry: %w", err)
	}

	return &entry, nil
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

func New() *Client {
	return &Client{Lookup: lookup}
}

func lookup(id string) ([]byte, error) {
	log.Logger.Trace().Str("provider", "bitwarden").Str("id", id).Msg("getJSON")
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
