package diff

import (
	"fmt"
	"sort"

	"github.com/harshdevelops/snagify/internal/model"
)

// diffPorts reports ports whose listening state differs between machines.
func diffPorts(a, b model.Services) []Entry {
	var entries []Entry

	ports := unionPorts(a.Ports, b.Ports)
	for _, p := range ports {
		pa, oka := a.Ports[p]
		pb, okb := b.Ports[p]
		if !oka || !okb {
			continue // port not probed on one side; skip
		}
		if pa.Listening == pb.Listening {
			continue
		}
		label := pa.Service
		if label == "" {
			label = pb.Service
		}
		name := fmt.Sprintf("%d (%s)", p, label)
		entries = append(entries, Entry{
			Category: "Port", Name: name,
			A:        listeningLabel(pa.Listening),
			B:        listeningLabel(pb.Listening),
			Severity: Difference,
		})
	}
	return entries
}

func listeningLabel(l bool) string {
	if l {
		return "running"
	}
	return "not running"
}

func unionPorts(a, b map[int]model.PortInfo) []int {
	set := map[int]struct{}{}
	for p := range a {
		set[p] = struct{}{}
	}
	for p := range b {
		set[p] = struct{}{}
	}
	out := make([]int, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Ints(out)
	return out
}
