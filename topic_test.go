package main

import (
	"net/url"
	"strconv"
	"testing"
)

func TestTopic_Section_Protocols(t *testing.T) {

	for idx, protocol := range []string{"ipv4", "ipv6"} {

		var topic = NewTopic("system/{{ protocol}}/{{interface }}/{{ hostname }}/{{ not_existing }}", &TopicDefaultCtx{Hostname: "foo"})
		var test = map[int]string{1: "protocol", 2: "interface", 3: "hostname"}
		var interf = "etc" + strconv.Itoa(idx)

		var ctx = url.Values{
			"protocol":  []string{protocol},
			"interface": []string{interf},
			"hostname":  []string{"bar"},
		}

		for idx, name := range test {
			if x, ok := topic.partsIdx[idx]; !ok || x != name {
				t.Errorf("expecting %s with index %d in vars got %#v", name, idx, topic.partsIdx)
			}
		}

		var expecting = "system/" + protocol + "/" + interf + "/bar"

		if topic.GetPath(ctx) != expecting {
			t.Errorf("expecting topic \"%s\" got \"%s\"", expecting, topic.GetPath(ctx))
		}
	}
}
