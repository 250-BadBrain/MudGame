import { computed, ref } from 'vue'

// 信息面板（属性/背包/技能/装备/世界/副本/任务/势力）的开关与切换、
// 各 tab 的数据列表与选中项，以及"内部刷新不弹面板"的静默计数。
export function useInfoPanels({ room, sendGameCommand, getPlayerId, checkCanAct }) {
  const showInfoPanel = ref(false)
  const currentPanel = ref('') // 'backpack', 'skills', etc.
  const panelTitle = ref('')

  const skillsList = ref([])
  const statusAttributes = ref(null)
  const worldsList = ref([])
  const questsList = ref([])
  const factionsList = ref([])

  const selectedWorldId = ref(null)
  const selectedSkill = ref(null)
  const viewingSkill = ref(null)
  const skillTab = ref('basic') // 'basic' or 'special'

  // 内部数据刷新（交谈/用物品/装备等流程附带刷新）时递增，
  // 对应的 get_* 响应只更新数据，不强制打开或切换信息面板
  const silentRefreshCount = ref(0)
  const sendSilentRefresh = (subtype) => {
    silentRefreshCount.value++
    sendGameCommand("command", subtype, getPlayerId(), {})
  }

  const filteredSkills = computed(() => {
    if (!skillsList.value) return []
    return skillsList.value.filter(skill => {
      if (skillTab.value === 'basic') {
        return skill.id.startsWith('basic_')
      } else {
        return !skill.id.startsWith('basic_')
      }
    })
  })

  const selectSkill = (skill) => {
    if (selectedSkill.value && selectedSkill.value.id === skill.id) {
      selectedSkill.value = null
    } else {
      selectedSkill.value = skill
      viewingSkill.value = null
    }
  }

  const viewSkill = (skill) => {
    viewingSkill.value = viewingSkill.value?.id === skill.id ? null : skill
  }

  const sortedWorldsList = computed(() => {
    if (!worldsList.value) return []
    return [...worldsList.value]
  })

  const selectedWorld = computed(() => {
    if (!selectedWorldId.value || !worldsList.value) return null
    return worldsList.value.find(w => w.id === selectedWorldId.value)
  })

  const selectWorld = (worldId) => {
    selectedWorldId.value = selectedWorldId.value === worldId ? null : worldId
  }

  const enterWorld = () => {
    if (!checkCanAct('enter_world')) return
    if (!selectedWorldId.value) return
    sendGameCommand("command", "enter_world", getPlayerId(), { worldId: selectedWorldId.value })
  }

  const toggleBackpack = () => {
    if (showInfoPanel.value && currentPanel.value === 'backpack') {
      closeInfoPanel()
    } else {
      sendGameCommand("command", "get_backpack", getPlayerId(), {})
    }
  }

  const toggleSkills = () => {
    if (showInfoPanel.value && currentPanel.value === 'skills') {
      closeInfoPanel()
    } else {
      sendGameCommand("command", "get_skills", getPlayerId(), {})
    }
  }

  const toggleEquipment = () => {
    if (showInfoPanel.value && currentPanel.value === 'equipment') {
      closeInfoPanel()
    } else {
      sendGameCommand("command", "get_equipment", getPlayerId(), {})
    }
  }

  const toggleStatus = () => {
    if (showInfoPanel.value && currentPanel.value === 'status') {
      closeInfoPanel()
    } else {
      sendGameCommand("command", "get_attributes", getPlayerId(), {})
    }
  }

  const toggleMaps = () => {
    if (showInfoPanel.value && currentPanel.value === 'maps') {
      closeInfoPanel()
    } else {
      sendGameCommand("command", "get_worlds", getPlayerId(), {})

      // 按房间所属世界选中；副本等非世界房间回退到第一个可用世界
      const currentWorldId = room.value?.worldId
      if (currentWorldId && worldsList.value.some(w => w.id === currentWorldId)) {
        selectedWorldId.value = currentWorldId
      } else {
        selectedWorldId.value = worldsList.value[0]?.id || 'luoyang'
      }
    }
  }

  const toggleDungeons = () => {
    if (showInfoPanel.value && currentPanel.value === 'dungeons') {
      closeInfoPanel()
    } else {
      sendGameCommand("command", "get_dungeons", getPlayerId(), {})
    }
  }

  const toggleQuests = () => {
    if (showInfoPanel.value && currentPanel.value === 'quests') {
      closeInfoPanel()
    } else {
      sendGameCommand("command", "get_quests", getPlayerId(), {})
    }
  }

  const toggleFactions = () => {
    if (showInfoPanel.value && currentPanel.value === 'factions') {
      closeInfoPanel()
    } else {
      sendGameCommand("command", "get_factions", getPlayerId(), {})
    }
  }

  const refreshOpenPanels = () => {
    if (!showInfoPanel.value) return

    const refreshByPanel = {
      backpack: "get_backpack",
      skills: "get_skills",
      equipment: "get_equipment",
      status: "get_attributes",
      maps: "get_worlds",
      dungeons: "get_dungeons",
      quests: "get_quests",
      factions: "get_factions"
    }
    const subtype = refreshByPanel[currentPanel.value]
    if (subtype) {
      sendSilentRefresh(subtype)
    }
  }

  const closeInfoPanel = () => {
    showInfoPanel.value = false
    currentPanel.value = ''
  }

  return {
    showInfoPanel,
    currentPanel,
    panelTitle,
    skillsList,
    statusAttributes,
    worldsList,
    questsList,
    factionsList,
    selectedWorldId,
    selectedSkill,
    viewingSkill,
    skillTab,
    silentRefreshCount,
    sendSilentRefresh,
    filteredSkills,
    selectSkill,
    viewSkill,
    sortedWorldsList,
    selectedWorld,
    selectWorld,
    enterWorld,
    toggleBackpack,
    toggleSkills,
    toggleEquipment,
    toggleStatus,
    toggleMaps,
    toggleDungeons,
    toggleQuests,
    toggleFactions,
    refreshOpenPanels,
    closeInfoPanel
  }
}
