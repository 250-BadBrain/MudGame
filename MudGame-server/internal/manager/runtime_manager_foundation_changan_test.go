package manager

import (
	"testing"
)

// 三本基础秘籍（踏浪诀/伏虎劲/铁壁功）装备后各自加成生效，卸下即失效
func TestFoundationManualsEquipEffects(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("foundation", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("foundation", "根基客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	// 前置基本武功各 20，三本秘籍各练到 10 级
	r.mu.Lock()
	players, _ := r.loadPlayers()
	p := players[playerID]
	p.Skills["basic_qinggong"] = 20
	p.Skills["basic_unarmed"] = 20
	p.Skills["basic_parry"] = 20
	p.Skills["talang_jue"] = 10
	p.Skills["fuhu_jin"] = 10
	p.Skills["tiebi_gong"] = 10
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	effOf := func() (dodge, initiative, attack, maxHP, parry, defense int) {
		r.mu.RLock()
		defer r.mu.RUnlock()
		eff := effectiveAttributesForPlayerNoLock(r.playersCache[playerID])
		return eff.Dodge, eff.Initiative, eff.Attack, eff.MaxHP, eff.Parry, eff.Defense
	}

	d0, i0, a0, hp0, p0, def0 := effOf()

	// 踏浪诀：躲闪 +1、先手 +2（10 级）
	if _, err := r.EquipSkill(playerID, "talang_jue"); err != nil {
		t.Fatalf("equip talang failed: %v", err)
	}
	d1, i1, _, _, _, _ := effOf()
	if d1 != d0+1 || i1 != i0+2 {
		t.Fatalf("talang expected dodge+1 initiative+2, got dodge %d->%d initiative %d->%d", d0, d1, i0, i1)
	}

	// 伏虎劲：攻击 +1（每10级）、气血上限 +6（每级，10 级共 +60）
	if _, err := r.EquipSkill(playerID, "fuhu_jin"); err != nil {
		t.Fatalf("equip fuhu failed: %v", err)
	}
	_, _, a1, hp1, _, _ := effOf()
	if a1 != a0+1 || hp1 != hp0+60 {
		t.Fatalf("fuhu expected attack+1 maxHP+60, got attack %d->%d maxHP %d->%d", a0, a1, hp0, hp1)
	}

	// 铁壁功：招架 +2、防御 +1（10 级）
	if _, err := r.EquipSkill(playerID, "tiebi_gong"); err != nil {
		t.Fatalf("equip tiebi failed: %v", err)
	}
	_, _, _, _, p1, def1 := effOf()
	if p1 != p0+2 || def1 != def0+1 {
		t.Fatalf("tiebi expected parry+2 defense+1, got parry %d->%d defense %d->%d", p0, p1, def0, def1)
	}
}

// 长安风云全流程：死士 → 听禅破招 → 逐走边不负 → 受赠小还丹 → 100% 结算
func TestChanganFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_changan")

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

	// 西市：先听茶博士递话（chan unheard→invited）
	if _, err := r.DoRoomAction(playerID, "listen_tea"); err != nil {
		t.Fatalf("listen tea failed: %v", err)
	}
	street := room("ca_01")
	moveDungeonTestPlayer(t, r, playerID, street)
	defeatAll(street, "ca_yinkui_guard", 2)
	lane := room("ca_02")
	moveDungeonTestPlayer(t, r, playerID, lane)
	defeatAll(lane, "ca_yinkui_guard", 2)

	// 和春亭：听禅 → 旗标写入 → 破招成立
	pavilion := room("ca_03")
	moveDungeonTestPlayer(t, r, playerID, pavilion)
	if _, err := r.DoRoomAction(playerID, "hear_chan"); err != nil {
		t.Fatalf("hear chan failed: %v", err)
	}
	if inst.PuzzleStates["chan"] != "purified" {
		t.Fatalf("expected chan purified, got %s", inst.PuzzleStates["chan"])
	}
	players, _ := r.loadPlayers()
	if !players[playerID].Flags["chan_purified"] {
		t.Fatal("expected chan_purified flag on player")
	}

	// 别院：先验破招再 defeat 边不负
	courtyard := room("ca_04")
	moveDungeonTestPlayer(t, r, playerID, courtyard)
	bianID := dungeonNPCIDByTemplateInRoom(t, inst, courtyard, "ca_bian_bufu")
	baseAttack := inst.NPCs[bianID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bianID); err != nil {
		t.Fatalf("engage bian failed: %v", err)
	}
	if !inst.NPCs[bianID].WeaknessApplied {
		t.Fatal("expected chan purification to weaken bian bufu")
	}
	if got := inst.NPCs[bianID].Attributes.Derived.Attack; got != baseAttack-12 {
		t.Fatalf("expected attack weakened by 12, got %d (base %d)", got, baseAttack)
	}
	r.mu.Lock()
	_, defeated, _, ok := r.dungeonMgr.DefeatNPCNoLock(bianID)
	r.mu.Unlock()
	if !ok || !defeated.Alive || !defeated.Defeated {
		t.Fatalf("bian should flee defeated but alive, got %+v", defeated)
	}
	// Kill 将玩家置入 COMBAT；defeat 直连结算不改状态——手动归位再受赠
	r.mu.Lock()
	players, _ = r.loadPlayers()
	players[playerID].State = "IDLE"
	players[playerID].TargetID = ""
	players[playerID].CombatType = ""
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()
	if inst.Progress != 100 {
		t.Fatalf("expected 100%% without optional gift, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}

	// 受赠小还丹（可选）
	if _, err := r.DoRoomAction(playerID, "accept_gift"); err != nil {
		t.Fatalf("accept gift failed: %v", err)
	}

	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave dungeon failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["xiaohuandan"] != 2 {
		t.Fatalf("expected 2 xiaohuandan, got %d", p.Backpack["xiaohuandan"])
	}
	if !p.Flags["dungeon_cleared_datang_changan"] {
		t.Fatal("expected dungeon_cleared_datang_changan flag")
	}
}
