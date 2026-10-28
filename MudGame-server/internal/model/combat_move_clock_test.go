package model

import (
	"math"
	"testing"
)

// 攻速联动：出招CD=max(1000, ceil(base×4/speed))；技能CD不经过本函数。
func TestActionDurationMsSpeedTiers(t *testing.T) {
	cases := []struct {
		speed float64
		want  int64
	}{
		{4, 4000},
		{4.8, 3334}, // ceil(4000×4/4.8)=ceil(3333.33)=3334，不能先把4.8取整成4
		{5, 3200},
		{8, 2000},
		{100000, 1000}, // 防极端下限
		{0, 4000},      // 非正按基准4兼容
		{-2, 4000},
		{math.NaN(), 4000},
		{math.Inf(1), 4000},
		{math.Inf(-1), 4000},
		{2, 8000}, // 合法减速可以长于4秒
	}
	for _, c := range cases {
		if got := ActionDurationMs(4000, c.speed); got != c.want {
			t.Fatalf("speed %v: got %d want %d", c.speed, got, c.want)
		}
	}
	if got := ActionDurationMs(2000, 8); got != 1000 {
		t.Fatalf("special base 2000 at 200%%: got %d want 1000", got)
	}
}

// 合并阻塞窗口三行表（实现规范§4）：解锁时刻取max而非相加。
func TestMergeMoveBlockScenarios(t *testing.T) {
	now := int64(10_000)
	action := CooldownWindow{StartedAt: now, ReadyAt: now + 4000}
	// A本招还剩12秒，新出招CD 4秒：A仍按原结束时刻结束，不回满、不改技能CD
	oldA := CooldownWindow{StartedAt: now - 8000, ReadyAt: now + 12000}
	got := MergeMoveBlock(oldA, action, oldA, now)
	if got.StartedAt != oldA.StartedAt || got.ReadyAt != oldA.ReadyAt {
		t.Fatalf("12s-left case polluted: %+v", got)
	}
	// A本招还剩2秒，新出招CD 4秒：实际解锁延后到4秒；保留原窗口起点
	oldA2 := CooldownWindow{StartedAt: now - 8000, ReadyAt: now + 2000}
	got = MergeMoveBlock(oldA2, action, oldA2, now)
	if got.StartedAt != oldA2.StartedAt || got.ReadyAt != now+4000 {
		t.Fatalf("2s-left case: %+v", got)
	}
	// A已可用：新窗口起点为本次施放时刻，进度从满开始
	got = MergeMoveBlock(CooldownWindow{}, action, CooldownWindow{}, now)
	if got.StartedAt != now || got.ReadyAt != now+4000 {
		t.Fatalf("available case: %+v", got)
	}
	// 两个窗口都为空：不可用
	if got := MergeMoveBlock(CooldownWindow{}, CooldownWindow{}, CooldownWindow{}, now); got != (CooldownWindow{}) {
		t.Fatalf("empty case: %+v", got)
	}
	// 原窗口已过期：重新从now起算
	oldA3 := CooldownWindow{StartedAt: now - 20000, ReadyAt: now - 100}
	got = MergeMoveBlock(oldA3, action, CooldownWindow{}, now)
	if got.StartedAt != now || got.ReadyAt != now+4000 {
		t.Fatalf("expired case: %+v", got)
	}
}

// 进度按窗口线性减少，重连恢复同一窗口不能从满条重播。
func TestCooldownFractionLinear(t *testing.T) {
	w := CooldownWindow{StartedAt: 1000, ReadyAt: 21000}
	if got := CooldownFraction(w, 2000); math.Abs(got-0.95) > 1e-9 {
		t.Fatalf("t=2000: got %v want 0.95", got)
	}
	if got := CooldownFraction(w, 21000); got != 0 {
		t.Fatalf("ready: got %v", got)
	}
	if got := CooldownFraction(w, 500); got != 1 {
		t.Fatalf("before start clamps to full: got %v", got)
	}
	if got := CooldownFraction(CooldownWindow{}, 9999); got != 0 {
		t.Fatalf("empty window: got %v", got)
	}
}

// 成长系数边界：S(100)=1，S(200)≈1.067，渐近1.2；等级<1按1。
func TestCombatMoveGrowth(t *testing.T) {
	// 文档给的是舍入值：S(1)≈0.804、S(20)≈0.867
	if math.Abs(CombatMoveGrowth(1)-0.804) > 5e-4 || math.Abs(CombatMoveGrowth(20)-0.867) > 5e-4 {
		t.Fatal("S(1)/S(20) drift")
	}
	if CombatMoveGrowth(100) != 1 {
		t.Fatal("S(100) must be exactly 1")
	}
	if math.Abs(CombatMoveGrowth(200)-(0.8+0.4*200/300)) > 1e-12 {
		t.Fatal("S(200) formula drift")
	}
	if CombatMoveGrowth(0) != CombatMoveGrowth(1) || CombatMoveGrowth(-5) != CombatMoveGrowth(1) {
		t.Fatal("level below 1 must clamp to 1")
	}
	if CombatMoveGrowth(1000000) >= 1.2 {
		t.Fatal("growth must stay below 1.2")
	}
}

// 招架乘数按削减幅度成长：新倍率=1-幅度×S，下限0.3；"乘0.5"存0.5不直接乘S。
func TestParryMultiplierDirection(t *testing.T) {
	def := CombatMoveDef{ParryReduction: 0.5}
	if got := def.ParryMultiplier(100); got != 0.5 {
		t.Fatalf("level 100: got %v want 0.5", got)
	}
	if got := def.ParryMultiplier(1); got <= def.ParryMultiplier(100) {
		t.Fatalf("lower level must be MORE parriable, got %v vs %v", got, def.ParryMultiplier(100))
	}
	if got := def.ParryMultiplier(200); got >= def.ParryMultiplier(100) {
		t.Fatalf("higher level must be LESS parriable, got %v vs %v", got, def.ParryMultiplier(100))
	}
	// 削减幅度0.5即使渐近S≈1.2也只有0.6，不触0.7上限
	if got := def.ParryMultiplier(100000); math.Abs(got-(1-0.5*CombatMoveGrowth(100000))) > 1e-9 {
		t.Fatalf("asymptote: got %v", got)
	}
	// 更大削减在S增长后触0.7上限：乘数下限0.3
	big := CombatMoveDef{ParryReduction: 0.6}
	if got := big.ParryMultiplier(1000000); got != 0.3 {
		t.Fatalf("cap: got %v want 0.3", got)
	}
	if got := (CombatMoveDef{ParryReduction: 0.3}).ParryMultiplier(100); got != 0.7 {
		t.Fatalf("乘0.7 case: got %v want 0.7", got)
	}
	if got := (CombatMoveDef{}).ParryMultiplier(50); got != 1 {
		t.Fatalf("no reduction: got %v", got)
	}
}
