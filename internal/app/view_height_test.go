package app

import (
	"strings"
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestView_FillsTerminalHeight(t *testing.T) {
	for _, mode := range []model.ViewMode{model.ModeExplorer, model.ModePolicyList} {
		for _, tabs := range []int{1, 2} {
			m := newTestModel()
			m.width, m.height = 120, 30
			m.mode = mode
			m.entries = []model.Entry{{Name: "a/", IsDir: true}}
			for len(m.tabs) < tabs {
				m.tabs = append(m.tabs, m.tabs[0])
			}

			lines := strings.Count(m.View(), "\n") + 1
			assert.Equal(t, m.height, lines, "mode=%d tabs=%d", mode, tabs)
		}
	}
}
