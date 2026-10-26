"""Generate the player's factual reference pages from the current game resources."""

import argparse
import collections
import hashlib
import json
import os
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RESOURCES = ROOT / "MudGame-server/resources"
BOOK = ROOT / "docs/player"
CHAIN_FILES = {
    "luoyang_rain_temple": "luoyang", "datang_double_dragon": "datang",
    "fuyu_fan_yun": "fuyu", "posui_xukong": "posui",
    "bianhuang_legend": "bianhuang", "luoshui_raiders": "luoshui",
    "yangzhou_caoyun": "yunhe",
}
CHAIN_INTROS = {
    "luoyang_rain_temple": "从洛阳城外的一封染血密信起步。三间房的小副本，适合熟悉挑战、进度、现场搜索与离开结算。",
    "datang_double_dragon": "从扬子江追兵、杨公宝库，推进到竟陵守城、宋缺试刀、飞马牧场与长安风云，再延伸到粮道、巴陵、瓦岗和丝路护送。前期剧情、机关和人物较完整，后期部分章节为简短战斗关。",
    "fuyu_fan_yun": "怒蛟帮的水寨防守、拦江宗师残影、魔师宫与方夜羽串起本卷；另有水战、浪翻云试炼和双头目关。",
    "posui_xukong": "惊雁宫观星夺图、鸡鸣山之约、战神殿心象与图录续卷，最终延伸到北斗七步机关和破碎虚空题材的战斗试炼。",
    "bianhuang_legend": "从护送纪千千突围，到马贼王庭、边荒夺旗、狼山口和雪原，主题是边地护送、骑军与狼群。",
    "luoshui_raiders": "洛水水匪、魔门枯井伏笔、江南剑池、东都账册与东溟海市都归入本卷；后期还有三路鬼市、归墟龙宫等独立小故事。",
    "yangzhou_caoyun": "盐帮、唐门与运河支线被归入这一目录。当前六章主要是四房战斗与开门机关；并不自动接取世界任务。",
}
NOTES = {
    "abandoned_temple_rain": "翻检刺客遗物为可选心得与钱币奖励；伏虎劲改由河洛武馆赵横江传授，不再从此处取残页。",
    "datang_changshengjue": "打捞给止血草4、干粮2；满结算给金创药2、干粮1。此处不再给长生诀，完整传承位于序47《行宫惊变》。",
    "posui_jingyan": "初次按「观察穹顶星图 → 对照殿壁星官壁画 → 转动石门星盘 → 向左转动星盘」完成机关。选右会弹回先前状态，需要再对照壁画后重选；开门的同时获得削弱战神幻影的旗标。",
    "datang_jingling": "东门强攻和水门护民的目标属于同一独占组，完成任一条即可计满该组；无需把两条路都杀一遍。",
    "datang_tiandao": "「以刚为纲」「以灵为要」都能完成悟刀并得到破招旗标。「接三刀」是剧情称呼；战斗仍由普通战斗 tick 推进至宋缺认负。",
    "youming_guishi": "介绍文案写「三选一、不能回头」，但目前没有分支锁：三岔路口的东、西、北都可往返。清两名入口守卫并击败任一路头目就能满进度，三路的取物动作可分别探索；不要把互斥计分理解成地图只许走一条。",
    "tianbei_beidou": "按「贪狼位·踏天枢 → 巨门位·转巨门 → 禄存位·举禄存 → 文曲位·抚文曲 → 廉贞位·燃廉贞 → 武曲位·握武曲 → 破军位·破军门」操作。房间文字称走错重置，但当前没有配置失败重置；点错会被前置状态挡住。星盘是满完成度离开时结算的奖励，不是密室里的独立拾取按钮。",
    "silk_road_hugong": "当前用沿途战斗和抵达玉门关目标模拟护送，首领在各段房间均有配置；没有独立的跟随 AI、护送血量或马车受袭失败机制。",
}
ACTION_NAMES = {"gather": "采集", "work": "帮工", "search": "搜索", "listen_rumor": "听传闻"}
DIRECTIONS = {"north": "北", "south": "南", "east": "东", "west": "西", "northeast": "东北", "northwest": "西北", "southeast": "东南", "southwest": "西南", "up": "上", "down": "下"}
ATTRIBUTES = {"attack": "攻击", "defense": "防御", "hitRate": "命中", "dodge": "躲闪", "parry": "招架", "initiative": "先手", "maxHp": "气血上限", "maxMp": "内力上限", "critRate": "暴击率"}
SLOTS = {"WEAPON": "兵器", "BODY": "衣甲", "FEET": "鞋靴", "HEAD": "头部", "NECKLACE": "项链", "RING": "戒指", "ACCESSORY": "饰品", "HIDDEN_WEAPON": "暗器"}


def read(path):
    return json.loads(path.read_text(encoding="utf-8"))


def safe(value):
    return str(value).replace("|", "\\|").replace("\r", "").replace("\n", "<br>")


def quote(value):
    return "\n".join("> " + line for line in str(value).strip().splitlines())


class Guide:
    def __init__(self):
        self.items = read(RESOURCES / "items.json")
        self.skills = read(RESOURCES / "skills.json")
        self.npcs = read(RESOURCES / "npcs.json")
        self.dialogues = read(RESOURCES / "dialogues.json")
        self.quests = read(RESOURCES / "quests.json")
        self.titles = read(RESOURCES / "titles.json")["titles"]
        self.scripts = read(RESOURCES / "dungeon_dialogues.json")
        self.worlds = sorted((read(p) for p in (RESOURCES / "worlds").glob("*.json")), key=lambda w: (w.get("order", 0), w["id"]))
        self.dungeons = sorted((read(p) for p in (RESOURCES / "dungeons").glob("*.json")), key=lambda d: (d["order"], d["id"]))
        self.by_dungeon = {d["id"]: d for d in self.dungeons}
        self.locations = collections.defaultdict(list)
        self.starts = collections.defaultdict(list)
        self.ends = collections.defaultdict(list)
        self.flags = collections.defaultdict(list)
        self.item_sources = collections.defaultdict(list)
        self.unbound = []
        for kind, owners in (("world", self.worlds), ("dungeon", self.dungeons)):
            for owner in owners:
                for rid, room in owner["rooms"].items():
                    for npc in room.get("npcs", []):
                        self.locations[npc].append((kind, owner, rid))
        self.live_dialogues = {self.npcs[n].get("dialogueId") for n, loc in self.locations.items() if loc}
        for did, dialogue in self.dialogues.items():
            if did not in self.live_dialogues:
                continue
            for option in dialogue.get("options", []):
                entry = (did, option)
                if option.get("startQuestId"):
                    self.starts[option["startQuestId"]].append(entry)
                if option.get("completeQuestId"):
                    self.ends[option["completeQuestId"]].append(entry)
                label = self.npc_name(dialogue.get("npcId", "")) + "交谈「" + option["text"] + "」"
                for flag, value in option.get("setFlags", {}).items():
                    if value:
                        self.flags[flag].append((label, "worlds.md", "dialogue-" + did))
                for item in option.get("giveItems", {}):
                    self.item_sources[item].append((label, "worlds.md", "dialogue-" + did))
        self.wired_quests = {qid for qid in self.quests if self.starts.get(qid) and self.ends.get(qid)}
        for quest in self.quests.values():
            if quest["id"] in self.wired_quests:
                for flag, value in quest.get("rewards", {}).get("flags", {}).items():
                    if value:
                        self.flags[flag].append(("完成任务《" + quest["name"] + "》", "quests.md", quest["id"]))
                for item in quest.get("rewards", {}).get("items", {}):
                    self.item_sources[item].append(("任务《" + quest["name"] + "》", "quests.md", quest["id"]))
        for dungeon in self.dungeons:
            page = self.dungeon_page(dungeon)
            origin = ("100% 结算《" + dungeon["name"] + "》", page, dungeon["id"])
            self.flags["dungeon_cleared_" + dungeon["id"]].append(origin)
            for flag, value in dungeon.get("rewards", {}).get("flags", {}).items():
                if value:
                    self.flags[flag].append(origin)
            for item in dungeon.get("rewards", {}).get("items", {}):
                self.item_sources[item].append(origin)
            for aid, action in dungeon.get("puzzleActions", {}).items():
                if not self.action_bound(dungeon, aid):
                    self.unbound.append((dungeon, aid))
                    continue
                source = ("《" + dungeon["name"] + "》「" + action.get("label", aid) + "」现场获得", page, dungeon["id"])
                for flag, value in action.get("setFlags", {}).items():
                    if value:
                        self.flags[flag].append(source)
                for item in action.get("giveItems", {}):
                    self.item_sources[item].append(source)
        for npc, definition in self.npcs.items():
            if not self.locations.get(npc):
                continue
            for item in definition.get("shopItems", []):
                self.item_sources[item].append((self.npc_name(npc) + "商店", "collections.md", "shop-" + npc))
        for flag, room in {"found_caoyin_clue": "yz_reed_marsh", "found_smuggling_evidence": "ly_salt_wharf", "found_beacon_signal": "tg_jingou_beacon"}.items():
            self.flags[flag].append((self.room_name(room) + "点击「搜索」", "worlds.md", room))
        for item, place in {"iron_pickaxe": "创角自带", "bronze_fragment": "洛阳城郊等默认采集点（60%）", "dry_ration": "各采集点（40%）", "herb": "洛阳芦苇滩采集（60%）", "reed_root": "扬州城郊苇荡采集（60%）"}.items():
            self.item_sources[item].append((place, "worlds.md", "labor"))
        text = (ROOT / "MudGame-server/internal/manager/runtime_manager_player.go").read_text(encoding="utf-8")
        rumor_function = text.split("func roomRumor(", 1)[1].split("\nfunc ", 1)[0]
        self.rumors = {rid: json.loads(value) for rid, value in re.findall(r'case "([^"]+)":\s*return ("(?:\\.|[^"\\])*")', rumor_function)}
        search_function = text.split("func (r *RuntimeManager) searchRoomNoLock(", 1)[1].split("\nfunc ", 1)[0]
        self.search_text = {rid: json.loads(value) for rid, value in re.findall(r'if room.ID == "([^"]+)"[^\{]*\{.*?return ("(?:\\.|[^"\\])*")', search_function, re.S)}

    def dungeon_page(self, dungeon):
        return "stories/" + CHAIN_FILES[dungeon["chainId"]] + ".md"

    def link(self, text, target, current, anchor=""):
        relative = os.path.relpath(BOOK / target, (BOOK / current).parent).replace("\\", "/")
        return "[" + safe(text) + "](" + relative + ("#" + anchor if anchor else "") + ")"

    def dungeon_link(self, dungeon, current):
        return self.link("《" + dungeon["name"] + "》", self.dungeon_page(dungeon), current, dungeon["id"])

    def npc_name(self, npc):
        return self.npcs.get(npc, {}).get("name", npc or "未指定")

    def room_name(self, rid):
        for owner in self.worlds + self.dungeons:
            if rid in owner["rooms"]:
                return owner["name"] + "·" + owner["rooms"][rid]["name"]
        return rid

    def location(self, npc, current):
        values = []
        for kind, owner, rid in self.locations.get(npc, []):
            if kind == "world":
                values.append(self.link(owner["name"] + "·" + owner["rooms"][rid]["name"], "worlds.md", current, rid))
            else:
                values.append(self.dungeon_link(owner, current) + "·" + owner["rooms"][rid]["name"])
        return "；".join(dict.fromkeys(values)) or "当前未放置到任何房间"

    def sources(self, entries, current):
        return "；".join(dict.fromkeys(self.link(label, page, current, anchor) for label, page, anchor in entries)) or "未找到当前可达获取入口"

    def action_bound(self, dungeon, aid, seen=None):
        seen = set() if seen is None else seen
        if aid in seen:
            return False
        seen.add(aid)
        action = dungeon.get("puzzleActions", {}).get(aid, {})
        for rid, room in dungeon["rooms"].items():
            if action.get("rooms") and rid not in action["rooms"]:
                continue
            if aid in room.get("actions", []):
                return True
            npc = action.get("dialogueNpcId")
            if npc in room.get("npcs", []) and self.npcs[npc].get("dialogueId") in self.dialogues:
                return True
        return any(any(c["id"] == aid for c in parent.get("choices", [])) and self.action_bound(dungeon, key, seen.copy())
                   for key, parent in dungeon.get("puzzleActions", {}).items())

    def items_text(self, items):
        return "、".join(self.items.get(key, {}).get("name", key) + "×" + str(value) for key, value in items.items()) or "无"

    def skill_bonuses(self, bonuses):
        names = {"post_strength": "后天臂力", "post_constitution": "后天根骨", "post_agility": "后天身法", "post_perception": "后天悟性", "post_willpower": "后天心志", "hit_rate": "命中", "crit_rate": "暴击率", "max_mp": "内力绝对上限", "max_hp": "气血上限", "meditate_recovery": "打坐回复", "learning_invest_bonus": "学习结算投入上限", "intelligence_check": "悟性检定"}
        values = []
        for key, value in bonuses.items():
            base = key.replace("_per_100_levels", "").replace("_per_10_levels", "").replace("_per_level", "")
            label = names.get(base, ATTRIBUTES.get(base, base))
            if key.endswith("_per_100_levels"):
                values.append("100级基线 " + label + "+" + str(value))
            elif key.endswith("_per_10_levels"):
                values.append("每 10 级 " + label + "+" + str(value))
            elif key.endswith("_per_level"):
                values.append("每级 " + label + "+" + str(value))
            elif key == "trained_mp_to_max_hp_percent":
                values.append(("固定转血" if value >= .3 else "100级额外转血封顶") + str(round(value * 100)) + "个百分点")
            else:
                values.append(key + "=" + str(value))
        return "；".join(values) or "未配置额外加成"

    def rewards(self, rewards):
        parts = [label + " " + str(rewards.get(key, 0)) for key, label in (("experience", "经验"), ("potential", "潜能"), ("money", "钱币"))]
        if rewards.get("items"):
            parts.append(self.items_text(rewards["items"]))
        if rewards.get("reputations"):
            factions = read(RESOURCES / "factions.json")
            parts.extend(factions.get(key, {}).get("name", key) + "声望 +" + str(value) for key, value in rewards["reputations"].items())
        return "；".join(parts)

    def unlock(self, dungeon, current):
        previous_order = max((d["order"] for d in self.dungeons if d["order"] < dungeon["order"]), default=0)
        previous = [d for d in self.dungeons if d["order"] == previous_order]
        if dungeon["order"] <= 1:
            result = "初始开放，无前置副本。"
        elif previous:
            result = "先将 " + " 或 ".join(self.dungeon_link(d, current) for d in previous) + " 任一副本 100% 完成并离开结算。"
        else:
            result = "初始组无前置；前一组序号为 " + str(dungeon["order"] - 1) + "。"
        for condition in dungeon.get("unlockConditions", []):
            if condition["type"] == "flag":
                sources = self.flags.get(condition["targetId"], [])
                result += " 还需" + (self.sources(sources, current) if sources else "旗标 `" + condition["targetId"] + "`") + "。"
            else:
                result += " 额外条件：`" + json.dumps(condition, ensure_ascii=False) + "`。"
        return result

    def route(self, owner, rid):
        start = owner["spawnRoomId"]
        queue = collections.deque([(start, [])])
        seen = {start}
        while queue:
            current, path = queue.popleft()
            if current == rid:
                return "出生点" if not path else " → ".join(DIRECTIONS.get(direction, direction) + "到「" + owner["rooms"][target]["name"] + "」" for direction, target in path)
            for direction, target in owner["rooms"][current].get("exits", {}).items():
                if target not in seen:
                    seen.add(target)
                    queue.append((target, path + [(direction, target)]))
        return "没有从出生点连通的路线"

    def requirements(self, requirement, current):
        parts = []
        if requirement.get("questId"):
            qid = requirement["questId"]
            labels = {"not_started": "尚未接取", "active": "进行中", "completed": "已完成"}
            parts.append("任务《" + self.quests[qid]["name"] + "》" + labels.get(requirement.get("questStatus"), "存在"))
        if requirement.get("flag"):
            flag = requirement["flag"]
            if requirement.get("flagValue", True):
                parts.append("需 " + self.sources(self.flags.get(flag, []), current))
            else:
                parts.append("尚未取得旗标 `" + flag + "`")
        if requirement.get("reputationId"):
            faction = read(RESOURCES / "factions.json")[requirement["reputationId"]]["name"]
            parts.append(faction + "声望 ≥" + str(requirement.get("reputationMin", 0)))
        if requirement.get("items"):
            parts.append("携带 " + self.items_text(requirement["items"]))
        return "；".join(parts) or "无额外条件"

    def option_effect(self, option, current):
        parts = []
        if option.get("startQuestId"):
            qid = option["startQuestId"]
            parts.append("接取 " + self.link("《" + self.quests[qid]["name"] + "》", "quests.md", current, qid))
        if option.get("completeQuestId"):
            qid = option["completeQuestId"]
            parts.append("交付 " + self.link("《" + self.quests[qid]["name"] + "》", "quests.md", current, qid) + "；任务全部目标必须完成")
        if option.get("takeMoney"):
            parts.append("花费 " + str(option["takeMoney"]) + " 两")
        if option.get("takeItems"):
            parts.append("消耗 " + self.items_text(option["takeItems"]))
        if option.get("giveItems"):
            parts.append("现场获得 " + self.items_text(option["giveItems"]))
        if option.get("setFlags"):
            parts.append("记录剧情进度：" + "、".join("`" + flag + "`=" + str(value).lower() for flag, value in option["setFlags"].items()))
        return "；".join(parts)

    def dialogue_lines(self, did, current, anchor=True):
        dialogue = self.dialogues[did]
        lines = ([f'<a id="dialogue-{did}"></a>', ""] if anchor else []) + ["**" + dialogue.get("title", "交谈") + "**", "", quote(dialogue.get("opening", "")), ""]
        for option in dialogue.get("options", []):
            lines += ["- 「" + option["text"] + "」；" + self.requirements(option.get("requires", {}), current), "", quote(option.get("response", "")), ""]
            effect = self.option_effect(option, current)
            if effect:
                lines += ["  效果：" + effect, ""]
        return lines

    def state_requirement(self, dungeon, puzzle, states):
        if not states:
            return "无状态前置"
        values = []
        for state in states:
            if dungeon.get("puzzleInitialStates", {}).get(puzzle) == state:
                values.append("该机关尚未操作")
            else:
                producers = [a.get("label", aid) for aid, a in dungeon.get("puzzleActions", {}).items() if a.get("puzzleId") == puzzle and a.get("nextState") == state]
                values.append("先完成「" + "」或「".join(producers) + "」" if producers else "状态 `" + puzzle + ":" + state + "`")
        return " / ".join(values)

    def action_requirements(self, dungeon, action):
        parts = [self.state_requirement(dungeon, action.get("puzzleId", ""), action.get("fromStates", []))]
        if action.get("requiredDefeatNpcs"):
            parts.append("先击败本房间 " + "、".join(self.npc_name(npc) for npc in action["requiredDefeatNpcs"]))
        return "；".join(parts)

    def door_requirement(self, dungeon, requirement):
        parts = []
        if requirement.get("defeatNpcs"):
            parts.append("击败本房间全部 " + "、".join(self.npc_name(npc) for npc in requirement["defeatNpcs"]))
        for puzzle, state in requirement.get("requiredPuzzleStates", {}).items():
            parts.append(self.state_requirement(dungeon, puzzle, [state]))
        if requirement.get("requiredItemId"):
            parts.append("携带 " + self.items_text({requirement["requiredItemId"]: requirement.get("requiredAmount", 1)}))
        return "；".join(parts) or "无额外门槛"

    def chapter(self, dungeon, current):
        did = dungeon["id"]
        lines = [f'<a id="{did}"></a>', "", f'## 序号 {dungeon["order"]} · {dungeon["name"]}', "", dungeon.get("description", ""), "",
                 "- **触发入口**：右侧「境」→本卷→本副本→进入；不需要先在城市找到 NPC。", "- **解锁**：" + self.unlock(dungeon, current),
                 "- **标识**：`" + did + "`；卷内标签「第 " + str(dungeon.get("chapter", "—")) + " 章」，解锁按全局序号判断。", "- **满额结算奖励**：" + self.rewards(dungeon.get("rewards", {})) + "。"]
        title = self.titles.get("dungeon_cleared_" + did)
        if title:
            lines.append("- **通关称号**：" + title + "。")
        if did in NOTES:
            lines += ["", "**本关要点**：" + NOTES[did]]
        unbound = [aid for d, aid in self.unbound if d["id"] == did]
        if unbound:
            labels = "、".join(dungeon["puzzleActions"][aid].get("label", aid) for aid in unbound)
            lines += ["", "**当前内容状态**：「" + labels + "」有文字定义，但没有接到房间按钮或可用交谈，当前不能触发；不要寻找这个按钮。"]
        lines += ["", "### 100% 完成清单", "", "| 目标 | 数量 | 权重 | 计分规则 |", "|---|---:|---:|---|"]
        for objective in dungeon.get("objectives", []):
            rule = "可选，不影响 100%" if objective.get("optional") else ("同组任一完成即可：" + objective["exclusiveGroup"] if objective.get("exclusiveGroup") else "必须完成")
            lines.append("| " + safe(objective.get("description", objective["id"])) + " | " + str(objective.get("required", 1)) + " | " + str(objective.get("weight", 1)) + " | " + rule + " |")
        lines += ["", "### 房间与操作顺序", "", "从「" + dungeon["rooms"][dungeon["spawnRoomId"]]["name"] + "」进入。下列路线是从出生点到该房间的方向；有门槛时先做完门槛再移动。", ""]
        for rid, room in dungeon["rooms"].items():
            lines += ["**" + room["name"] + "**（" + self.route(dungeon, rid) + "）", "", room.get("description", ""), ""]
            counts = collections.Counter(room.get("npcs", []))
            if counts:
                names = []
                for npc, count in counts.items():
                    definition = self.npcs[npc]
                    mode = "（剧情保护，勿攻击）" if definition.get("combatMode") == "protected" else ("（无战意）" if definition.get("nonCombat") else ("（认负型，用挑战推进，别用切磋代替目标）" if definition.get("combatMode") == "defeat" else ""))
                    names.append(self.npc_name(npc) + ("×" + str(count) if count > 1 else "") + mode)
                lines += ["人物：" + "；".join(names) + "。", ""]
            for aid in room.get("actions", []):
                action = dungeon.get("puzzleActions", {}).get(aid)
                if not action:
                    lines += ["- 房间动作：「" + ACTION_NAMES.get(aid, aid) + "」。"]
                    continue
                npc = action.get("dialogueNpcId")
                operation = "反复与" + self.npc_name(npc) + "「交谈」，听完对应剧情" if npc else "点击「" + action.get("label", aid) + "」"
                lines.append("- " + operation + "。前置：" + self.action_requirements(dungeon, action) + "。")
                if action.get("giveItems"):
                    lines.append("  现场获得：" + self.items_text(action["giveItems"]) + "。")
                if action.get("choices"):
                    for choice in action["choices"]:
                        branch = dungeon["puzzleActions"][choice["id"]]
                        lines.append("  - 选择「" + choice["label"] + "」：" + branch.get("message", "") + ("；获得 " + self.items_text(branch["giveItems"]) if branch.get("giveItems") else ""))
            for direction, target in room.get("exits", {}).items():
                req = room.get("exitRequirements", {}).get(direction)
                if req:
                    lines += ["- 去" + DIRECTIONS.get(direction, direction) + "侧「" + dungeon["rooms"][target]["name"] + "」：" + self.door_requirement(dungeon, req) + "。"]
            for npc in counts:
                definition = self.npcs[npc]
                if definition.get("weaknessFlag"):
                    flag = definition["weaknessFlag"]
                    lines += ["- " + self.npc_name(npc) + "破招：先取得 " + self.sources(self.flags.get(flag, []), current) + "，再发起挑战。"]
            lines.append("")
        lines += ["### 剧情全文与分支结果（剧透）", "", "<details>", "<summary>展开本副本当前已写出的台词与机关结果</summary>", ""]
        for aid, action in dungeon.get("puzzleActions", {}).items():
            if action.get("giveMoney"):
                lines += ["成功操作的钱币奖励：" + str(action["giveMoney"]), ""]
            lines += ["**" + action.get("label", aid) + "**" + ("（当前未接入口）" if aid in unbound else ""), ""]
            script = self.scripts.get(did, {}).get(aid, {})
            for text in script.get("lines", []) or [action.get("message", "")]:
                if text:
                    lines += [quote(text), ""]
            if script.get("doneMessage"):
                lines += ["对话结束后：" + script["doneMessage"], ""]
            if action.get("failureMessage"):
                lines += ["条件不足时：" + action["failureMessage"], ""]
            if action.get("repeatMessage"):
                lines += ["完成后再问/再点：" + action["repeatMessage"], ""]
        bound_npcs = {action.get("dialogueNpcId") for action in dungeon.get("puzzleActions", {}).values()}
        for npc in dict.fromkeys(npc for room in dungeon["rooms"].values() for npc in room.get("npcs", [])):
            if self.npcs[npc].get("description"):
                lines += ["**人物小传：" + self.npc_name(npc) + "**", "", quote(self.npcs[npc]["description"]), ""]
            did_static = self.npcs[npc].get("dialogueId")
            if did_static in self.dialogues and npc not in bound_npcs:
                lines += ["**" + self.npc_name(npc) + "的额外交谈**", ""] + self.dialogue_lines(did_static, current, anchor=False)
        lines += ["</details>", "", "**结束方式**：确认必需目标全满、现场可选奖励已取，再点击「离开副本」结算。只有 100% 结算才记录通关、发结算物品并解锁下一序号。", "", "---", ""]
        return lines

    def story_pages(self):
        pages = {}
        for chain, filename in CHAIN_FILES.items():
            chapters = [d for d in self.dungeons if d["chainId"] == chain]
            current = "stories/" + filename + ".md"
            lines = ["# " + chapters[0]["chainName"] + "：触发、流程与完整剧透", "", "[返回玩家入口](../README.md) · [全局解锁顺序](../README.md#unlock-order)", "", CHAIN_INTROS[chain], "",
                     "本卷 " + str(len(chapters)) + " 个副本。编号采用当前全局序号，同号可能有其他卷的并行关。每节先给操作攻略，再折叠列出已有剧情；内容以当前资源为准，未补写原著情节。", ""]
            for dungeon in chapters:
                lines.append("- " + self.dungeon_link(dungeon, current) + "（序号 " + str(dungeon["order"]) + "）")
            lines.append("")
            for dungeon in chapters:
                lines += self.chapter(dungeon, current)
            pages[current] = "\n".join(lines)
        return pages

    def quest_page(self):
        current = "quests.md"
        lines = ["# 世界任务：接取地点、触发与结局", "", "[返回玩家入口](README.md) · [八城地图与交谈](worlds.md)", "",
                 f"当前资源定义 {len(self.quests)} 个任务，实际接好接取和交付对话的是 {len(self.wired_quests)} 条。其他任务列在后半页，标出缺口；不要把 NPC 的一句托付当成任务已接取。右侧「令」只列已经写入角色任务状态的任务。", "",
                 "## 已接通的四条任务", "", "与 NPC 反复点击「交谈」即可逐句推进。首次通常只说开场白，下一次才接任务；有提醒台词时可能还需再交谈一次。交付前先完成目标，没完成时会提示任务未完成。", ""]
        for qid, quest in self.quests.items():
            if qid not in self.wired_quests:
                continue
            lines += [f'<a id="{qid}"></a>', "", "### " + quest["name"], "", quest["description"], "", "1. 到 " + self.location(quest["giverNpcId"], current) + "，与" + self.npc_name(quest["giverNpcId"]) + "反复交谈，听到接取台词。"]
            for index, objective in enumerate(quest.get("objectives", []), 2):
                lines.append(str(index) + ". " + objective["description"] + " 实际触发：" + self.sources(self.flags.get(objective["targetId"], []), current) + "。")
            lines += ["3. 回到 " + self.location(quest["turnInNpcId"], current) + "，继续交谈至交付台词。", "", "**奖励**：" + self.rewards(quest["rewards"]) + "。一次性任务。", ""]
            if qid == "escort_rumor_run":
                lines += ["注意：在酒楼只点击房间「听传闻」不完成任务；要和许说书人交谈到写入传闻进度。", ""]
            if qid == "refugee_rumor_case":
                lines += ["注意：哑婆没有独立 NPC。她的三幅画被写在老瘸子的下一段对话中；继续与老瘸子交谈即可得到线索，无需寻找另一个人物。", ""]
            dialogue = self.starts[qid][0][0]
            lines += ["<details>", "<summary>展开接取、推进和结局原文</summary>", ""] + self.dialogue_lines(dialogue, current, anchor=False) + ["</details>", ""]
        lines += ["## 已有定义，但当前未接通的任务", "", "以下目标和奖励是配置记录，当前没有可用的接取/交付流程，因此不会正常出现在任务面板，也不能领取这些任务奖励。相关副本本身仍可按章节规则游玩。", ""]
        for qid, quest in self.quests.items():
            if qid in self.wired_quests:
                continue
            lines += [f'<a id="{qid}"></a>', "", "### " + quest["name"] + "（未接通）", "", "- **标识**：`" + qid + "`。", "- **现有剧情**：" + quest["description"],
                      "- **配置发起人**：" + self.npc_name(quest["giverNpcId"]) + "；位置：" + self.location(quest["giverNpcId"], current) + "。",
                      "- **入口缺口**：" + ("没有接取对话；" if not self.starts.get(qid) else "") + ("没有交付对话。" if not self.ends.get(qid) else "")]
            for objective in quest.get("objectives", []):
                source = self.flags.get(objective["targetId"], [])
                lines += ["- **配置目标**：" + objective["description"] + "（`" + objective["targetId"] + "`）。", "- **目标当前来源**：" + (self.sources(source, current) if source else "没有找到写入点；目标 ID 与现有副本不对应，需要补接后才能完成") + "。"]
            lines += ["- **配置奖励，当前不可领**：" + self.rewards(quest["rewards"]) + "。", ""]
        return "\n".join(lines)

    def world_page(self):
        current = "worlds.md"
        lines = ["# 八城探索、生活点与全部常驻交谈", "", "[返回玩家入口](README.md) · [任务攻略](quests.md) · [武学与收藏](collections.md)", "",
                 "通过右侧「图」选择城市进入，跨城消耗100钱币（1两银）；同城不收费。以下方向都从该城出生点计算，可以返回出生点重新按路线走。公共世界不靠杀 NPC 产出经验和掉落，剧情人物、商人和老师应优先交谈、请教或交易。", "",
                 '<a id="labor"></a>', "", "## 不打副本时能做什么", "", "- 「帮工」持续 60 秒，完成给20钱币、2潜能。", "- 「采集」持续 30 秒，完成给 4 经验、6 潜能，再给材料或干粮。扬州苇荡60%芦根，其余地点默认60%止血草；各处另外40%干粮。普通采集不再给古铜残片。", "- 停止会放弃本次劳作收益；学习、打坐、疗伤等忙碌状态先停止再做其他事。", "- 与老师「请教」入门，已学武功从「技」自主修炼；与商人「交易」，携带材料后反复交谈可触发已有合成。", ""]
        shown_dialogues = set()
        for world in self.worlds:
            lines += [f'<a id="world-{world["id"]}"></a>', "", "## " + world["name"], "", world.get("description", ""), "", "出生点：**" + world["rooms"][world["spawnRoomId"]]["name"] + "**。", ""]
            for rid, room in world["rooms"].items():
                lines += [f'<a id="{rid}"></a>', "", "### " + room["name"], "", "路线：" + self.route(world, rid) + "。" + ("安全区。" if room.get("safeZone") else "非安全区。"), "", room.get("description", ""), ""]
                if room.get("actions"):
                    lines += ["房间动作：" + "、".join(room.get("actionLabels", {}).get(a, ACTION_NAMES.get(a, a)) for a in room["actions"]) + "。", ""]
                if "listen_rumor" in room.get("actions", []):
                    lines += ["**当前「听传闻」实际台词**：", "", quote(self.rumors.get(rid, "你驻足听了一阵，只听见市井喧嚣里夹着几句真假难辨的江湖传闻。")), ""]
                if "search" in room.get("actions", []):
                    found = [(flag, src) for flag, src in self.flags.items() if any(page == current and anchor == rid for _, page, anchor in src)]
                    lines += ["**搜索**：" + ("首次记录 " + "、".join("`" + flag + "`" for flag, _ in found) if found else "当前没有专门调查脚本，返回没有发现") + "。", ""]
                    if rid in self.search_text:
                        lines += [quote(self.search_text[rid]), ""]
                if room.get("rumors"):
                    lines += ["<details>", "<summary>资源中的补充传闻（当前「听传闻」按钮未接入此字段）</summary>", ""]
                    for rumor in room["rumors"]:
                        lines += [quote(rumor), ""]
                    lines += ["</details>", ""]
                for npc in room.get("npcs", []):
                    definition = self.npcs[npc]
                    lines += ["**" + self.npc_name(npc) + "**" + ("（" + definition["title"] + "）" if definition.get("title") else ""), "", definition.get("description", ""), ""]
                    if definition.get("skills"):
                        lines += ["可请教：" + "、".join(self.skills[s]["name"] + "（入门指导）" for s, level in definition["skills"].items()) + "。", ""]
                    if definition.get("shopItems"):
                        lines += [self.link("查看该 NPC 商店货单", "collections.md", current, "shop-" + npc), ""]
                    did = definition.get("dialogueId")
                    if did in self.dialogues and did not in shown_dialogues:
                        shown_dialogues.add(did)
                        lines += ["<details>", "<summary>交谈全文、条件与效果（剧透）</summary>", ""] + self.dialogue_lines(did, current) + ["</details>", ""]
        # Some live dialogue is attached only to dungeon NPCs; preserve its reference anchor here too.
        lines += ["## 副本专属及尚未绑定的对话档案", "", "副本专属人物的位置见对应副本；有专用剧情脚本时，实际播放内容以七卷攻略为准，此处保留普通对话定义。标为未绑定的对话虽然有文字，当前 NPC 没有引用这份对话，不能据此领取任务或物品。", ""]
        for did, dialogue in self.dialogues.items():
            if did in shown_dialogues:
                continue
            lines += ["### " + dialogue.get("title", did), "", "状态：" + ("副本人物引用" if did in self.live_dialogues else "当前未绑定到已放置人物") + "；人物：" + self.npc_name(dialogue.get("npcId", "")) + "。", ""]
            lines += ["<details>", "<summary>展开对话原文</summary>", ""] + self.dialogue_lines(did, current) + ["</details>", ""]
        return "\n".join(lines)

    def collection_page(self):
        current = "collections.md"
        lines = ["# 修行、商店、传承与收藏目录", "", "[返回玩家入口](README.md) · [八城路线](worlds.md)", "",
                 "## 修行怎么形成长期目标", "", "经验决定技能上限，潜能投入技能，老师仅负责免费入门。新角色有500潜能、5000钱币（50两银），经验上限起步20。每10秒投入10潜能，已装备悟真篇可加速；一次可升多级，未满一级保留进度。断线停止修炼，不送离线收益。", "",
                 "首版38门、五档品阶，一门一条等级线。江湖武学向指定老师入门，名家以上从指定副本100%结算取得完整秘本，满足根基和读书要求后参悟；失败不扣书。内功/身法/护体/战斗/学识各装备一门，战斗须匹配当前主武器；暗器也是主武器。老师不再限制后续修为。", "",
                 "恢复药与干粮共享10秒冷却，战斗中可以使用，换药或重连不能清除冷却。完整秘本绑定、不可售，已学或已有书不重复结算。钱币数值以100=1两银折算。", "",
                 "## 全部武学", "", "| 武学 | 品阶与前置 | 风格与100级基线 | 入门来源 | 入门老师与修炼 |", "|---|---|---|---|---|"]
        for sid, skill in self.skills.items():
            req = skill.get("requirements", {})
            prereq = req.get("prerequisite", {})
            category = {1:"基础",2:"江湖",3:"名家",4:"上乘",5:"绝学"}[skill["rarity"]]
            if prereq:
                category += "；" + self.skills[prereq["skill"]]["name"] + " ≥" + str(prereq["level"])
            if req.get("literacy"):
                category += "；读书识文 ≥" + str(req["literacy"])
            item = skill.get("manualId") if not skill.get("teacherId") else None
            source = self.items[item]["name"] + "：" + self.sources(self.item_sources.get(item, []), current) if item else "向老师请教"
            teachers = []
            for npc, definition in self.npcs.items():
                if sid in definition.get("skills", {}) and self.locations.get(npc):
                    teachers.append(self.npc_name(npc) + "（入门）：" + self.location(npc, current))
            style = safe(skill.get("description", "")) + "<br>" + safe(self.skill_bonuses(skill.get("bonuses", {})))
            lines.append("| " + safe(skill["name"]) + "<br>`" + sid + "` | " + safe(category) + " | " + style + " | " + source + " | " + ("；".join(teachers) or "参悟后自主修炼，无老师上限") + " |")
        lines += ["", "## 炼药、熔铸与材料交换", "", "带齐材料与银两，与对应 NPC 反复交谈，按当前符合条件的对话顺序触发。当前普通交谈会记住已读选项：同一角色已做过的配方不会再次自动触发；后端另有选项接口，但网页尚未提供通用重复选择菜单。", "", "| 操作 | 人物与位置 | 实际消耗 | 获得 |", "|---|---|---|---|"]
        for did, dialogue in self.dialogues.items():
            if did not in self.live_dialogues:
                continue
            for option in dialogue.get("options", []):
                if option.get("takeItems") and option.get("giveItems"):
                    npc = dialogue["npcId"]
                    cost = self.items_text(option["takeItems"]) + (" + " + str(option["takeMoney"]) + " 两" if option.get("takeMoney") else "")
                    lines.append("| " + safe(option["text"]) + " | " + self.npc_name(npc) + "：" + self.location(npc, current) + " | " + safe(cost) + " | " + safe(self.items_text(option["giveItems"])) + " |")
        lines += ["", "## 商店与价格", "", "购买按 value 原价，出售通常按 value 的一半向下取整，任务物品不可卖。烟尘粉虽有逃命用途的描述，目前没有配置可用效果，不能当成已经实现的脱战药。", ""]
        for npc, definition in self.npcs.items():
            if not definition.get("shopItems") or not self.locations.get(npc):
                continue
            lines += [f'<a id="shop-{npc}"></a>', "", "### " + self.npc_name(npc), "", self.location(npc, current), "", "| 商品 | 单价（钱币） | 用途 |", "|---|---:|---|"]
            for item in definition["shopItems"]:
                value = self.items[item]
                lines.append("| " + safe(value["name"]) + " | " + str(value.get("value", 0)) + " | " + safe(value.get("description", "")) + " |")
            lines.append("")
        lines += ["## 全部物品与实际获取入口", "", "同名装备可能有历史别名；武学已归并为一门一名。7种江湖秘本仅供旧档迁移保留，新角色直接向指定老师入门。没有可达入口的条目不能作为当前收集目标。", "", "| 物品 / ID | 类型与用途 | 当前来源 |", "|---|---|---|"]
        for iid, item in self.items.items():
            use = SLOTS.get(item.get("equipmentSlot"), item.get("type", "物品"))
            effects = item.get("effects", {})
            if effects.get("learnSkill"):
                use += "；使用参悟「" + self.skills[effects["learnSkill"]]["name"] + "」"
            elif effects.get("recoverHp"):
                use += "；回复气血 " + str(effects["recoverHp"])
            elif item.get("usable"):
                use += "；当前使用逻辑未支持其效果"
            sources = self.sources(self.item_sources.get(iid, []), current)
            if not self.item_sources.get(iid) and item.get("source"):
                sources += "；资源备注「" + safe(item["source"]) + "」尚不足以证明入口已接通"
            flat = [ATTRIBUTES.get(k, k) + ("+" if isinstance(v, (int, float)) and v >= 0 else "") + str(v) for k, v in effects.items() if k not in ("learnSkill", "recoverHp")]
            use += ("；" + "、".join(flat)) if flat else ""
            lines.append("| " + safe(item["name"]) + "<br>`" + iid + "` | " + safe(use) + "<br>" + safe(item.get("description", "")) + " | " + sources + " |")
        lines += ["", "## 全部称号", "", "称号由人物旗标自动授予，通关型称号要在 100% 完成后离开结算才会记录。", "", "| 称号 | 条件来源 | 旗标 |", "|---|---|---|"]
        for flag, title in self.titles.items():
            lines.append("| " + safe(title) + " | " + self.sources(self.flags.get(flag, []), current) + " | `" + flag + "` |")
        lines += ["", "## 装备强化的当前入口", "", "后端已有 `enhance_item`，但当前网页源码没有强化按钮，聊天框也不解析游戏命令，因此普通网页游玩暂时无法直接发起强化。不要把「在聊天框输入 enhance_item」当作操作方法；后续接好 UI 后再把它加入常规成长路线。", ""]
        return "\n".join(lines)

    def status_page(self):
        current = "content_status.md"
        lines = ["# 当前内容状态与容易踩空的入口", "", "[返回玩家入口](README.md)", "", "本页按当前资源引用与运行时代码检查，记录玩家能遇到的缺口；没有修改游戏行为，也没有把未接内容补写成已发生的剧情。", "",
                 "## 覆盖统计", "", "| 内容 | 配置条目 | 手册覆盖方式 |", "|---|---:|---|",
                 f"| 世界 / 公共房间 | {len(self.worlds)} / {sum(len(w['rooms']) for w in self.worlds)} | 出生点路线、NPC、生活动作和实际交谈 |",
                 f"| 副本 | {len(self.dungeons)} | 七卷全量逐房攻略、机关分支与剧情 |",
                 f"| 任务 | {len(self.quests)} | {len(self.wired_quests)} 条已接通，{len(self.quests)-len(self.wired_quests)} 条缺少接取/交付 |",
                 f"| 技能 / 物品 / 称号 | {len(self.skills)} / {len(self.items)} / {len(self.titles)} | 老师、前置、获取入口与条件 |",
                 f"| 普通对话档案 | {len(self.dialogues)} | 常驻人物或附录逐份保留原文；未绑定的明确标记 |", "",
                 "## 解锁与叙事差异", "", "- 当前序27空缺；序28《蜀道栈桥》仍需序26完成，服务端按最近存在的前序组检查，不会提前开放。", "- 同一全局序号的不同副本是并行替代入口：任一满通关结算可解下一序号，卷内 chapter 不是解锁条件。", "- 序号 42《幽冥鬼市》是三路地图，当前没有「选一锁死」机制；序号 62 的同名关是另一个四房战斗模板。", "- 序号 43《天罡北斗阵》有七步机关，但未实现文案所称的错误重置；序号 61 的同名关只有通用开门流程。", "- 《护送商队》用进入终点计目标，没有独立跟随 AI 和护送失败血条。", "- 后期部分副本的剧情只有「要点已明」「通道开启」等短句。攻略保留这些实际文字，不据副本名称编造一段完整原著故事。", "",
                 "## 未绑定的副本动作", "", "这些动作有定义，但没有被房间 actions、可用对话或分支入口引用。一般是未接入的「听取任务」，主战斗路线仍可看各关攻略。", "", "| 副本 | 当前不能触发的动作 |", "|---|---|"]
        for dungeon, aid in self.unbound:
            lines.append("| " + self.dungeon_link(dungeon, current) + " | " + safe(dungeon["puzzleActions"][aid].get("label", aid)) + "（`" + aid + "`） |")
        lines += ["", "## 其他实际入口差异", "", "- 世界任务只有四条接好了 startQuestId / completeQuestId；NPC 的 questIds 或一句「有赏」不会自动建立任务状态。其余 37 条见任务页。", "- 后十条任务的目标如 `dungeon_cleared_datangdoubledragonxb2` 使用不存在的副本 ID，即使补上接交对话，还需要修正目标。", "- 巴蜀药田与洛阳普通采集默认产止血草；扬州芦根单独保留。", "- 世界 JSON 的补充 rumors 尚未由 listen_rumor 读取；新城某些听传闻按钮目前只回泛用台词。", "- 强化已有后端接口，没有网页按钮。底部输入框是公开聊天，不能执行后端指令。", "- 没有查到可达来源的物品或老师，在收藏表中单列；资源总量不直接代表可收集总量。", "",
                 "- 普通交谈会记录已读选项，炼药、熔铸等配方用此入口各自动触发一次；不能据此承诺网页支持无限重复炼药。", "- 烟尘粉只有用途描述，没有接入可用物品效果；当前不能用它脱战。", "",
                 "## 维护与校对", "", "玩家入口和推荐路线是手写内容；本页、七卷故事、世界、任务与收藏事实页由 `python scripts/generate_player_guide.py` 从资源生成，`--check` 检查漂移和本手册的内部链接。行为规则已对照 manager 的副本解锁、交谈/任务、房间动作、学习/使用物品代码以及前端按钮核对；没有逐关实机通关或验证数值难度。", ""]
        return "\n".join(lines)

    def pages(self):
        result = self.story_pages()
        result.update({"worlds.md": self.world_page(), "quests.md": self.quest_page(), "collections.md": self.collection_page(), "content_status.md": self.status_page()})
        current = "unlock_order.md"
        index = ["# 全部副本的全局解锁顺序", "", "[返回玩家入口](README.md)", "", "同序号的关卡任一 100% 离开结算即可解锁下一序号；它们不是必须全部完成的连续章节。卷内章号不参与判断。当前序号27空缺，28仍检查最近存在的序26。", "", "| 全局序号 | 副本与卷 | 实际前置 |", "|---:|---|---|"]
        for dungeon in self.dungeons:
            index.append("| " + str(dungeon["order"]) + " | " + self.dungeon_link(dungeon, current) + " · " + dungeon["chainName"] + " | " + self.unlock(dungeon, current) + " |")
        result[current] = "\n".join(index)
        hashes = hashlib.sha256()
        for path in sorted(RESOURCES.rglob("*.json")):
            hashes.update(path.relative_to(RESOURCES).as_posix().encode())
            hashes.update(json.dumps(read(path), ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode("utf-8"))
        marker = "<!-- 资源校验指纹：" + hashes.hexdigest() + "；由 scripts/generate_player_guide.py 生成。 -->\n\n"
        return {path: marker + text.split("\n",1)[0] + "\n\n> 本手册对应武学V2源码；服务器需配套部署与旧档迁移后生效。\n" + text.split("\n",1)[1].rstrip() + "\n" for path, text in result.items()}


def validate_links(pages):
    pages = dict(pages)
    pages["README.md"] = (BOOK / "README.md").read_text(encoding="utf-8")
    for current, text in pages.items():
        for link in re.findall(r"\]\(([^)]+)\)", text):
            if "://" in link:
                continue
            file, _, anchor = link.partition("#")
            target = ((BOOK / current).parent / file).resolve() if file else (BOOK / current).resolve()
            if not target.is_relative_to(BOOK.resolve()):
                if not target.is_file():
                    raise ValueError("missing reference: " + current + " -> " + link)
                continue
            relative = target.relative_to(BOOK.resolve()).as_posix()
            if relative not in pages:
                raise ValueError("missing player page: " + current + " -> " + link)
            if anchor and 'id="' + anchor + '"' not in pages[relative]:
                raise ValueError("missing player anchor: " + current + " -> " + link)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    guide = Guide()
    pages = guide.pages()
    validate_links(pages)
    for name, text in pages.items():
        path = BOOK / name
        if args.check:
            if not path.is_file() or path.read_text(encoding="utf-8") != text:
                raise SystemExit("Outdated guide: " + str(path.relative_to(ROOT)))
        else:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text, encoding="utf-8", newline="\n")
    anchors = "\n".join(pages.values())
    for key in guide.by_dungeon:
        assert 'id="' + key + '"' in anchors
    print(f"{'Checked' if args.check else 'Generated'} {len(pages)} pages; {len(guide.dungeons)} dungeons, {len(guide.quests)} quests ({len(guide.wired_quests)} wired), {len(guide.worlds)} worlds; links valid.")


if __name__ == "__main__":
    main()
