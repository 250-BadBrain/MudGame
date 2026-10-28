package manager

import (
	"strings"
	"testing"

	engruntime "mudgame/server/internal/engine/runtime"
)

// 竟陵之围全流程：三段城防拦路 → 弩机机关（选错回退重试）→ 东线独占分支 → 破招决战 → 100% 结算
func TestJinglingSiegeExclusiveBranchFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_jingling")

	room := func(id string) string { return r.dungeonMgr.InstanceRoomIDNoLock(inst, id) }
	defeatAll := func(roomID, template string, count int) {
		t.Helper()
		for i := 0; i < count; i++ {
			npcID := livingDungeonNPCByTemplateInRoom(t, inst, roomID, template)
			r.mu.Lock()
			_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(npcID)
			r.mu.Unlock()
			if !ok {
				t.Fatalf("defeat %s #%d failed", template, i+1)
			}
		}
	}

	// 南门：长矛手封路，杀完才放行
	south := room("jl_00")
	if _, err := r.Move(playerID, "north"); err == nil || !strings.Contains(err.Error(), "长矛手") {
		t.Fatalf("expected south gate blocked by spearmen, got %v", err)
	}
	defeatAll(south, "jl_spearman", 3)
	if inst.ObjectiveProgress["kill_spearmen"] != 3 {
		t.Fatalf("expected spearmen objective done, got %+v", inst.ObjectiveProgress)
	}

	// 城头：云梯锐士
	wall := room("jl_01")
	moveDungeonTestPlayer(t, r, playerID, wall)
	defeatAll(wall, "jl_ladder_trooper", 3)

	// 街巷：细作
	street := room("jl_02")
	moveDungeonTestPlayer(t, r, playerID, street)
	defeatAll(street, "jl_infiltrator", 2)

	// 机枢房：先故意选错修正（退回装填），再走对锁定
	workshop := room("jl_03")
	moveDungeonTestPlayer(t, r, playerID, workshop)
	for _, action := range []string{"load_ballista", "aim_ballista", "loose_ballista", "wrong_west"} {
		if _, err := r.DoRoomAction(playerID, action); err != nil {
			t.Fatalf("action %s failed: %v", action, err)
		}
	}
	if inst.PuzzleStates["ballista"] != "loaded" {
		t.Fatalf("expected retry back to loaded, got %s", inst.PuzzleStates["ballista"])
	}
	for _, action := range []string{"aim_ballista", "loose_ballista", "correct_east"} {
		if _, err := r.DoRoomAction(playerID, action); err != nil {
			t.Fatalf("action %s failed: %v", action, err)
		}
	}
	if inst.PuzzleStates["ballista"] != "locked" {
		t.Fatalf("expected ballista locked, got %s", inst.PuzzleStates["ballista"])
	}

	// 东线（独占分支）：击溃铁骑
	east := room("jl_04")
	moveDungeonTestPlayer(t, r, playerID, east)
	defeatAll(east, "jl_iron_cavalry", 3)

	// 决战：弩机已锁定 → 都督被破招
	marsh := room("jl_06")
	moveDungeonTestPlayer(t, r, playerID, marsh)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, marsh, "jl_vanguard_commander")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage commander failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied {
		t.Fatal("expected locked ballista to weaken vanguard commander")
	}
	if got := inst.NPCs[bossID].Attributes.Derived.Attack; got != baseAttack-10 {
		t.Fatalf("expected attack weakened by 10, got %d (base %d)", got, baseAttack)
	}

	// 结束战斗并真正击杀都督（Kill 只负责开战与破招判定）
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	// Direct objective simulation bypasses ProcessCombatTick's combat cleanup.
	finishedPlayers, _ := r.loadPlayers()
	r.clearCombatNoLock(finishedPlayers[playerID])
	_ = r.savePlayers(finishedPlayers)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat commander failed")
	}

	// 水线目标完全未做，但独占分支只计其一 → 仍应 100%
	if inst.Progress != 100 {
		t.Fatalf("expected 100%% with east branch only, got %d (progress %+v)", inst.Progress, inst.ObjectiveProgress)
	}

	// 结算：离开副本 → 奖励与通关旗标
	if _, err := leaveDungeonWithRewardCheck(t, r, playerID); err != nil {
		t.Fatalf("leave dungeon failed: %v", err)
	}
	players, _ := r.loadPlayers()
	p := players[playerID]
	// 古铜残片走概率掉落，不保证必得
	if !p.Flags["dungeon_cleared_datang_jingling"] {
		t.Fatal("expected dungeon_cleared_datang_jingling flag")
	}
}

func livingDungeonNPCByTemplateInRoom(t *testing.T, inst *engruntime.Instance, roomID, templateID string) string {
	t.Helper()
	for id, candidate := range inst.NPCs {
		if candidate.RoomID == roomID && candidate.TemplateID == templateID && candidate.Alive {
			return id
		}
	}
	t.Fatalf("missing living dungeon npc template %s", templateID)
	return ""
}
