package manager

import (
	"math/rand"
	"testing"
	"time"

	"mudgame/server/internal/model"
	"mudgame/server/internal/storage"
)

// 测试夹具：可施放招式的玩家（切磋武馆教头）+可注入时钟。
type moveTestFixture struct {
	r      *RuntimeManager
	id     string
	npcID  string
	now    int64
	battle string
}

func newMoveTestFixture(t *testing.T) *moveTestFixture {
	t.Helper()
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.Skills["basic_unarmed"] = 100
	p.Skills["basic_neigong"] = 100
	p.Skills["basic_blade"] = 120
	p.Skills["basic_literacy"] = 80
	p.Skills["basic_hidden_weapon"] = 100
	p.Skills["basic_spear"] = 120
	// 全部基本功补到120：进阶武学的传承前置检查需要
	for sid, def := range fileSkills {
		if def.Type == "BASIC" {
			p.Skills[sid] = 120
		}
	}
	p.Attributes.Resources.TrainedMaxMP = 500
	p.Attributes.Resources.MP = 500
	grantArt(p, "hunyuan_neigong", 100)
	grantArt(p, "mantian_huayu", 100)
	grantArt(p, "bawang_qiang", 100)
	grantArt(p, "xiulou_daofa", 100)
	grantArt(p, "wuzhen_pian", 100)
	grantArt(p, "talang_jue", 100)
	grantArt(p, "thunder_sword", 100)
	grantArt(p, "fuhu_jin", 100)
	grantArt(p, "tiebi_gong", 100)
	grantArt(p, "taiyi_jianfa", 100)
	grantArt(p, "dongting_xinf", 100)
	grantArt(p, "lieshi_jian", 100)
	grantArt(p, "poxu_zhi", 100)
	grantArt(p, "taxue_wuhen", 100)
	p.ActiveArts["internal"] = "hunyuan_neigong"
	p.Equipment["WEAPON"] = "steel_blade"
	p.Attributes.Resources.HP = effectiveAttributesForPlayerNoLock(p).MaxHP // 切磋要求满血
	if err := r.savePlayers(players); err != nil {
		t.Fatal(err)
	}
	npc, _ := r.staticNPCInRoomNoLock(fileRooms["ly_weapon_school"], "weapon_school_master")
	// 测试确定性：削弱教头闪避/招架，保证普攻与招式稳定命中
	npc.Attributes.Derived.Dodge = 1
	npc.Attributes.Derived.Parry = 0
	if _, err := r.Spar(id, npc.ID); err != nil {
		t.Fatalf("spar failed: %v", err)
	}
	fx := &moveTestFixture{r: r, id: id, npcID: npc.ID, now: 1_000_000}
	r.combatRandom = rand.New(rand.NewSource(7))
	r.combatNow = func() time.Time { return time.UnixMilli(fx.now) }
	players, _ = r.loadPlayers()
	fx.battle = r.currentBattleIDNoLock(ActorKey{Kind: "player", EntityID: id})
	return fx
}

func (fx *moveTestFixture) player(t *testing.T) *model.Player {
	t.Helper()
	players, _ := fx.r.loadPlayers()
	return players[fx.id]
}

func (fx *moveTestFixture) view(t *testing.T) model.CombatMovesView {
	t.Helper()
	view, err := fx.r.GetCombatMoves(fx.id)
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func findMove(t *testing.T, view model.CombatMovesView, id string) model.CombatMoveView {
	t.Helper()
	for _, m := range view.Moves {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("move %s not in view (%d moves)", id, len(view.Moves))
	return model.CombatMoveView{}
}

// 双CD与序号：施放A开启出招CD与A技能CD；出招CD内B被拒且不重开任何CD；
// 出招CD结束后B可用而A仍在技能CD；相同序号重放只结算一次。
func TestCastMoveDualCDAndSequence(t *testing.T) {
	fx := newMoveTestFixture(t)
	startMP := fx.player(t).Attributes.Resources.MP

	res, err := fx.r.CastCombatMove(fx.id, "basic_blade/m01", fx.battle, 0)
	if err != nil || res.Code != "" {
		t.Fatalf("cast A failed: %v %+v", err, res)
	}
	if fx.player(t).Attributes.Resources.MP != startMP-6 {
		t.Fatalf("MP not deducted exactly once: %d", fx.player(t).Attributes.Resources.MP)
	}
	if res.AppliedSequence != 1 {
		t.Fatalf("sequence: %d", res.AppliedSequence)
	}
	// 重放同一序号：拒绝且不再扣费、不再推进
	res2, err := fx.r.CastCombatMove(fx.id, "basic_blade/m01", fx.battle, 0)
	if err != nil || res2.Code != "stale_sequence" {
		t.Fatalf("replay must be stale, got %v %+v", err, res2)
	}
	if fx.player(t).Attributes.Resources.MP != startMP-6 {
		t.Fatal("replay deducted MP again")
	}
	// 双CD窗口：action到now+4000，A技能CD到now+10000
	view := fx.view(t)
	mvA := findMove(t, view, "basic_blade/m01")
	if view.ActionReadyAt != fx.now+4000 || mvA.SkillReadyAt != fx.now+15000 {
		t.Fatalf("dual CD windows: action=%d skill=%d", view.ActionReadyAt, mvA.SkillReadyAt)
	}
	// 出招CD内放B：拒绝且B不进技能CD、出招CD不延长
	fx.now += 1000
	res3, _ := fx.r.CastCombatMove(fx.id, "hunyuan_neigong/m02", fx.battle, 1)
	if res3.Code != "action_cooldown" {
		t.Fatalf("B during action CD: %q", res3.Code)
	}
	if mv := findMove(t, fx.view(t), "hunyuan_neigong/m02"); mv.SkillReadyAt != 0 {
		t.Fatal("failed cast started skill CD")
	}
	// 出招CD结束后B可用；A仍在技能CD
	fx.now += 3000
	res4, err := fx.r.CastCombatMove(fx.id, "hunyuan_neigong/m02", fx.battle, 1)
	if err != nil || res4.Code != "" {
		t.Fatalf("B after action CD: %v %q", err, res4.Code)
	}
	view = fx.view(t)
	mvA = findMove(t, view, "basic_blade/m01")
	if mvA.SkillReadyAt != 1_000_000+15000 {
		t.Fatalf("B cast must not reset A skill CD: %d", mvA.SkillReadyAt)
	}
	if view.ActionReadyAt != fx.now+4000 {
		t.Fatalf("B action window: %d", view.ActionReadyAt)
	}
}

// 失败不扣费不进CD：缺内力、非战斗、过期战斗令牌。
func TestCastMoveFailuresDoNotConsume(t *testing.T) {
	fx := newMoveTestFixture(t)
	players, _ := fx.r.loadPlayers()
	players[fx.id].Attributes.Resources.MP = 2
	_ = fx.r.savePlayers(players)
	res, _ := fx.r.CastCombatMove(fx.id, "basic_blade/m01", fx.battle, 0)
	if res.Code != "mp_not_enough" {
		t.Fatalf("want mp_not_enough, got %q", res.Code)
	}
	view := fx.view(t)
	if view.ActionReadyAt != 0 || findMove(t, view, "basic_blade/m01").SkillReadyAt != 0 {
		t.Fatal("failed cast started a CD")
	}
	// 非战斗：解除切磋后不可施放
	fx.r.mu.Lock()
	pls, _ := fx.r.loadPlayers()
	fx.r.clearCombatNoLock(pls[fx.id])
	_ = fx.r.savePlayers(pls)
	fx.r.mu.Unlock()
	res, _ = fx.r.CastCombatMove(fx.id, "basic_blade/m01", fx.battle, 0)
	if res.Code != "not_in_combat" {
		t.Fatalf("want not_in_combat, got %q", res.Code)
	}
}

// 普攻并行：主动施放即时结算，2秒普攻tick不被替代/重置（施放后立即tick仍产生普攻）。
func TestCastMoveRunsAlongsideAutoAttack(t *testing.T) {
	fx := newMoveTestFixture(t)
	npcBefore := fx.r.staticNPCs[fx.npcID].Attributes.Resources.HP
	res, _ := fx.r.CastCombatMove(fx.id, "basic_blade/m01", fx.battle, 0)
	if res.Code != "" {
		t.Fatalf("cast failed: %q", res.Code)
	}
	afterCast := fx.r.staticNPCs[fx.npcID].Attributes.Resources.HP
	if afterCast >= npcBefore {
		t.Fatal("move dealt no immediate damage")
	}
	// 普攻tick照常结算（对已受损目标继续攻击）
	fx.now += 2000
	if _, err := fx.r.ProcessCombatTick(); err != nil {
		t.Fatal(err)
	}
	if fx.r.staticNPCs[fx.npcID].Attributes.Resources.HP > afterCast {
		t.Fatal("auto attack did not run on its own tick")
	}
}

// 触发域：察势的蓄势不被普攻消耗，只被主动招式消耗一次。
func TestTriggerDomainNextActive(t *testing.T) {
	fx := newMoveTestFixture(t)
	res, _ := fx.r.CastCombatMove(fx.id, "basic_literacy/m01", fx.battle, 0)
	if res.Code != "" {
		t.Fatalf("察势 cast failed: %q", res.Code)
	}
	key := ActorKey{Kind: "player", EntityID: fx.id}
	countNextActive := func() int {
		n := 0
		for _, ef := range fx.r.activeEffectsNoLock(key) {
			if ef.Op == "next_active" {
				n++
			}
		}
		return n
	}
	if countNextActive() != 1 {
		t.Fatal("察势 effect missing")
	}
	// 普攻tick不消耗
	fx.now += 2000
	if _, err := fx.r.ProcessCombatTick(); err != nil {
		t.Fatal(err)
	}
	if countNextActive() != 1 {
		t.Fatal("auto attack consumed next_active")
	}
	// 主动招式消耗一次
	fx.now += 4000
	res, _ = fx.r.CastCombatMove(fx.id, "basic_blade/m01", fx.battle, 1)
	if res.Code != "" {
		t.Fatalf("劈斩 cast failed: %q", res.Code)
	}
	if countNextActive() != 0 {
		t.Fatal("next_active not consumed by move")
	}
}

// 最近事件窗口：3999ms可用、4000ms失效；成功使用即消耗。
func TestRecentEventWindowBoundary(t *testing.T) {
	fx := newMoveTestFixture(t)
	binding := fx.r.currentBindingNoLock(ActorKey{Kind: "player", EntityID: fx.id})
	src := ActorKey{Kind: "static_npc", EntityID: fx.npcID}
	fx.r.recordCombatEventNoLock(binding, RecentCombatEvent{BattleID: binding.ID, Source: src, ActionKind: "auto", Dodged: true, At: fx.now})
	if binding.recentEventNoLock(fx.now+3999, src, "", false) == nil {
		t.Fatal("event must be usable within 3999ms")
	}
	if binding.recentEventNoLock(fx.now+4000, src, "", false) != nil {
		t.Fatal("event must expire at 4000ms")
	}
	// 踏浪穿隙：无事件时拒绝；有事件时成功并消耗（先推进时间让边界事件过期）
	fx.now += 5000
	players, _ := fx.r.loadPlayers()
	players[fx.id].ActiveArts["movement"] = "talang_jue"
	_ = fx.r.savePlayers(players)
	res, _ := fx.r.CastCombatMove(fx.id, "talang_jue/m03", fx.battle, 0)
	if res.Code != "condition_not_met" {
		t.Fatalf("want condition_not_met, got %q", res.Code)
	}
	fx.r.recordCombatEventNoLock(binding, RecentCombatEvent{BattleID: binding.ID, Source: src, ActionKind: "auto", Dodged: true, At: fx.now})
	fx.now += 100
	res, _ = fx.r.CastCombatMove(fx.id, "talang_jue/m03", fx.battle, 0)
	if res.Code != "" {
		t.Fatalf("踏浪穿隙 with recent dodge failed: %q", res.Code)
	}
	if binding.recentEventNoLock(fx.now, src, "", false) != nil {
		t.Fatal("recent event not consumed on successful use")
	}
}

// 踏雪折影：只认对手主动动作的实际扣血，多段合为一笔。
func TestRecentActiveDamageCondition(t *testing.T) {
	fx := newMoveTestFixture(t)
	binding := fx.r.currentBindingNoLock(ActorKey{Kind: "player", EntityID: fx.id})
	src := ActorKey{Kind: "static_npc", EntityID: fx.npcID}
	// 普攻伤害不算
	fx.r.recordCombatEventNoLock(binding, RecentCombatEvent{BattleID: binding.ID, Source: src, ActionKind: "auto", ActualDamage: 50, At: fx.now})
	players, _ := fx.r.loadPlayers()
	players[fx.id].ActiveArts["movement"] = "taxue_wuhen"
	_ = fx.r.savePlayers(players)
	res, _ := fx.r.CastCombatMove(fx.id, "taxue_wuhen/m03", fx.battle, 0)
	if res.Code != "condition_not_met" {
		t.Fatalf("auto damage must not satisfy recent_active_damage: %q", res.Code)
	}
	// 主动动作实扣血满足
	fx.r.recordCombatEventNoLock(binding, RecentCombatEvent{BattleID: binding.ID, Source: src, ActionKind: "move", ActualDamage: 50, At: fx.now})
	fx.now += 100
	res, _ = fx.r.CastCombatMove(fx.id, "taxue_wuhen/m03", fx.battle, 0)
	if res.Code != "" {
		t.Fatalf("踏雪折影 failed: %q", res.Code)
	}
	players, _ = fx.r.loadPlayers()
	if players[fx.id].Attributes.Resources.HP <= 50 {
		t.Fatalf("expected heal from recent damage, HP=%d", players[fx.id].Attributes.Resources.HP)
	}
}

// 流血与持续效果：预算快照按tick拆分，dot不消耗护体。
func TestBleedDotTickProcessing(t *testing.T) {
	fx := newMoveTestFixture(t)
	// 漫天花雨 m02 雨落无声：需要暗器在手
	players, _ := fx.r.loadPlayers()
	players[fx.id].Equipment["WEAPON"] = "darts"
	players[fx.id].ActiveArts["combat"] = "mantian_huayu"
	_ = fx.r.savePlayers(players)
	res, _ := fx.r.CastCombatMove(fx.id, "mantian_huayu/m02", fx.battle, 0)
	if res.Code != "" {
		t.Fatalf("雨落无声 failed: %q", res.Code)
	}
	// dot已建立：NPC身上有bleed效果
	key := ActorKey{Kind: "static_npc", EntityID: fx.npcID}
	var bleed *CombatEffect
	for _, ef := range fx.r.activeEffectsNoLock(key) {
		if ef.Op == "bleed" {
			bleed = ef
		}
	}
	if bleed == nil || len(bleed.TickBudget) != 3 {
		t.Fatalf("bleed effect missing: %+v", bleed)
	}
	npcHP := fx.r.staticNPCs[fx.npcID].Attributes.Resources.HP
	fx.now += 2000
	if _, err := fx.r.ProcessCombatTick(); err != nil {
		t.Fatal(err)
	}
	if fx.r.staticNPCs[fx.npcID].Attributes.Resources.HP >= npcHP {
		t.Fatal("bleed tick did not damage NPC")
	}
}

// 十二个补充招式全部可正常施放（各按其条件准备）。
func TestTwelveNewMovesCastable(t *testing.T) {
	cases := []struct {
		moveID string
		weapon string // 空手传"none"
		slot   string // 需要装备的战斗位（空=回退基本功）
		prep   func(t *testing.T, fx *moveTestFixture)
	}{
		{"thunder_sword/m03", "iron_sword", "combat", nil},
		{"hunyuan_neigong/m03", "", "internal", nil},
		{"talang_jue/m03", "", "movement", func(t *testing.T, fx *moveTestFixture) {
			binding := fx.r.currentBindingNoLock(ActorKey{Kind: "player", EntityID: fx.id})
			fx.r.recordCombatEventNoLock(binding, RecentCombatEvent{BattleID: binding.ID,
				Source: ActorKey{Kind: "static_npc", EntityID: fx.npcID}, ActionKind: "auto", Dodged: true, At: fx.now})
		}},
		{"fuhu_jin/m03", "none", "combat", nil},
		{"tiebi_gong/m03", "", "guard", nil},
		{"wuzhen_pian/m03", "", "study", nil},
		{"taiyi_jianfa/m03", "iron_sword", "combat", nil},
		{"dongting_xinf/m03", "", "internal", nil},
		{"bawang_qiang/m03", "training_spear", "combat", nil},
		{"lieshi_jian/m03", "training_mace", "combat", nil},
		{"poxu_zhi/m03", "none", "combat", nil},
		{"taxue_wuhen/m03", "", "movement", func(t *testing.T, fx *moveTestFixture) {
			binding := fx.r.currentBindingNoLock(ActorKey{Kind: "player", EntityID: fx.id})
			fx.r.recordCombatEventNoLock(binding, RecentCombatEvent{BattleID: binding.ID,
				Source: ActorKey{Kind: "static_npc", EntityID: fx.npcID}, ActionKind: "move", ActualDamage: 30, At: fx.now})
		}},
	}
	for _, c := range cases {
		seq := int64(0)
		fx := newMoveTestFixture(t)
		players, _ := fx.r.loadPlayers()
		if c.weapon == "none" {
			delete(players[fx.id].Equipment, "WEAPON")
		} else if c.weapon != "" {
			players[fx.id].Equipment["WEAPON"] = c.weapon
		}
		if c.slot != "" {
			artID := ""
			for _, part := range []string{"/m01", "/m02", "/m03"} {
				if idx := len(c.moveID) - len(part); idx > 0 && c.moveID[idx:] == part {
					artID = c.moveID[:idx]
				}
			}
			players[fx.id].ActiveArts[c.slot] = artID
		}
		_ = fx.r.savePlayers(players)
		if c.prep != nil {
			c.prep(t, fx)
		}
		res, err := fx.r.CastCombatMove(fx.id, c.moveID, fx.battle, seq)
		if err != nil {
			t.Fatalf("%s: %v", c.moveID, err)
		}
		if res.Code != "" {
			t.Fatalf("%s rejected: %q %s", c.moveID, res.Code, res.Message)
		}
		seq = res.AppliedSequence
	}
}

// NPC试点：开战先普攻，两CD就绪时提前2秒公开意图，到点即时结算并开启双CD。
func TestNPCIntentScheduleAndRelease(t *testing.T) {
	r, id, inst := createDungeonTestPlayer(t, "datang_baling")
	bossID := ""
	for nid, n := range inst.NPCs {
		if n.TemplateID == "db_zhang_xiu" {
			bossID = nid
		}
	}
	if bossID == "" {
		t.Fatal("missing boss")
	}
	boss := inst.NPCs[bossID]
	moveDungeonTestPlayer(t, r, id, boss.RoomID)
	if _, err := r.Kill(id, bossID); err != nil {
		t.Fatal(err)
	}
	now := int64(5_000_000)
	r.combatNow = func() time.Time { return time.UnixMilli(now) }
	// 开战先给一个普攻tick（AutoRounds≥1）
	if _, err := r.ProcessCombatTick(); err != nil {
		t.Fatal(err)
	}
	r.mu.Lock()
	key := ActorKey{Kind: "dungeon_npc", EntityID: bossID}
	binding := r.currentBindingNoLock(key)
	if binding == nil || binding.AutoRounds < 1 {
		t.Fatal("NPC binding missing after first tick")
	}
	mpBefore := inst.NPCs[bossID].Attributes.Resources.MP
	if mpBefore < 16 {
		t.Fatalf("pilot MP too low: %d", mpBefore)
	}
	combatPlayers, _ := r.loadPlayers()
	r.scheduleNPCIntentsNoLock(combatPlayers)
	intent := r.npcIntents[key.String()]
	if intent == nil || intent.MoveID != "bawang_qiang/m01" {
		t.Fatalf("intent not scheduled: %+v", intent)
	}
	if intent.ReleaseAt != now+2000 {
		t.Fatalf("intent must be announced 2s ahead: %+v", intent)
	}
	r.mu.Unlock()
	// 到点前不释放
	now += 1999
	if res, _ := r.ProcessCombatMovesTick(); len(res.Messages) != 0 {
		t.Fatal("intent released early")
	}
	// 到点即时结算：MP实扣、双CD开启、意图清除
	now += 1
	res, err := r.ProcessCombatMovesTick()
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Messages) == 0 {
		t.Fatal("intent not released")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if got := inst.NPCs[bossID].Attributes.Resources.MP; got != mpBefore-16 {
		t.Fatalf("NPC MP not deducted: %d", got)
	}
	state := r.npcCombatStateNoLock(key)
	if state.Action.ReadyAt != now+4000 {
		t.Fatalf("NPC action CD: %d", state.Action.ReadyAt)
	}
	if _, ok := r.npcIntents[key.String()]; ok {
		t.Fatal("intent not cleared after release")
	}
}

// NPC MP耗尽只普攻：不再发布意图。
func TestNPCIntentNeedsMP(t *testing.T) {
	r, id, inst := createDungeonTestPlayer(t, "datang_baling")
	bossID := ""
	for nid, n := range inst.NPCs {
		if n.TemplateID == "db_zhang_xiu" {
			bossID = nid
		}
	}
	boss := inst.NPCs[bossID]
	moveDungeonTestPlayer(t, r, id, boss.RoomID)
	if _, err := r.Kill(id, bossID); err != nil {
		t.Fatal(err)
	}
	now := int64(5_000_000)
	r.combatNow = func() time.Time { return time.UnixMilli(now) }
	r.mu.Lock()
	inst.NPCs[bossID].Attributes.Resources.MP = 0
	r.mu.Unlock()
	if _, err := r.ProcessCombatTick(); err != nil {
		t.Fatal(err)
	}
	r.mu.Lock()
	combatPlayers, _ := r.loadPlayers()
	r.scheduleNPCIntentsNoLock(combatPlayers)
	if intent := r.npcIntents[("dungeon_npc||" + bossID)]; intent != nil {
		t.Fatal("intent scheduled without MP")
	}
	r.mu.Unlock()
}

// 存档：施放后的双CD与序号跨重启保留；旧档无字段读零值。
func TestCombatMoveStateSurvivesRestart(t *testing.T) {
	fx := newMoveTestFixture(t)
	res, _ := fx.r.CastCombatMove(fx.id, "basic_blade/m01", fx.battle, 0)
	if res.Code != "" {
		t.Fatalf("cast failed: %q", res.Code)
	}
	actionReady := res.View.ActionReadyAt
	skillReady := findMove(t, *res.View, "basic_blade/m01").SkillReadyAt
	fs, ok := fx.r.store.(*storage.FileStore)
	if !ok {
		t.Fatal("file store expected")
	}
	dir := fs.BaseDir()
	if err := fx.r.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := NewRuntimeManager(dir)
	defer func() { _ = reopened.Close() }()
	players, _ := reopened.loadPlayers()
	p := players[fx.id]
	if p.CombatMoves.Sequence != 1 {
		t.Fatalf("sequence lost: %d", p.CombatMoves.Sequence)
	}
	if p.CombatMoves.Action.ReadyAt != actionReady {
		t.Fatalf("action CD lost: %d vs %d", p.CombatMoves.Action.ReadyAt, actionReady)
	}
	if got := p.CombatMoves.Skills["basic_blade/m01"].ReadyAt; got != skillReady {
		t.Fatalf("skill CD lost: %d vs %d", got, skillReady)
	}
	// 玩家在切磋中被Close→重启复位为IDLE（重启作废战斗会话）
	if p.State == "COMBAT" {
		t.Fatal("restart must reset COMBAT to IDLE")
	}
}
