package manager

import (
	"testing"
)

// 秘籍装备规则：同槽可学多本但只能装备一本，仅装备中的秘籍参与属性加成，按秘籍自身等级计算
func TestManualEquipSlotSemantics(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("duelist", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("duelist", "双修刀客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	// 基本刀法 40（满足天刀八诀门槛）+ 两本刀系秘籍各练到 10 级
	r.mu.Lock()
	players, _ := r.loadPlayers()
	p := players[playerID]
	p.Skills["basic_blade"] = 40
	p.Skills["jyg_zhanshentu_dao"] = 10 // 攻+3/10 → 10级 +3
	p.Skills["tiandao_blade"] = 10      // 攻+2/10 → 10级 +2
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	basicAttack := func() int {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return effectiveAttributesForPlayerNoLock(r.playersCache[playerID]).Attack
	}

	// 未装备任何秘籍：只有基本刀法的加成
	attackNone := basicAttack()

	// 装备战神图录 → 攻击 +3
	if _, err := r.EquipSkill(playerID, "jyg_zhanshentu_dao"); err != nil {
		t.Fatalf("equip zhanshentu failed: %v", err)
	}
	attackZS := basicAttack()
	if attackZS != attackNone+3 {
		t.Fatalf("expected zhanshentu to add 3 attack, got %d -> %d", attackNone, attackZS)
	}

	// 换装天刀八诀（同槽择一）→ 攻击 +2，战神图录加成消失
	if _, err := r.EquipSkill(playerID, "tiandao_blade"); err != nil {
		t.Fatalf("equip tiandao failed: %v", err)
	}
	attackTD := basicAttack()
	if attackTD != attackNone+2 {
		t.Fatalf("expected tiandao to add 2 attack (zhanshentu unequipped), got %d -> %d", attackNone, attackTD)
	}

	// 装备状态与持久化
	after, _ := r.loadPlayers()
	if after[playerID].EquippedSkills["basic_blade"] != "tiandao_blade" {
		t.Fatalf("expected tiandao equipped on basic_blade, got %+v", after[playerID].EquippedSkills)
	}
}

// 秘籍前置门槛：前置基本武功等级不足时学习被拒
func TestManualPrerequisiteGate(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("bladeless", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("bladeless", "无刀客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].RoomID = "ly_old_alley"
	players[playerID].Skills["basic_blade"] = 39 // 天刀八诀要求 40
	players[playerID].Skills["tiandao_blade"] = 1
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	if err := r.LearnSkill(playerID, "ancient_scroll_keeper", "tiandao_blade"); err == nil {
		t.Fatal("learning tiandao with basic_blade 39 must fail")
	}

	r.mu.Lock()
	players, _ = r.loadPlayers()
	players[playerID].Skills["basic_blade"] = 40
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()
	if err := r.LearnSkill(playerID, "ancient_scroll_keeper", "tiandao_blade"); err != nil {
		t.Fatalf("learning tiandao with basic_blade 40 must pass, got %v", err)
	}
}
