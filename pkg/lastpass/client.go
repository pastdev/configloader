package lastpass

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

type Entry struct {
	Fullname        string `json:"fullname"`
	Group           string `json:"group"`
	ID              string `json:"id"`
	LastModifiedGmt string `json:"last_modified_gmt"`
	LastTouch       string `json:"last_touch"`
	Name            string `json:"name"`
	Note            string `json:"note"`
	Password        string `json:"password"`
	Share           string `json:"share"`
	URL             string `json:"url"`
	Username        string `json:"username"`
}

func (c Client) AddFuncs(funcs map[string]any) {
	funcs["lastpassFormat"] = c.GetFormat
	funcs["lastpassJSON"] = c.GetJSON
}

func (c Client) GetJSON(id string) (string, error) {
	data, err := c.Lookup(id)
	if err != nil {
		return "", err
	}

	return string(data), nil
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

	var entry []Entry
	err = json.Unmarshal(data, &entry)
	if err != nil {
		return nil, fmt.Errorf("unmarshal lastpass entry: %w", err)
	}

	return &entry[0], nil
}

func (e *Entry) Format(format string, name ...string) string {
	formatArgs := make([]any, len(name))
	for i, n := range name {
		var v any
		switch n {
		case "fullname":
			v = e.Fullname
		case "group":
			v = e.Group
		case "id":
			v = e.ID
		case "last_modified_gmt":
			v = e.LastModifiedGmt
		case "last_touch":
			v = e.LastTouch
		case "name":
			v = e.Name
		case "note":
			v = e.Note
		case "password":
			v = e.Password
		case "share":
			v = e.Share
		case "url":
			v = e.URL
		case "username":
			v = e.Username
		}
		formatArgs[i] = v
	}

	return fmt.Sprintf(format, formatArgs...)
}

func New() *Client {
	return &Client{
		Lookup: lookup,
	}
}

func lookup(id string) ([]byte, error) {
	log.Logger.Trace().Str("provider", "lastpass").Str("id", id).Msg("getJSON")
	cmd := exec.Command("lpass", "show", id, "--json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		errStr := stderr.String()
		if strings.Contains(strings.ToLower(errStr), "could not find decryption key") {
			return nil, errors.New("lpass agent not active, run `lpass login` and try again")
		}
		return nil, fmt.Errorf("run lpass: %w", err)
	}

	return stdout.Bytes(), nil
}
