<!-- 资源校验指纹：bb5ad0d37fc1c59838c9abb65fefa3b8a75f89d18ff6b789e4762c0ba0dd0c23；由 scripts/generate_player_guide.py 生成。 -->

# 当前内容状态与容易踩空的入口

[返回玩家入口](README.md)

本页按当前资源引用与运行时代码检查，记录玩家能遇到的缺口；没有修改游戏行为，也没有把未接内容补写成已发生的剧情。

## 覆盖统计

| 内容 | 配置条目 | 手册覆盖方式 |
|---|---:|---|
| 世界 / 公共房间 | 8 / 65 | 出生点路线、NPC、生活动作和实际交谈 |
| 副本 | 78 | 七卷全量逐房攻略、机关分支与剧情 |
| 任务 | 41 | 4 条已接通，37 条缺少接取/交付 |
| 技能 / 物品 / 称号 | 83 / 178 / 33 | 老师、前置、获取入口与条件 |
| 普通对话档案 | 89 | 常驻人物或附录逐份保留原文；未绑定的明确标记 |

## 解锁与叙事差异

- 当前没有序号 27；序号 28《蜀道栈桥》因代码只检查紧邻上一序号，初始即可进入。它不是新手难度关，推荐仍按成长顺序挑战。
- 同一全局序号的不同副本是并行替代入口：任一满通关结算可解下一序号，卷内 chapter 不是解锁条件。
- 序号 42《幽冥鬼市》是三路地图，当前没有「选一锁死」机制；序号 62 的同名关是另一个四房战斗模板。
- 序号 43《天罡北斗阵》有七步机关，但未实现文案所称的错误重置；序号 61 的同名关只有通用开门流程。
- 《护送商队》用进入终点计目标，没有独立跟随 AI 和护送失败血条。
- 后期部分副本的剧情只有「要点已明」「通道开启」等短句。攻略保留这些实际文字，不据副本名称编造一段完整原著故事。

## 未绑定的副本动作

这些动作有定义，但没有被房间 actions、可用对话或分支入口引用。一般是未接入的「听取任务」，主战斗路线仍可看各关攻略。

| 副本 | 当前不能触发的动作 |
|---|---|
| [《东都宫变》](stories/datang.md#datang_gongbian) | 听取任务简报（`listen_brief`） |
| [《运河追击》](stories/datang.md#datang_yunhe) | 听取任务简报（`listen_brief`） |
| [《魔师宫秘道》](stories/fuyu.md#fuyu_mishi) | 听取任务简报（`listen_brief`） |
| [《战神图录·终卷》](stories/posui.md#posui_zhongjuan) | 听取任务简报（`listen_brief`） |
| [《边荒传奇》](stories/bianhuang.md#bianhuang_chuanqi) | 听取任务简报（`listen_brief`） |
| [《洛水龙吟》](stories/luoshui.md#luoshui_longyin) | 听取任务简报（`listen_brief`） |
| [《盐帮内乱》](stories/yunhe.md#yangzhou_neiluan) | 听取任务简报（`listen_brief`） |
| [《唐门试炼》](stories/yunhe.md#bashu_tangmen_trial) | 听取任务简报（`listen_brief`） |
| [《狼烟再起》](stories/bianhuang.md#saibei_langyan) | 听取任务简报（`listen_brief`） |
| [《剑首之争》](stories/luoshui.md#jiangnan_jianshou) | 听取任务简报（`listen_brief`） |
| [《行宫惊变》](stories/datang.md#datang_xinggong) | 听取任务（`listen_brief`） |
| [《浪翻云试炼》](stories/fuyu.md#fuyu_langfan) | 听取任务（`listen_brief`） |
| [《破碎空明》](stories/posui.md#posui_kongming) | 听取任务（`listen_brief`） |
| [《边荒绝地》](stories/bianhuang.md#bianhuang_jueji) | 听取任务（`listen_brief`） |
| [《归墟龙宫》](stories/luoshui.md#luoshui_guilou) | 听取任务（`listen_brief`） |
| [《运河夜战》](stories/yunhe.md#yangzhou_yunhe2) | 听取任务（`listen_brief`） |
| [《唐门天梯》](stories/yunhe.md#bashu_tangmen2) | 听取任务（`listen_brief`） |
| [《狼山雪岭》](stories/bianhuang.md#saibei_langshan2) | 听取任务（`listen_brief`） |
| [《剑池深处》](stories/luoshui.md#jiangnan_jianchi2) | 听取任务（`listen_brief`） |
| [《神工秘殿》](stories/luoshui.md#dongming_shengong) | 听取任务（`listen_brief`） |
| [《绝壁悬栈》](stories/datang.md#datang_xb2) | 听取任务（`listen_brief`） |
| [《寒潭蛟影》](stories/fuyu.md#fuyu_ht2) | 听取任务（`listen_brief`） |
| [《沙海孤城》](stories/posui.md#posui_sh2) | 听取任务（`listen_brief`） |
| [《毒瘴密林》](stories/bianhuang.md#bianhuang_dz2) | 听取任务（`listen_brief`） |
| [《天罡北斗》](stories/luoshui.md#luoshui_tg2) | 听取任务（`listen_brief`） |
| [《幽冥鬼市》](stories/yunhe.md#yangzhou_ym2) | 听取任务（`listen_brief`） |
| [《铁血雄关》](stories/datang.md#datang_tx2) | 听取任务（`listen_brief`） |
| [《烟雨剑冢》](stories/fuyu.md#fuyu_yy2) | 听取任务（`listen_brief`） |
| [《天火熔炉》](stories/posui.md#posui_th2) | 听取任务（`listen_brief`） |
| [《冰封王座》](stories/bianhuang.md#bianhuang_bf2) | 听取任务（`listen_brief`） |
| [《雷鸣峡谷》](stories/luoshui.md#luoshui_lm2) | 听取任务（`listen_brief`） |
| [《苍穹之眼》](stories/yunhe.md#yangzhou_cj2) | 听取任务（`listen_brief`） |
| [《万刃窟》](stories/datang.md#datang_wr2) | 听取任务（`listen_brief`） |
| [《幽谷回音》](stories/fuyu.md#fuyu_yg2) | 听取任务（`listen_brief`） |
| [《太虚幻境》](stories/posui.md#posui_tx3) | 听取任务（`listen_brief`） |

## 其他实际入口差异

- 世界任务只有四条接好了 startQuestId / completeQuestId；NPC 的 questIds 或一句「有赏」不会自动建立任务状态。其余 37 条见任务页。
- 后十条任务的目标如 `dungeon_cleared_datangdoubledragonxb2` 使用不存在的副本 ID，即使补上接交对话，还需要修正目标。
- 巴蜀药田当前产出默认采集材料古铜残片，止血草的实际稳定采集点是洛阳芦苇滩。
- 世界 JSON 的补充 rumors 尚未由 listen_rumor 读取；新城某些听传闻按钮目前只回泛用台词。
- 强化已有后端接口，没有网页按钮。底部输入框是公开聊天，不能执行后端指令。
- 没有查到可达来源的残页、物品或老师，在收藏表中单列；资源总量不直接代表可收集总量。

- 普通交谈会记录已读选项，炼药、熔铸等配方用此入口各自动触发一次；不能据此承诺网页支持无限重复炼药。
- 烟尘粉只有用途描述，没有接入可用物品效果；当前不能用它脱战。

## 维护与校对

玩家入口和推荐路线是手写内容；本页、七卷故事、世界、任务与收藏事实页由 `python scripts/generate_player_guide.py` 从资源生成，`--check` 检查漂移和本手册的内部链接。行为规则已对照 manager 的副本解锁、交谈/任务、房间动作、学习/使用物品代码以及前端按钮核对；没有逐关实机通关或验证数值难度。
