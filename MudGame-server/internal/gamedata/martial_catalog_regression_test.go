package gamedata

import "testing"

func TestApprovedMartialTransmissionCatalog(t *testing.T) {
	data, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]struct {
		grade           int
		teacher, source string
	}{
		"basic_unarmed":       {1, "", ""},
		"basic_parry":         {1, "", ""},
		"basic_neigong":       {1, "", ""},
		"basic_qinggong":      {1, "", ""},
		"basic_blade":         {1, "", ""},
		"basic_sword":         {1, "", ""},
		"basic_literacy":      {1, "", ""},
		"csj_changshengjue":   {5, "", "datang_xinggong"},
		"jyg_zhanshentu_dao":  {5, "", "posui_zhongjuan"},
		"tiandao_blade":       {5, "", "tiandao_trial_hall"},
		"basic_staff":         {1, "", ""},
		"basic_hidden_weapon": {1, "", ""},
		"thunder_sword":       {2, "military_instructor", ""},
		"bloody_blade":        {2, "military_instructor", ""},
		"basic_spear":         {1, "", ""},
		"basic_whip":          {1, "", ""},
		"basic_mace":          {1, "", ""},
		"basic_hook":          {1, "", ""},
		"cihang_jiandian":     {5, "", "changan_trial_hall"},
		"hunyuan_neigong":     {2, "academy_teacher", ""},
		"tianmo_ce":           {5, "", "fuyu_mishi"},
		"talang_jue":          {2, "weapon_school_master", ""},
		"fuhu_jin":            {2, "weapon_school_master", ""},
		"tiebi_gong":          {2, "salt_hall_master", ""},
		"wuzhen_pian":         {2, "academy_teacher", ""},
		"xiulou_daofa":        {3, "", "bianhuang_wangting"},
		"taiyi_jianfa":        {3, "", "datang_jingling"},
		"dongting_xinf":       {3, "", "datang_baling"},
		"lingxu_bu":           {3, "", "datang_sanhua"},
		"poxu_zhi":            {4, "", "posui_houjuan"},
		"bawang_qiang":        {3, "", "datang_liangdao"},
		"youlong_bian":        {3, "", "fuyu_fangfuchou"},
		"lieshi_jian":         {3, "", "datang_wagang"},
		"suolang_gou":         {3, "", "fuyu_shuizhan"},
		"mantian_huayu":       {4, "", "bashu_tangmen_trial"},
		"fengmo_gun":          {3, "", "datang_baling"},
		"taxue_wuhen":         {4, "", "jiangnan_jianchi"},
		"jingang_buhuai":      {4, "", "jingling_trial_hall"},
	}
	for id, want := range expected {
		got := data.Skills[id]
		if got.Rarity != want.grade || got.TeacherID != want.teacher || got.SourceDungeon != want.source {
			t.Errorf("approved transmission changed for %s: grade=%d teacher=%s source=%s", id, got.Rarity, got.TeacherID, got.SourceDungeon)
		}
	}
}
