package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"mudgame/server/internal/gamedata"
	"mudgame/server/internal/model"
)

// fileTitles 旗标→称号映射（从 titles.json 加载，flag → title name）
var (
	fileTitles     map[string]string
	fileTitlesOnce sync.Once
)

func loadTitleDefinitions() (map[string]string, error) {
	dir, err := gamedata.ResolveResourceDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "titles.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read titles %s: %w", path, err)
	}
	var raw struct {
		Titles map[string]string `json:"titles"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse titles %s: %w", path, err)
	}
	return raw.Titles, nil
}

func loadTitles() {
	fileTitlesOnce.Do(func() {
		titles, err := loadTitleDefinitions()
		if err != nil {
			fileTitles = map[string]string{}
			log.Printf("[titles] %v; titles disabled", err)
			return
		}
		fileTitles = titles
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
