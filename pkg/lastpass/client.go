package lastpass

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

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

func Show(id string, format string, name ...string) (string, error) {
	cmd := exec.Command("lpass", "show", id, "--json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("run lpass: %w", err)
	}

	var entry []Entry
	err = json.NewDecoder(&stdout).Decode(&entry)
	if err != nil {
		return "", fmt.Errorf("unmarshal lpass entry: %w", err)
	}

	formatArgs := make([]any, len(name))
	for i, n := range name {
		var v any
		switch n {
		case "fullname":
			v = entry[0].Fullname
		case "group":
			v = entry[0].Group
		case "id":
			v = entry[0].ID
		case "last_modified_gmt":
			v = entry[0].LastModifiedGmt
		case "last_touch":
			v = entry[0].LastTouch
		case "name":
			v = entry[0].Name
		case "note":
			v = entry[0].Note
		case "password":
			v = entry[0].Password
		case "share":
			v = entry[0].Share
		case "url":
			v = entry[0].URL
		case "username":
			v = entry[0].Username
		}
		formatArgs[i] = v
	}

	return fmt.Sprintf(format, formatArgs...), nil
}
