package manager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTitleDefinitionsUseRuntimeResourceDirectory(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"items.json", "skills.json", "npcs.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "worlds"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MUD_RESOURCES_DIR", dir)
	t.Chdir(t.TempDir())
	path := filepath.Join(dir, "titles.json")
	if err := os.WriteFile(path, []byte(`{"titles":{"runtime_flag":"运行时称号"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	titles, err := loadTitleDefinitions()
	if err != nil {
		t.Fatal(err)
	}
	if len(titles) != 1 || titles["runtime_flag"] != "运行时称号" {
		t.Fatalf("expected runtime resource titles, got %v", titles)
	}
	if err := os.WriteFile(path, []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadTitleDefinitions(); err == nil {
		t.Fatal("expected malformed titles to return an error")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := loadTitleDefinitions(); err == nil {
		t.Fatal("expected missing titles to return an error")
	}
}

// 称号系统：通关副本后，GetAttributes 返回对应称号
func TestTitleAwardedOnDungeonClear(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("titleholder", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("titleholder", "有称号的人", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	// 初始无称号
	attrs, err := r.GetAttributes(playerID)
	if err != nil {
		t.Fatalf("get attributes failed: %v", err)
	}
	titles, ok := attrs["titles"].([]string)
	if ok && len(titles) > 0 {
		t.Fatalf("expected no titles initially, got %v", titles)
	}

	// 模拟通关荒庙夜雨
	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].Flags["dungeon_cleared_abandoned_temple_rain"] = true
	players[playerID].Flags["dungeon_cleared_datang_changshengjue"] = true
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	// 应获得两个称号
	attrs, err = r.GetAttributes(playerID)
	if err != nil {
		t.Fatalf("get attributes after clear failed: %v", err)
	}
	titleList, ok := attrs["titles"].([]string)
	if !ok || len(titleList) != 2 {
		t.Fatalf("expected 2 titles, got %v", attrs["titles"])
	}
	found := map[string]bool{}
	for _, t := range titleList {
		found[t] = true
	}
	if !found["荒庙义士"] || !found["长生初悟"] {
		t.Fatalf("expected 荒庙义士 and 长生初悟, got %v", titleList)
	}
}
