package manager

import (
	"strings"
	"testing"
	"time"

	"mudgame/server/internal/model"
)

// 虚弱期内战斗衍生属性减半、到期自动恢复（见 docs/design/formulas.md 1.5）
func TestWeaknessHalvesCombatDerived(t *testing.T) {
	p := newFullyPopulatedPlayer("weak-hero")
	p.State = "IDLE"
	p.WeakUntil = 0 // 辅助构造默认带未来虚弱时间，先清零取基准值

	normal := effectiveAttributesForPlayerNoLock(p)

	p.WeakUntil = time.Now().Unix() + 300
	weak := effectiveAttributesForPlayerNoLock(p)

	if weak.Attack != normal.Attack/2 || weak.Defense != normal.Defense/2 ||
		weak.HitRate != normal.HitRate/2 || weak.Dodge != normal.Dodge/2 ||
		weak.Parry != normal.Parry/2 || weak.Initiative != normal.Initiative/2 {
		t.Fatalf("weakness should halve combat derived attrs: normal attack=%d weak attack=%d", normal.Attack, weak.Attack)
	}
	if weak.MaxHP != normal.MaxHP || weak.Strength != normal.Strength {
		t.Fatalf("weakness must not change max hp or base attrs: normal=%+v weak=%+v", normal, weak)
	}

	// 虚弱过期后恢复
	p.WeakUntil = time.Now().Unix() - 1
	recovered := effectiveAttributesForPlayerNoLock(p)
	if recovered.Attack != normal.Attack {
		t.Fatalf("combat attrs should recover after weakness expires: %d != %d", recovered.Attack, normal.Attack)
	}
}

// 玩家气血上限应随根骨生效：MaxHP = 60 + 根骨×4 + 内功转化
func TestPlayerMaxHealthScalesWithConstitution(t *testing.T) {
	lowCon := newFullyPopulatedPlayer("low-con")
	lowCon.State = "IDLE"
	lowCon.Attributes.Base.Constitution = 10
	lowCon.Attributes.Growth.PostConstitution = 0
	lowCon.Skills = map[string]int{}

	highCon := newFullyPopulatedPlayer("high-con")
	highCon.State = "IDLE"
	highCon.Attributes.Base.Constitution = 20
	highCon.Attributes.Growth.PostConstitution = 0
	highCon.Skills = map[string]int{}

	lowEff := effectiveAttributesForPlayerNoLock(lowCon)
	highEff := effectiveAttributesForPlayerNoLock(highCon)
	if highEff.MaxHP-lowEff.MaxHP != 40 {
		t.Fatalf("constitution should add 4 hp each: low=%d high=%d", lowEff.MaxHP, highEff.MaxHP)
	}
	if lowEff.MaxHP != 60+10*4 {
		t.Fatalf("expected max hp 60+con*4 without neigong conversion, got %d", lowEff.MaxHP)
	}
}

// 复活应附带虚弱并只恢复半血半内力
func TestReliveAppliesWeaknessAndHalfResources(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	if err := r.Register("reliver", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("reliver", "复活侠", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	players, err := r.loadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	p := players[playerID]
	p.State = "DEAD"
	p.Attributes.Resources.HP = 0
	if err := r.savePlayers(players); err != nil {
		t.Fatalf("save players failed: %v", err)
	}

	msg, err := r.Relive(playerID)
	if err != nil {
		t.Fatalf("relive failed: %v", err)
	}
	if !strings.Contains(msg, "虚弱") {
		t.Fatalf("relive message should mention weakness, got: %s", msg)
	}

	reloaded, err := r.loadPlayers()
	if err != nil {
		t.Fatalf("reload players failed: %v", err)
	}
	after := reloaded[playerID]
	if !after.IsWeak(time.Now().Unix()) {
		t.Fatalf("player should be weak after relive, WeakUntil=%d", after.WeakUntil)
	}
	if after.WeakUntil-time.Now().Unix() > weaknessDurationSeconds {
		t.Fatalf("weakness duration exceeds %d seconds", weaknessDurationSeconds)
	}
	eff := effectiveAttributesForPlayerNoLock(after)
	if after.Attributes.Resources.HP != eff.MaxHP/2 {
		t.Fatalf("relive should restore half hp, hp=%d maxHp=%d", after.Attributes.Resources.HP, eff.MaxHP)
	}
	if after.Attributes.Resources.MP != after.Attributes.Resources.TrainedMaxMP/2 {
		t.Fatalf("relive should restore half mp, mp=%d trainedMax=%d", after.Attributes.Resources.MP, after.Attributes.Resources.TrainedMaxMP)
	}
}

// 虚弱到期时间需随存档持久化
func TestWeakUntilPersistsAcrossSaveLoad(t *testing.T) {
	dir := t.TempDir()
	r := NewRuntimeManager(dir)
	players := map[string]*model.Player{
		"weak-p": {
			ID:             "weak-p",
			MartialVersion: model.MartialVersion,
			UserID:         "u1",
			State:          "IDLE",
			Attributes:     model.NewPlayerAttributes(16, 16, 16, 16, 16),
			WeakUntil:      1893456000,
		},
	}
	if err := r.savePlayers(players); err != nil {
		t.Fatalf("save players failed: %v", err)
	}
	// savePlayers 只更新缓存，需 flush 落盘后另一个实例才能读到
	if err := r.Close(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	r2 := NewRuntimeManager(dir)
	loaded, err := r2.loadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	if got := loaded["weak-p"].WeakUntil; got != 1893456000 {
		t.Fatalf("WeakUntil should persist across save/load, got %d", got)
	}
}
