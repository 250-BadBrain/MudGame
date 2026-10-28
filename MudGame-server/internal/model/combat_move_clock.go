package model

import "math"

// 主动招式双CD时钟纯函数（实现规范§4）。所有时序以Unix毫秒保存，
// 测试注入固定时钟，不得真实等待。基准AttackSpeed=4。

// ActionDurationMs 计算一次主动施放的角色出招时长：
// actionMs = max(1000, ceil(baseMs × 4 / effectiveSpeed))。
// 无效/非正攻速按基准4兼容；合法减速可以令出招CD长于4秒；
// 1000ms 是防极端候选下限。技能CD不经过本函数。
func ActionDurationMs(baseMs int, effectiveSpeed float64) int64 {
	if effectiveSpeed <= 0 || math.IsNaN(effectiveSpeed) || math.IsInf(effectiveSpeed, 0) {
		effectiveSpeed = 4
	}
	return int64(math.Max(1000, math.Ceil(float64(baseMs)*4/effectiveSpeed)))
}

// MergeMoveBlock 合并按钮的连续受限窗口（仅用于表现）：
// 实际解锁时间始终是 max(actionReadyAt, skillReadyAt)，两种CD不相加。
//   - 原窗口已经结束后再进入限制：新窗口起点=now，进度从满开始；
//   - 原窗口仍在进行且解锁时刻未提前：保留原起点，条形继续匀速减少；
//   - 新出招CD比剩余技能CD更长时：只延长终点，保留起点（进度可回升）。
func MergeMoveBlock(old CooldownWindow, action, skill CooldownWindow, now int64) CooldownWindow {
	end := max(action.ReadyAt, skill.ReadyAt)
	if end <= now {
		return CooldownWindow{}
	}
	if old.ReadyAt > now && old.StartedAt <= now {
		return CooldownWindow{StartedAt: old.StartedAt, ReadyAt: end}
	}
	return CooldownWindow{StartedAt: now, ReadyAt: end}
}

// CooldownFraction 计算窗口的剩余进度（1=刚进入冷却，0=可用），供界面与测试共用。
func CooldownFraction(w CooldownWindow, now int64) float64 {
	if w.ReadyAt <= now || w.ReadyAt <= w.StartedAt {
		return 0
	}
	return min(1, max(0, float64(w.ReadyAt-now)/float64(w.ReadyAt-w.StartedAt)))
}
