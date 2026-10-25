package manager

import (
	"testing"
)

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
