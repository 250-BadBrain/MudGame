package manager

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"mudgame/server/internal/model"
)

// fileTitles 旗标→称号映射（从 titles.json 加载，flag → title name）
var (
	fileTitles     map[string]string
	fileTitlesOnce sync.Once
)

func titlesPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "resources", "titles.json")
}

func loadTitles() {
	fileTitlesOnce.Do(func() {
		fileTitles = map[string]string{}
		data, err := os.ReadFile(titlesPath())
		if err != nil {
			log.Printf("[titles] titles.json not found at %s, titles disabled", titlesPath())
			return
		}
		var raw struct {
			Titles map[string]string `json:"titles"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			log.Printf("[titles] failed to parse titles.json: %v", err)
			return
		}
		fileTitles = raw.Titles
		log.Printf("[titles] loaded %d titles", len(fileTitles))
	})
}

// getPlayerTitles 返回玩家已获得的称号列表（基于通关旗标）。
func getPlayerTitles(p *model.Player) []string {
	loadTitles()
	if p == nil || len(fileTitles) == 0 {
		return nil
	}
	titles := make([]string, 0, len(fileTitles))
	for flag, title := range fileTitles {
		if p.Flags[flag] {
			titles = append(titles, title)
		}
	}
	if len(titles) == 0 {
		return nil
	}
	return titles
}

