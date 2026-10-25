import { computed, ref } from 'vue'

// 商店与请教技能两个复用 GameShop 组件的面板状态与流程。
export function useShopLearn({ sendGameCommand, getPlayerId, checkCanAct, closeInfoPanel }) {
  // Shop State
  const showShopPanel = ref(false)
  const shopItems = ref([])
  const currentShopNpc = ref(null)
  const selectedShopItem = ref(null)
  const buyAmount = ref(1)

  // Learn State
  const showLearnPanel = ref(false)
  const learnSkills = ref([])
  const currentTeacher = ref(null)
  const selectedLearnSkill = ref(null)

  const isMaxAmountReached = computed(() => {
    if (!selectedShopItem.value) return true
    if (selectedShopItem.value.limit) {
      return buyAmount.value >= selectedShopItem.value.limit
    }
    return buyAmount.value >= 10000
  })

  const changeAmount = (delta) => {
    let newAmount = buyAmount.value

    if (delta === 'min') {
      newAmount = 1
    } else if (delta === 'max') {
      if (selectedShopItem.value.limit) {
        newAmount = selectedShopItem.value.limit
      } else {
        newAmount = 10000
      }
    } else {
      newAmount += delta
    }

    if (newAmount < 1) newAmount = 1

    if (selectedShopItem.value.limit && newAmount > selectedShopItem.value.limit) {
      newAmount = selectedShopItem.value.limit
    } else if (!selectedShopItem.value.limit && newAmount > 10000) {
      newAmount = 10000
    }

    buyAmount.value = newAmount
  }

  const openShop = (npc) => {
    if (!checkCanAct('open_shop')) return
    sendGameCommand("command", "get_shop", getPlayerId(), { npcId: npc.id })
  }

  const closeShop = () => {
    showShopPanel.value = false
    currentShopNpc.value = null
    shopItems.value = []
  }

  const openLearn = (npc) => {
    if (!checkCanAct('learn')) return
    currentTeacher.value = npc
    // npc.skills 是 { skillId: level }，技能名由服务端 room payload 的 skillNames 提供
    learnSkills.value = Object.entries(npc.skills || {}).map(([id, level]) => ({
      id,
      name: npc.skillNames?.[id] || id,
      level
    }))

    showLearnPanel.value = true
    showShopPanel.value = false // Close shop if open
    closeInfoPanel() // Close info panel if open
  }

  const closeLearn = () => {
    showLearnPanel.value = false
    currentTeacher.value = null
    learnSkills.value = []
    selectedLearnSkill.value = null
  }

  const selectLearnSkill = (skill) => {
    selectedLearnSkill.value = skill
  }

  const startLearning = () => {
    if (!checkCanAct('learn')) return
    if (!currentTeacher.value || !selectedLearnSkill.value) return

    sendGameCommand("command", "learn", getPlayerId(), {
      npcId: currentTeacher.value.id,
      skillId: selectedLearnSkill.value.id
    })
    showLearnPanel.value = false
  }

  const selectShopItem = (item) => {
    if (item.limit === 0) return
    selectedShopItem.value = selectedShopItem.value?.id === item.id ? null : item
    buyAmount.value = 1
  }

  const buyItem = () => {
    if (!checkCanAct('buy')) return
    if (!selectedShopItem.value || !currentShopNpc.value) return
    sendGameCommand("command", "buy", getPlayerId(), {
      npcId: currentShopNpc.value.id,
      itemId: selectedShopItem.value.id,
      amount: buyAmount.value
    })
  }

  return {
    showShopPanel,
    shopItems,
    currentShopNpc,
    selectedShopItem,
    buyAmount,
    showLearnPanel,
    learnSkills,
    currentTeacher,
    selectedLearnSkill,
    isMaxAmountReached,
    changeAmount,
    openShop,
    closeShop,
    openLearn,
    closeLearn,
    selectLearnSkill,
    startLearning,
    selectShopItem,
    buyItem
  }
}
