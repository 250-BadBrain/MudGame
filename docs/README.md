# 项目文档索引

## 开发流程

- **[统一开发指导文档](DEVELOPMENT.md)**（仓库结构、各模块功能、标准开发流程：先读什么 → 文档先行 → 改哪些文件 → 如何测试 → 文档留存与提交规范）
- **[游戏系统总览与开发指引](SYSTEMS.md)**（副本/战斗/学习/商店/状态机等全部系统的机制、代码位置、扩展方法与注意事项；接手必读）

## 设计

- [黄易世界格局设计](design/world_design_huangyi.md)
- [玩家与 NPC 属性系统设计](design/attribute_system_design.md)（已实施，背景依据）
- [技能学习成长设计](design/skill_learning_progression_design.md)（流程规则）
- [属性与战斗公式文档](design/formulas.md)（数值公式唯一权威来源，先改文档再改代码）
- [收益与经济设计](design/economy_design.md)（资源来源/消耗/死亡规则）

## 内容

- [黄易武侠副本重设计（战斗主导版）](content/huangyi_dungeon_redesign.md)（副本设计总纲）
- [副本流程记录](content/dungeon_flows.md)（已实现副本的实际游玩流程）

后续新增文档应按主题放入对应目录，避免在 `docs` 根目录堆放同类文件。已清理的历史文档：《代码复用与架构评审》《新内容架构实施计划》（围绕已删除的寻秦记内容与潜行系统编写，2026-10 移除）。
