import { computed, ref } from 'vue'

// 背包/装备面板的选择、查看、丢弃、出售、装备与使用流程。
// 装备槽位的中文标签也在此维护。
export function useBackpackActions({ backpackItems, sendGameCommand, getPlayerId, checkCanAct }) {
  const equipmentSlots = {
    "WEAPON": "武器",
    "HEAD": "头部",
    "BODY": "衣服",
    "FEET": "鞋子",
    "NECKLACE": "项链",
    "RING": "戒指",
    "ACCESSORY": "配饰"
  }

  const equipment = ref({})
  const selectedEquipmentSlot = ref(null)

  const selectedBackpackItem = ref(null)
  const viewingItem = ref(null)
  const discardingItem = ref(null)
  const discardAmount = ref(1)
  const sellingItem = ref(null)
  const enhancingItem = ref(null)
  const sellAmount = ref(1)

  const selectEquipmentSlot = (slot) => {
    if (selectedEquipmentSlot.value === slot) {
      selectedEquipmentSlot.value = null
    } else {
      selectedEquipmentSlot.value = slot
    }
  }

  const equipItem = (item) => {
    if (!checkCanAct('equip')) return
    sendGameCommand("command", "equip", getPlayerId(), { itemId: item.id })
    viewingItem.value = null
    selectedBackpackItem.value = null
  }

  const useBackpackItem = (item) => {
    if (!checkCanAct('use_item')) return
    if (!item?.id) return
    sendGameCommand("command", "use_item", getPlayerId(), { itemId: item.id })
    viewingItem.value = null
    selectedBackpackItem.value = null
  }

  const unequipItem = (slot) => {
    if (!checkCanAct('unequip')) return
    sendGameCommand("command", "unequip", getPlayerId(), { slot: slot })
    selectedEquipmentSlot.value = null
  }

  const selectBackpackItem = (item) => {
    const currentId = selectedBackpackItem.value?.uniqueId || selectedBackpackItem.value?.id
    const targetId = item.uniqueId || item.id

    if (currentId === targetId) {
      selectedBackpackItem.value = null
      viewingItem.value = null
      discardingItem.value = null
      sellingItem.value = null
    } else {
      selectedBackpackItem.value = item
      viewingItem.value = null
      discardingItem.value = null
      sellingItem.value = null
    }
  }

  const viewBackpackItem = (item) => {
    const currentId = viewingItem.value?.uniqueId || viewingItem.value?.id
    const targetId = item.uniqueId || item.id
    viewingItem.value = currentId === targetId ? null : item
    discardingItem.value = null
    sellingItem.value = null
  }

  const initiateDiscard = (item) => {
    discardingItem.value = item
    viewingItem.value = null
    sellingItem.value = null
    discardAmount.value = 1
  }

  const initiateSell = (item) => {
    sellingItem.value = item
    viewingItem.value = null
    discardingItem.value = null
    // Default to max amount for selling
    sellAmount.value = item.count || 1
  }

  const changeDiscardAmount = (delta) => {
    let newAmount = discardAmount.value + delta
    if (newAmount < 1) newAmount = 1

    // Count items in backpack
    // Since we now group items, item.count should be available
    const count = discardingItem.value.count || backpackItems.value.filter(i => i.id === discardingItem.value.id).length
    if (newAmount > count) newAmount = count

    discardAmount.value = newAmount
  }

  const confirmDiscard = () => {
    if (!checkCanAct('discard')) return
    if (!discardingItem.value) return
    sendGameCommand("command", "discard_item", getPlayerId(), {
      itemId: discardingItem.value.id,
      amount: discardAmount.value
    })
    discardingItem.value = null
    selectedBackpackItem.value = null
  }

  const changeSellAmount = (delta) => {
    let newAmount = sellAmount.value
    const max = sellingItem.value.count || 1

    if (delta === 'min') {
      newAmount = 1
    } else if (delta === 'max') {
      newAmount = max
    } else {
      newAmount += delta
    }

    if (newAmount < 1) newAmount = 1
    if (newAmount > max) newAmount = max

    sellAmount.value = newAmount
  }

  const maxSellCount = computed(() => {
    if (!sellingItem.value) return 1
    return sellingItem.value.count || 1
  })

  const maxDiscardCount = computed(() => {
    if (!discardingItem.value) return 1
    return discardingItem.value.count || backpackItems.value.filter(i => i.id === discardingItem.value.id).length || 1
  })

  const discardShowAmount = computed(() => {
    if (!discardingItem.value) return false
    return backpackItems.value.filter(i => i.id === discardingItem.value.id).length > 1 || (discardingItem.value.stackable && discardingItem.value.count > 1)
  })

  const confirmSell = () => {
    if (!checkCanAct('sell')) return
    if (!sellingItem.value) return
    sendGameCommand("command", "sell", getPlayerId(), {
      itemId: sellingItem.value.id,
      amount: sellAmount.value
    })
    sellingItem.value = null
    selectedBackpackItem.value = null
  }

  const cancelSell = () => {
    sellingItem.value = null
  }

  const cancelDiscard = () => {
    discardingItem.value = null
  }

  const initiateEnhance = (item) => {
    if (!checkCanAct('enhance_item') || !item?.equipmentSlot || item.enhancement?.atMax) return
    enhancingItem.value = item
  }
  const cancelEnhance = () => { enhancingItem.value = null }
  const confirmEnhance = () => {
    const item = enhancingItem.value
    if (!item || !checkCanAct('enhance_item')) return
    const info = item.enhancement
    if (!info || info.atMax || info.ownedMoney < info.nextMoney || info.ownedBronze < info.nextBronze) return
    enhancingItem.value = null
    sendGameCommand('command', 'enhance_item', getPlayerId(), { itemId: item.id })
  }

  // 关闭面板/切换选择时的统一清理
  const resetSelection = () => {
    enhancingItem.value = null
    viewingItem.value = null
    discardingItem.value = null
    sellingItem.value = null
    selectedBackpackItem.value = null
  }

  return {
    enhancingItem, initiateEnhance, confirmEnhance, cancelEnhance,
    equipmentSlots,
    equipment,
    selectedEquipmentSlot,
    selectedBackpackItem,
    viewingItem,
    discardingItem,
    discardAmount,
    sellingItem,
    sellAmount,
    selectEquipmentSlot,
    equipItem,
    useBackpackItem,
    unequipItem,
    selectBackpackItem,
    viewBackpackItem,
    initiateDiscard,
    initiateSell,
    changeDiscardAmount,
    confirmDiscard,
    changeSellAmount,
    maxSellCount,
    maxDiscardCount,
    discardShowAmount,
    confirmSell,
    cancelSell,
    cancelDiscard,
    resetSelection
  }
}
