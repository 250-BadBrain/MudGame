package manager

import (
	"mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/model"
	"sort"
	"strings"
)

func journeyNPCLocation(id string) string {
	ids := make([]string, 0, len(fileRooms))
	for rid := range fileRooms {
		ids = append(ids, rid)
	}
	sort.Strings(ids)
	for _, rid := range ids {
		room := fileRooms[rid]
		for _, npc := range room.NPCIDs {
			if npc == id {
				return room.WorldName + "·" + room.Name + "的" + fileNpcs[id].Name
			}
		}
	}
	return fileNpcs[id].Name
}

// Called under the runtime lock. No new progress is written by reading hints.
func (r *RuntimeManager) journeyEntriesNoLock(p *model.Player) []map[string]string {
	out := []map[string]string{}
	if r.dungeonMgr == nil {
		return out
	}
	templates := []runtime.Template{}
	for _, id := range r.dungeonMgr.DungeonTemplateIDsNoLock() {
		tpl, _ := r.dungeonMgr.DungeonTemplateNoLock(id)
		templates = append(templates, tpl)
	}
	for _, d := range fileDialogues {
		j := d.Journey
		if j == nil {
			continue
		}
		known := p.Flags[j.StartFlag]
		missing := []string{}
		for _, npc := range j.Witnesses {
			seen := p.Flags["discovery_"+npc+"_specialty"]
			known = known || seen
			if !seen {
				missing = append(missing, journeyNPCLocation(npc))
			}
		}
		var branch *model.DialogueJourneyBranch
		for i := range j.Branches {
			if p.Flags[j.Branches[i].Flag] {
				branch = &j.Branches[i]
				known = true
				break
			}
		}
		if !known {
			continue
		}
		tpl, exists := r.dungeonMgr.DungeonTemplateNoLock(j.DungeonID)
		if !exists {
			continue
		}
		row := map[string]string{"id": "journey_" + d.NPCID, "group": "journey", "title": j.Title,
			"related": tpl.Name, "progress": "0/4", "state": "线索待核", "text": "将城中已知的证词与现场互证；普通调查路线始终可以独立完成。"}
		switch {
		case branch != nil:
			row["progress"], row["state"] = "2/4", "尚有后续"
			if !p.Flags[branch.FollowFlag] {
				row["next"] = "向" + journeyNPCLocation(branch.NPCID) + "明确选择“追问旧事：" + j.Title + "的后续”。"
				break
			}
			row["progress"], row["state"] = "3/4", "可在现场互证"
			a := tpl.PuzzleActions[branch.ActionID]
			if p.Flags[branch.UsedFlag] {
				row["progress"], row["state"] = "4/4", "已在现场互证"
				row["next"] = "你已将所选旧事带入《" + tpl.Name + "》。后续调查、敌人与终局判断仍按本章推进；不必为此重复刷本。"
			} else if p.Flags["discovery_chapter_"+tpl.ID] || p.Flags["discovery_alternative_"+tpl.ID] || p.Flags["dungeon_cleared_"+tpl.ID] {
				row["progress"], row["state"] = "4/4", "已自行解开现场"
				row["next"] = "你已用自己的方法完成《" + tpl.Name + "》的现场判断，无需重游补进度；本次追问所得的另一解法仍可在重游时尝试。"
			} else if locked, hint := r.dungeonLockStateNoLock(p, tpl, templates); locked {
				row["state"] = "等待章节解锁"
				row["next"] = "情报已经记住。《" + tpl.Name + "》" + hint + "；从“境”查看原有解锁路线，解锁后可在" + tpl.Rooms[a.Rooms[0]].Name + "使用“" + a.Label + "”。"
			} else {
				row["next"] = "在《" + tpl.Name + "》的" + tpl.Rooms[a.Rooms[0]].Name + "可用“" + a.Label + "”。按动作提示先完成现场前置调查，也可沿普通解法继续。"
			}
		case len(missing) > 0:
			row["next"] = "还需向" + strings.Join(missing, "、") + "明确询问专长，再回" + journeyNPCLocation(d.NPCID) + "核对。"
		case heldItemCount(p, j.ItemID) < 1:
			row["progress"], row["state"] = "1/4", "还需原始凭证"
			row["next"] = "两位证人已询问。携带“" + fileItems[j.ItemID].Name + "”到" + journeyNPCLocation(d.NPCID) + "作出选择；仅出示，不消耗。"
		default:
			row["progress"], row["state"] = "1/4", "可以作出选择"
			row["next"] = "证人与凭证齐备。回" + journeyNPCLocation(d.NPCID) + "明确选择自己的处理方式；本事只选一种，尚未选择的结果不会在此提前展示。"
		}
		out = append(out, row)
	}
	return out
}
