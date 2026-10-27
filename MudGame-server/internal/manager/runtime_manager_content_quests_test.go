package manager

import (
	"mudgame/server/internal/model"
	"strings"
	"testing"
)

func publicNPCRoomForTest(t *testing.T, npcID string) string {
	t.Helper()
	for rid, room := range fileRooms {
		for _, id := range room.NPCIDs {
			if id == npcID {
				return rid
			}
		}
	}
	t.Fatalf("no public location for %s", npcID)
	return ""
}

func TestContentAllUniqueQuestsCanStartFinishAndClaimOnlyOnce(t *testing.T) {
	r, id := v2Player(t)
	count := 0
	for qid, q := range fileQuests {
		if q.AliasOf != "" {
			continue
		}
		count++
		t.Run(qid, func(t *testing.T) {
			players, _ := r.loadPlayers()
			p := players[id]
			p.State = "IDLE"
			p.Money = 5000
			p.Flags = map[string]bool{}
			p.QuestStates = map[string]model.QuestState{}
			p.DialogueHistory = map[string]bool{}
			for _, tpl := range r.dungeonTemplatesNoLock() {
				p.Flags["dungeon_cleared_"+tpl.ID] = true
			}
			for _, o := range q.Objectives {
				delete(p.Flags, o.TargetID)
			}
			p.RoomID = publicNPCRoomForTest(t, q.GiverNPCID)
			_ = r.savePlayers(players)
			start := ""
			for _, opt := range fileDialogues[fileNpcs[q.GiverNPCID].DialogueID].Options {
				if opt.StartQuestID == qid {
					start = opt.ID
				}
			}
			if start == "" {
				t.Fatal("missing start")
			}
			if _, _, _, _, err := r.ChooseDialogue(id, q.GiverNPCID, start); err != nil {
				t.Fatal(err)
			}
			players, _ = r.loadPlayers()
			p = players[id]
			if p.QuestStates[qid].Status != "active" {
				t.Fatal("not active")
			}
			p.RoomID = publicNPCRoomForTest(t, q.TurnInNPCID)
			_ = r.savePlayers(players)
			complete := ""
			for _, opt := range fileDialogues[fileNpcs[q.TurnInNPCID].DialogueID].Options {
				if opt.CompleteQuestID == qid {
					complete = opt.ID
				}
			}
			if complete == "" {
				t.Fatal("missing turn in")
			}
			if _, _, _, _, err := r.ChooseDialogue(id, q.TurnInNPCID, complete); err == nil {
				t.Fatal("unfinished accepted")
			}
			players, _ = r.loadPlayers()
			p = players[id]
			for _, o := range q.Objectives {
				p.Flags[o.TargetID] = true
			}
			_ = r.savePlayers(players)
			if _, _, _, _, err := r.ChooseDialogue(id, q.TurnInNPCID, complete); err != nil {
				t.Fatal(err)
			}
			players, _ = r.loadPlayers()
			p = players[id]
			if p.QuestStates[qid].Status != "completed" || p.Money != 5000+q.Rewards.Money {
				t.Fatal("bad completion")
			}
			if _, _, _, _, err := r.ChooseDialogue(id, q.TurnInNPCID, complete); err == nil {
				t.Fatal("duplicate reward accepted")
			}
		})
	}
	if count != 38 {
		t.Fatalf("expected current 38 unique quests, got %d", count)
	}
}

func TestContentQuestAliasesRetainCompletionAndDirectoryIsHonest(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.QuestStates = map[string]model.QuestState{"xinggong_after": {Status: "completed"}}
	_ = r.savePlayers(players)
	rows, err := r.GetQuests(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 38 {
		t.Fatalf("directory size %d", len(rows))
	}
	for _, row := range rows {
		if row["id"] == "xinggong_after" {
			t.Fatal("alias duplicated")
		}
		if row["id"] == "xinggong_aftermath" && row["status"] != "completed" {
			t.Fatal("legacy completion lost")
		}
		if row["id"] == "quest_r204_0" && !row["locked"].(bool) {
			t.Fatal("late quest offered to new player")
		}
	}
	if log := r.startQuestNoLock(p, "xinggong_aftermath"); log != "" {
		t.Fatal("alias completion allowed restart")
	}
}

func TestContentAllBriefingsHaveWorkingRoomEntries(t *testing.T) {
	r, id := v2Player(t)
	count := 0
	for _, tpl := range r.dungeonTemplatesNoLock() {
		for aid, a := range tpl.PuzzleActions {
			if !strings.HasPrefix(aid, "listen_brief") || a.DialogueNPCID != "" {
				continue
			}
			count++
			inst, err := r.dungeonMgr.CreateInstanceNoLock(id, tpl.ID, "ly_00")
			if err != nil {
				t.Fatal(err)
			}
			players, _ := r.loadPlayers()
			p := players[id]
			p.State = "IDLE"
			p.RoomID = inst.SpawnRoomID
			for rid, room := range inst.Rooms {
				for _, action := range room.Actions {
					if action == aid {
						p.RoomID = rid
					}
				}
			}
			_ = r.savePlayers(players)
			result, err := r.DoRoomAction(id, aid)
			if err != nil {
				t.Fatalf("%s/%s: %v", tpl.ID, aid, err)
			}
			if result["actionStatus"] == "blocked" || inst.PuzzleStates[a.PuzzleID] != a.NextState {
				t.Fatalf("briefing did not run: %v", result)
			}
			r.dungeonMgr.DestroyInstanceNoLock(inst.ID)
		}
	}
	if count < 35 {
		t.Fatalf("missed briefings: %d", count)
	}
}

func TestContentRumorsAndSearchUseRoomDefinitions(t *testing.T) {
	r, id := v2Player(t)
	for rid, room := range fileRooms {
		if len(room.Rumors) == 0 && room.SearchFlag == "" {
			continue
		}
		players, _ := r.loadPlayers()
		p := players[id]
		p.RoomID = rid
		p.State = "IDLE"
		p.Flags = map[string]bool{}
		_ = r.savePlayers(players)
		if len(room.Rumors) > 0 {
			result, err := r.DoRoomAction(id, "listen_rumor")
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, line := range room.Rumors {
				if result["message"] == line {
					found = true
				}
			}
			if !found {
				t.Fatal("room rumor ignored")
			}
		}
		if room.SearchFlag != "" {
			if _, err := r.DoRoomAction(id, "search"); err != nil {
				t.Fatal(err)
			}
			players, _ = r.loadPlayers()
			if !players[id].Flags[room.SearchFlag] {
				t.Fatal("search flag missing")
			}
		}
	}
}
