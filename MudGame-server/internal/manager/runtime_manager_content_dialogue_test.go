package manager

import (
	"mudgame/server/internal/model"
	"testing"
)

func TestContentDialogueConditionsAreConjunctive(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	if r.dialogueRequirementMetNoLock(p, model.DialogueRequire{QuestID: "escort_rumor_run", QuestStatus: "not_started", Flag: "missing"}) {
		t.Fatal("missing quest bypassed flag requirement")
	}
}

func TestContentDialogueChoiceAndTalkShareOneTimeHistory(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.RoomID = "ly_herbal_hall"
	_ = r.savePlayers(players)
	original := fileDialogues["dlg_herbalist_sun"]
	t.Cleanup(func() { fileDialogues["dlg_herbalist_sun"] = original })
	fileDialogues["dlg_herbalist_sun"] = model.DialogueDef{ID: original.ID, Opening: "赠礼", Options: []model.DialogueOption{{ID: "gift", Text: "领赠礼", Response: "收下吧", GiveMoney: 100}}}
	_, _, _, _, err := r.ChooseDialogue(id, "herbalist_sun", "gift")
	if err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	money := players[id].Money
	if _, _, _, _, err = r.ChooseDialogue(id, "herbalist_sun", "gift"); err == nil {
		t.Fatal("gift replay accepted")
	}
	for i := 0; i < 3; i++ {
		_, _ = r.Talk(id, "herbalist_sun")
	}
	players, _ = r.loadPlayers()
	if players[id].Money != money {
		t.Fatal("Talk reissued chosen gift")
	}
}

func TestContentDialogueFailedEffectsAreAtomic(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.QuestStates = map[string]model.QuestState{"escort_rumor_run": {ID: "escort_rumor_run", Status: "active"}}
	p.Backpack = map[string]int{"herb": 3}
	before := p.Money
	opt := model.DialogueOption{ID: "bad_turn_in", TakeMoney: 100, TakeItems: map[string]int{"herb": 2}, GiveItems: map[string]int{"dry_ration": 2}, CompleteQuestID: "escort_rumor_run"}
	if _, err := r.applyDialogueOptionEffectsNoLock(p, &opt, "escort_bureau_chief"); err == nil {
		t.Fatal("unfinished quest accepted")
	}
	if p.Money != before || p.Backpack["herb"] != 3 || p.Backpack["dry_ration"] != 0 {
		t.Fatal("failure changed costs or rewards")
	}
	opt.CompleteQuestID = ""
	opt.TakeItems = map[string]int{"herb": 4}
	if _, err := r.applyDialogueOptionEffectsNoLock(p, &opt, "herbalist_sun"); err == nil {
		t.Fatal("unowned ingredients accepted")
	}
	if p.Money != before || p.Backpack["herb"] != 3 {
		t.Fatal("missing material failure charged")
	}
}

func TestContentTalkDoesNotAutomaticallySpendIngredients(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.RoomID = "ly_herbal_hall"
	p.Backpack = map[string]int{"herb": 14, "dry_ration": 2}
	_ = r.savePlayers(players)
	for i := 0; i < 3; i++ {
		if _, err := r.Talk(id, "herbalist_sun"); err != nil {
			t.Fatal(err)
		}
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Backpack["herb"] != 14 || p.Backpack["dry_ration"] != 2 || p.Backpack["simple_medicine"] != 0 {
		t.Fatal("passive talk spent ingredients")
	}
	for i := 0; i < 2; i++ {
		if _, _, _, _, err := r.ChooseDialogue(id, "herbalist_sun", "brew_with_herbs"); err != nil {
			t.Fatal(err)
		}
	}
	players, _ = r.loadPlayers()
	if players[id].Backpack["simple_medicine"] != 2 || players[id].Backpack["herb"] != 0 {
		t.Fatal("repeatable recipe did not charge each time")
	}
}
