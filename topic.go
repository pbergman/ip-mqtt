package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	DefaultTopic = "system/{{ hostname }}/networking/{{ interface }}/{{ protocol }}"
)

type TopicDefaultCtx struct {
	Hostname string
}

func (t TopicDefaultCtx) Get(vars url.Values) url.Values {

	var ret = url.Values{
		"hostname": []string{t.Hostname},
	}

	for k, v := range vars {
		ret[k] = v
	}

	return ret
}

type Topic struct {
	parts    []string
	partsIdx map[int]string
	defaults *TopicDefaultCtx
}

func (t Topic) GetPath(ctx url.Values) string {

	var parts []string
	var vars url.Values = t.defaults.Get(ctx)

	for i, c := 0, len(t.parts); i < c; i++ {

		if name, ok := t.partsIdx[i]; ok {

			if x := vars.Get(name); x != "" {
				parts = append(parts, x)
			}

			continue // skip no replacing parts
		}

		parts = append(parts, t.parts[i])
	}

	return strings.Join(parts, "/")
}

func getTopicsDefaults() (*TopicDefaultCtx, error) {

	hostname, err := os.Hostname()

	if err != nil {
		return nil, fmt.Errorf("could get hostsname: %s", err)
	}

	return &TopicDefaultCtx{Hostname: hostname}, nil
}

func NewTopic(path string, defaults *TopicDefaultCtx) *Topic {

	var parts = strings.Split(path, "/")
	var vars = map[int]string{}

	for i, c := 0, len(parts); i < c; i++ {

		var size = len(parts[i])

		if parts[i][0:2] == "{{" && parts[i][size-2:] == "}}" {
			vars[i] = strings.TrimSpace(parts[i][2 : size-2])
		}
	}

	return &Topic{
		parts:    parts,
		partsIdx: vars,
		defaults: defaults,
	}
}
