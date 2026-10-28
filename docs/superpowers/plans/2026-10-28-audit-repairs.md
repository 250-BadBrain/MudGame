# 全量审查修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 按用户确认规则修复审查缺陷，保留并标记后期内容问题，补货和强化预算留待判断。

**Architecture:** 以model共享价格/数值边界，manager执行权威状态约束，gamedata校验转换价值和装备别名。存档读取幂等合并别名，前端使用服务端卖价并结合实时状态显示快速完成。

**Tech Stack:** Go、JSON、Vue3、Python、独立Playwright浏览器验收。

## Global Constraints

- 规格见docs/design/audit_repairs_2026_10_28.md。20%回收，转化材料价值严格递减；免费快速完成不新增消耗。
- 安全区禁止主动攻击玩家，区外离线可攻击；NPC默认被动；切磋双满血、至少留1血。
- 不改强化预算，不实施烟尘粉补货，不删除重复度高的副本。
- 不读写正式存档、不推送部署；沿用composable依赖注入。用户已要求实施，当前会话内完成。

### Task 1: 输入、战斗和存档

Files: model新增economy.go；manager/system.go、player.go、combat.go、effective.go；storage/store.go；app/handlers_system_game.go和交易handler。

- [x] 测试CreateCharacter越界、Unicode姓名，Buy/Sell/Discard极大/零/负数/小数输入拒绝且不改变状态。
- [x] 增加安全数量和checked add/multiply，姓名按trim+runes，属性各10～25；协议入口严格整数而不是截断。
- [x] 测试双方满血、受击方20%伤害封顶、至少1HP、认输取消反击、NPC胜负均回满；安全区玩家拒绝、区外离线接受。
- [x] 保存副本死亡不改成IDLE，加载零血不补血；补initiative并测试装备和强化。

### Task 2: 经济规则和装备合并

Files: model/economy.go、gamedata新增economy_validation.go和item_aliases.go；manager新增item_aliases.go；resources/items.json、item_aliases.json、dialogues.json及引用；migrations/migrate_martial_v2.py。

Interfaces: `ItemSellPrice(value int) int`，`canonicalItemID(id string) string`，`normalizePlayerItemAliases(p *model.Player) (bool,error)`。

- [x] 确定性测试回收20%、完整配方输出价值低于原材料，缺材料失败无消费；同步服务器sellPrice和前端价格展示。
- [x] 调整金创药/回春丹/大还丹等材料，steel_blade估值300；更新所有菜单、文案和生成器。
- [x] 将9个长ID合并到短ID，规范奖励每组仅一条；测试库存/装备/强化最大值合并、重复费用退款、重载幂等和旧客户端请求解析。
- [x] 更新离线V1迁移识别别名，历史半价补偿不随20%交易规则变化；全量资源和迁移测试通过。

### Task 3: 界面、内容标记和文档

Files: GameView.vue、GameInfoPanel.vue、useDungeonView.js、CharacterCreateView.vue、useShopLearn.js、useBackpackActions.js；模板/目录contentStatus字段；generate_player_guide.py及player文档。

- [x] 快速完成按已通关事实+实时IDLE计算，保留防重序号，测试从忙碌恢复后按钮立即可用。
- [x] 40个后期四房关标contentStatus=needs_depth和说明，界面/玩家手册显示，关卡入口和奖励不变。
- [x] 移除攻速预留展示；尸体搜索增加刺客败后前置；退役物品来源及药品说明同步。
- [x] 生成12页玩家手册并check；登记审查修复结果，保留历史审计不覆写原事实。

### Task 4: 真实验收和交付

Files: 新增manager/gamedata/storage/前端回归测试，扩展scripts/tests真实协议与浏览器脚本。

- [x] 运行Go build/vet/test、前端test/lint/build、Python迁移/部署全测；隔离协议验证输入/经济/死亡/别名/安全区与快速完成。
- [x] 独立浏览器验证背包回收价显示、快速状态恢复、内容标记和长中文姓名；截图并确认无运行错误。
- [x] 回写规格和计划验收证据，清理测试服务，中文本地提交；最终详细说明补货和强化选项，等待用户决定。
