import { computed, ref } from 'vue'

// 副本相关的视图状态：副本列表/选中项、当前是否处于副本、
// 离开副本的确认提示（含防伪造 token）与房间同步。
export function useDungeonView({ room, addLog, addDungeonLeaveButtonLog, sendGameCommand, checkCanAct, getPlayerId, getPlayerState = () => 'IDLE', refreshAfterQuickComplete = () => {} }) {
  const dungeonsList = ref([])
  const selectedDungeonId = ref(null)
  const pendingDungeonLeave = ref(null)
  const quickCompleting = ref(false)
  let quickTimer = null
  const finishQuickComplete = () => {
    clearTimeout(quickTimer)
    quickTimer = null
    quickCompleting.value = false
    refreshAfterQuickComplete()
  }
  const disposeDungeonView = () => { clearTimeout(quickTimer); quickCompleting.value = false }

  const sortedDungeonsList = computed(() => {
    if (!dungeonsList.value) return []
    return [...dungeonsList.value].sort((a, b) => {
      const orderA = Number(a.order) || 999
      const orderB = Number(b.order) || 999
      if (orderA !== orderB) return orderA - orderB
      return String(a.id || '').localeCompare(String(b.id || ''))
    })
  })

  const selectedDungeon = computed(() => {
    if (!selectedDungeonId.value || !dungeonsList.value) return null
    return dungeonsList.value.find(d => d.id === selectedDungeonId.value) || null
  })

  const isInDungeon = computed(() => {
    const worldId = room.value?.worldId
    return typeof worldId === 'string' && worldId.startsWith('dungeon_')
  })

  const canQuickCompleteSelected = computed(() => !!(selectedDungeon.value?.cleared ?? selectedDungeon.value?.canQuickComplete) && getPlayerState() === 'IDLE' && !isInDungeon.value && !quickCompleting.value)
  const quickCompleteStatusHint = computed(() => isInDungeon.value ? '请先返回公共世界' : (getPlayerState() !== 'IDLE' ? '请先结束当前行为，恢复空闲后快速完成' : ''))

  const buildDungeonLeavePrompt = () => {
    if (!isInDungeon.value || !room.value) return null
    const dungeonName = room.value?.dungeon?.name || room.value?.worldName || selectedDungeon.value?.name || '未知'
    const progress = Math.max(0, Math.min(100, Number(room.value?.dungeon?.progress ?? 0) || 0))
    return { name: dungeonName, progress, token: `${Date.now()}_${Math.random().toString(36).slice(2, 8)}` }
  }

  const syncPendingDungeonLeavePrompt = () => {
    if (!pendingDungeonLeave.value) return
    const currentToken = pendingDungeonLeave.value.token
    const next = buildDungeonLeavePrompt()
    if (!next) {
      pendingDungeonLeave.value = null
      return
    }
    pendingDungeonLeave.value = { ...next, token: currentToken }
  }

  const syncDungeonSelectionFromRoom = () => {
    const currentDungeonId = room.value?.dungeon?.id
    if (currentDungeonId) {
      selectedDungeonId.value = currentDungeonId
    }
  }

  const applyRoomDungeonUpdate = (dungeon) => {
    if (!room.value || !dungeon) return
    room.value.dungeon = { ...(room.value.dungeon || {}), ...dungeon }
    syncPendingDungeonLeavePrompt()
  }

  const selectDungeon = (dungeonId) => {
    selectedDungeonId.value = selectedDungeonId.value === dungeonId ? null : dungeonId
  }

  const enterDungeon = () => {
    if (!checkCanAct('enter_dungeon')) return
    if (!selectedDungeonId.value) return
    sendGameCommand("command", "enter_dungeon", getPlayerId(), { dungeonId: selectedDungeonId.value })
  }

  const quickCompleteDungeon = () => {
    if (!canQuickCompleteSelected.value) return
    if (!checkCanAct('quick_complete_dungeon')) return
    quickCompleting.value = true
    quickTimer = setTimeout(() => {
      addLog('快速完成响应尚未收到，已刷新目录；请核对收益后重试。')
      finishQuickComplete()
    }, 15000)
    sendGameCommand('command', 'quick_complete_dungeon', getPlayerId(), { dungeonId: selectedDungeonId.value, expectedSequence: selectedDungeon.value.quickCompleteSequence })
  }

  const leaveCurrentDungeon = () => {
    if (!checkCanAct('leave_dungeon')) return
    const prompt = buildDungeonLeavePrompt()
    if (!prompt) {
      pendingDungeonLeave.value = null
      return
    }
    pendingDungeonLeave.value = prompt
    addLog(`确认离开${prompt.name}副本？`)
    addLog(`当前副本完成度${prompt.progress}%`)
    addDungeonLeaveButtonLog(prompt.token)
  }

  const confirmLeaveCurrentDungeon = () => {
    if (!pendingDungeonLeave.value) return
    if (!checkCanAct('leave_dungeon')) return
    pendingDungeonLeave.value = null
    sendGameCommand("command", "leave_dungeon", getPlayerId(), {})
  }

  return {
    canQuickCompleteSelected,
    quickCompleteStatusHint,
    quickCompleting,
    quickCompleteDungeon,
    finishQuickComplete,
    disposeDungeonView,
    dungeonsList,
    selectedDungeonId,
    pendingDungeonLeave,
    sortedDungeonsList,
    selectedDungeon,
    isInDungeon,
    buildDungeonLeavePrompt,
    syncPendingDungeonLeavePrompt,
    syncDungeonSelectionFromRoom,
    applyRoomDungeonUpdate,
    selectDungeon,
    enterDungeon,
    leaveCurrentDungeon,
    confirmLeaveCurrentDungeon
  }
}
