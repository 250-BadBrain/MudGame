# 项目文档索引

## 玩家内容与剧情（实际游玩入口）

- **[玩家手册：现在可以玩什么](player/README.md)**（新手路线、界面按钮、按兴趣选内容、卡关处理）
- [全部 78 个副本的全局解锁表](player/unlock_order.md)
- 七卷完整流程与剧透：[洛阳轶事](player/stories/luoyang.md) · [大唐双龙传](player/stories/datang.md) · [覆雨翻云](player/stories/fuyu.md) · [破碎虚空](player/stories/posui.md) · [边荒传说](player/stories/bianhuang.md) · [洛水风浪](player/stories/luoshui.md) · [运河风云](player/stories/yunhe.md)
- [八城路线与全部常驻交谈](player/worlds.md)
- [世界任务接取、推进与结局](player/quests.md)（38个唯一任务全接通，3个旧ID兼容别名）
- [修行、商店、武学传承与收藏目录](player/collections.md)
- [当前内容缺口与文案/实现差异](player/content_status.md)
- [全部既有人物、器物与章节的用途索引](content/existing_resource_roles.md)（313个人物、135件器物、78关交互及八城互证事件）
- [三件旧事与六门武学](player/connected_exploration.md)（玩家视角：分支后续、人物地点、武学探索手法和见闻进度）

## 开发流程

- **[统一开发指导文档](DEVELOPMENT.md)**（仓库结构、各模块功能、标准开发流程：先读什么 → 文档先行 → 改哪些文件 → 如何测试 → 文档留存与提交规范）
- **[游戏系统总览与开发指引](SYSTEMS.md)**（副本/战斗/学习/商店/状态机等全部系统的机制、代码位置、扩展方法与注意事项；接手必读）
- [后端安全部署修复计划](superpowers/plans/2026-10-26-safe-backend-deployment.md)（构建最新版本、存档保留、备份回滚与跨平台称号加载）
- [玩家手册整理计划](superpowers/plans/2026-10-26-player-guide.md)（资源核对、七卷剧情、任务状态与全量覆盖检查）
- [内容与机制修复计划](superpowers/plans/2026-10-27-content-mechanics-repair.md)（任务、对话交易、死亡恢复、撤离及强化；含实际验收记录与网页验证限制）
- [武功 V2 实施与验收](design/martial_arts_v2_implementation.md)（实现范围、测试证据、首次发布及迁移方法）

## 设计

- [城市选择、武学解法与见闻指引](design/connected_exploration_2026_10_28.md)（三件旧事接入现场、六门武学替代解法）

- [既有资源内容深化](design/existing_content_depth_2026_10_28.md)（冻结实体ID，人物专长、物品出示、在地调查及章节交互）
- [全量审查修复规格](design/audit_repairs_2026_10_28.md)（用户确认的战斗、20%回收、转化价值、装备合并及内容标记）
- [烟尘粉无限供给与强化属性系数](design/smoke_enhancement_2026_10_28.md)（用户选择的现行规则与旧强化免费重置）
- [烟尘粉补货与强化预算选项](design/restock_enhancement_options_2026_10_28.md)（历史分析，已选择无限供给与各属性独立系数）

- [当前游戏全量审查（2026-10-28）](design/full_game_audit_2026_10_28.md)（历史审查与隔离复现，后续修复见本轮规格）

- [快速完成与更新回归修复](design/dungeon_quick_complete_2026_10_28.md)（首通后免费快速完成、15%掉落、途中收益、领取兼容与传承修复）

- [当前内容与机制复核及修复规格](design/content_mechanics_review_2026_10_27.md)（任务闭环、对话交易、死亡恢复、撤离、强化和剧情入口）

- **[武功体系 V2 完整提案](design/martial_arts_v2_proposal.md)**（V2已首次上线：一功一名，38门五档目录、获取与修炼、全量旧技能映射和存档迁移）
- [武功 V2 数值与机制审计](design/martial_arts_v2_balance_review.md)（首版38门可继续扩展；兵器缺口、修炼效率、流派差距、内力、招架、用药及经济复核）
- [黄易世界格局设计](design/world_design_huangyi.md)
- [玩家与 NPC 属性系统设计](design/attribute_system_design.md)（已实施，背景依据）
- [技能学习成长设计](design/skill_learning_progression_design.md)（流程规则）
- [武学谱系设计](design/martial_arts_curriculum.md)（流派·段位·传承：全部技能/老师/秘籍总谱）
- [属性与战斗公式文档](design/formulas.md)（数值公式唯一权威来源，先改文档再改代码）
- [收益与经济设计](design/economy_design.md)（资源来源/消耗/死亡规则）

## 内容

- [黄易武侠副本重设计（战斗主导版）](content/huangyi_dungeon_redesign.md)（副本设计总纲）
- [前九关副本流程速览](content/dungeon_flows.md)（早期短版；全部 78 关见上方玩家手册）

后续新增文档应按主题放入对应目录，避免在 `docs` 根目录堆放同类文件。已清理的历史文档：《代码复用与架构评审》《新内容架构实施计划》（围绕已删除的寻秦记内容与潜行系统编写，2026-10 移除）。
