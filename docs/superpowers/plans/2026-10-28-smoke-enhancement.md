# 烟尘粉与强化系数 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 落实用户选择的无限烟尘粉、分属性强化系数，兼容旧投入并完成隔离验收。

**Architecture:** model提供系数，manager复用到属性与预览并记录旧档单次重置资格；现有交易体系处理无限库存，前端经GameView注入确认流程。

**Tech Stack:** Go、Vue3、JSON、Python及Playwright。

## Global Constraints

- 规格：docs/design/smoke_enhancement_2026_10_28.md；无限库存仍250铜，20%回收，不新增限购。
- 强化上限5、费用不变；保留等级，旧ID一次可选重置全额退料，失败零变化。
- 本会话内顺序执行，不派子代理；不推送、不部署、不操作正式存档。

### Task 1: 规则和无限货源

Files: model/enhancement.go、manager/runtime_manager_enhance.go、effective.go、gamedata/item_aliases.go、resources/items.json；对应测试。

- [x] 在测试中断言四店烟尘粉库存-1；连续购买后仍-1且正确扣250N铜。旧库存隔离测试改用明确的测试限量商品，继续验证有限库存不会污染模板。
- [x] 系数接口`EnhancementStep(key string) (float64,bool)`：普通战斗属性/魅力1、maxHp/maxMp5、critRate0.1。`getEnhanceBonus(itemID string,level int) map[string]float64`只增强已有正数效果，实际装备与预览共用。
- [x] 增加小数加成入口：critRate直接累加float64，其余属性转整数后交applyFlatBonus。加载拒绝装备未定义系数。调整先手+5期望为基础3+强化5。

### Task 2: 旧投入兼容与单次重置

Files: manager新增runtime_manager_enhance_reset.go、persistence.go、system.go、item_aliases.go；app/command_registry.go、handlers_phase2.go；engine/workflow.go。

- [x] `prepareEnhancementReset(p *model.Player) (bool,error)`在别名合并后记录兼容完成旗标和旧强化ID资格；新角色直接记录兼容完成。测试重启、别名与新角色。
- [x] `ResetEnhancement(playerID,itemID string)(string,error)`：检查IDLE/持有/资格/1～5级；checked add验证退款余额和古铜，再删等级和资格；夹紧有效HP/MP不回血。累计退款100L(L+1)铜、L(L+1)/2古铜。
- [x] 注册`reset_enhancement`，使用绑定角色ID，成功广播房间快照。测重复请求、忙碌、未持有、溢出均失败且不改变状态；重启后资格不再发。

### Task 3: 页面确认与验收

Files: useBackpackActions.js、GameInfoPanel.vue、GameView.vue、前端测试；scripts/tests新增本轮协议及浏览器验收；文档/生成器。

- [x] enhancementInfo返回canReset/resetMoney/resetBronze；页面增加免费重置与确认，关闭后不发送，确认双击只发一次；成功刷新背包/装备/属性。
- [x] 预览暴击带中文与“个百分点”，小数去尾数；装备和背包共用入口。更新玩家手册及历史选项状态。
- [x] 执行Go build/vet/test、前端test/lint/build、Python22项、真实协议、浏览器截图检查、指南生成/check；记录结果和中文本地提交。
