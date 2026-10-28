import { computed, ref } from 'vue'

// 信息面板（属性/背包/技能/装备/世界/副本/任务/势力）的开关与切换、
// 各 tab 的数据列表与选中项，以及按指令区分的面板打开意图。
export function useInfoPanels({ room, sendGameCommand, getPlayerId, checkCanAct }) {
  const showInfoPanel = ref(false)
  const currentPanel = ref('') // 'backpack', 'skills', etc.
  const panelTitle = ref('')

  const skillsList = ref([])
  const statusAttributes = ref(null)
  const worldsList = ref([])
  const questsList = ref([])
  const journalEntries = ref([])
  const factionsList = ref([])

  const selectedWorldId = ref(null)
  const selectedSkill = ref(null)
  const viewingSkill = ref(null)
  const skillTab = ref('basic') // 'basic' or 'special'

  // 不同 get_* 的响应可能交错，不能共用一个计数。
  const panelRequests = new Map()
  const sendPanelRequest = (subtype, silent = false) => {
    const pending = panelRequests.get(subtype) || []
    pending.push(silent)
    panelRequests.set(subtype, pending)
    sendGameCommand("command", subtype, getPlayerId(), {})
  }
  const sendSilentRefresh = (subtype) => sendPanelRequest(subtype, true)
  const consumeSilentRefresh = (subtype) => {
    const pending = panelRequests.get(subtype)
    if (!pending?.length) return true
    const silent = pending.shift()
    if (!pending.length) panelRequests.delete(subtype)
    return silent
  }
  const resetPanelRequests = () => {
    // 断开的连接可能丢掉旧刷新响应；保留等待连接恢复的主动查看意图。
    for (const [subtype, pending] of panelRequests) {
      const explicit = pending.filter(silent => !silent)
      if (explicit.length) panelRequests.set(subtype, explicit)
      else panelRequests.delete(subtype)
    }
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
      sendPanelRequest("get_backpack")
    }
  }

  const toggleSkills = () => {
    if (showInfoPanel.value && currentPanel.value === 'skills') {
      closeInfoPanel()
    } else {
      sendPanelRequest("get_skills")
    }
  }

  const toggleEquipment = () => {
    if (showInfoPanel.value && currentPanel.value === 'equipment') {
      closeInfoPanel()
    } else {
      sendPanelRequest("get_equipment")
    }
  }

  const toggleStatus = () => {
    if (showInfoPanel.value && currentPanel.value === 'status') {
      closeInfoPanel()
    } else {
      sendPanelRequest("get_attributes")
    }
  }

  const toggleMaps = () => {
    if (showInfoPanel.value && currentPanel.value === 'maps') {
      closeInfoPanel()
    } else {
      sendPanelRequest("get_worlds")

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
      sendPanelRequest("get_dungeons")
    }
  }

  const toggleQuests = () => {
    if (showInfoPanel.value && currentPanel.value === 'quests') {
      closeInfoPanel()
    } else {
      sendPanelRequest("get_quests")
    }
  }

  const toggleFactions = () => {
    if (showInfoPanel.value && currentPanel.value === 'factions') {
      closeInfoPanel()
    } else {
      sendPanelRequest("get_factions")
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
    journalEntries,
    factionsList,
    selectedWorldId,
    selectedSkill,
    viewingSkill,
    skillTab,
    consumeSilentRefresh,
    resetPanelRequests,
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
