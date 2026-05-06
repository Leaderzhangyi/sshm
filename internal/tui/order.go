package tui

import (
	"sort"

	"sshm/internal/config"
)

func buildOrder(cfg *config.Config) []int {
	type connWithGroup struct {
		idx   int
		group string
		name  string
	}

	list := make([]connWithGroup, len(cfg.Connections))
	for i := range cfg.Connections {
		list[i] = connWithGroup{
			idx:   i,
			group: cfg.Connections[i].Group,
			name:  cfg.Connections[i].Name,
		}
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].group != list[j].group {
			return list[i].group < list[j].group
		}
		return list[i].name < list[j].name
	})

	order := make([]int, len(list))
	for i, item := range list {
		order[i] = item.idx
	}
	return order
}
