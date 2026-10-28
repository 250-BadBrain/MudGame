package gamedata

import (
	"mudgame/server/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestQuickRewardsRejectInvalidOrPaidSources(t *testing.T) {
	for _, payload := range []string{
		`{"id":"bad","quickRewards":{"items":{"missing":1}}}`,
		`{"id":"bad","quickRewards":{"items":{"key":1}},"temporaryItems":["key"],"puzzleActions":{"a":{"giveItems":{"key":1}}}}`,
		`{"id":"bad","quickRewards":{"items":{"key":1}},"puzzleActions":{"a":{"giveItems":{"key":1},"takeItems":{"herb":2}}}}`,
		`{"id":"bad","quickRewards":{"money":500},"puzzleActions":{"a":{"giveMoney":20}}}`,
	} {
		dir := t.TempDir()
		_ = os.Mkdir(filepath.Join(dir, "dungeons"), 0700)
		_ = os.WriteFile(filepath.Join(dir, "dungeons", "bad.json"), []byte(payload), 0600)
		if err := validateQuickRewards(dir, map[string]model.ItemDef{"key": {ID: "key"}}); err == nil {
			t.Fatal("invalid quick source accepted:", payload)
		}
	}
}
