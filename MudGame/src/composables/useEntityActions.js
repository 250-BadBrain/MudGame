import { computed, ref } from 'vue'

// 房间实体（玩家/NPC）的展示模型、选中状态与实体交互指令，
// 以及所有动作前的状态检查（checkCanAct）。
export function useEntityActions({ room, currentPlayer, addLog, sendGameCommand, getPlayerId }) {
  const selectedEntityKey = ref(null)

  const combinedEntities = computed(() => {
    const result = []
    if (room.value?.playersInRoom) {
      for (const p of room.value.playersInRoom) {
        const isCorpse = p.state === 'DEAD'
        result.push({
          id: p.id,
          name: isCorpse ? `${p.name}的尸体` : p.name,
          isPlayer: true,
          isNpc: false,
          isItem: false,
          level: p.level,
          hp: isCorpse ? null : p.attributes?.currentHp,
          maxHp: isCorpse ? null : p.attributes?.maxHp,
          hostile: false,
          state: p.state,
          online: p.online,
          isCorpse,
          isShop: false,
          capabilities: [],
          description: '',
          attributes: p.attributes
        })
      }
    }
    if (room.value?.npcs) {
      for (const n of room.value.npcs) {
        const isCorpse = n.state === 'DEAD'
        result.push({
          id: n.id,
          name: isCorpse ? `${n.name}的尸体` : n.name,
          isPlayer: false,
          isNpc: true,
          isItem: false,
          level: n.level,
          hp: isCorpse ? null : n.attributes?.currentHp,
          maxHp: isCorpse ? null : n.attributes?.maxHp,
          hostile: isCorpse ? false : n.hostile || false,
          combatMode: n.combatMode || 'kill',
          state: n.state,
          isCorpse,
          isShop: n.isShop || false,
          capabilities: n.capabilities || [],
          dialogueId: n.dialogueId || '',
          questIds: n.questIds || [],
          skills: n.skills || {},
          skillNames: n.skillNames || {},
          description: n.description || '',
          attributes: n.attributes
        })
      }
    }
    return result
  })

  const selectedEntityObj = computed(() => {
    if (!selectedEntityKey.value) return null
    return combinedEntities.value.find(e => `entity_${e.id}` === selectedEntityKey.value) || null
  })

  const handleEntitySelect = (entity) => {
    if (!entity) return
    const key = `entity_${entity.id}`
    if (selectedEntityKey.value === key) {
      selectedEntityKey.value = null
    } else {
      selectedEntityKey.value = key
    }
  }

  // 状态名称映射，用于显示友好的状态提示
  const stateNames = {
    'HEALING': '疗伤',
    'MEDITATING': '打坐',
    'COMBAT': '战斗',
    'LEARNING': '修炼',
    'GATHERING': '采集',
    'WORKING': '帮工'
  }

  // 检查当前状态是否允许执行动作
  const checkCanAct = (actionType) => {
    // 停止动作总是允许的
    if (actionType === 'stop' || actionType === 'leave_dungeon') return true

    const state = currentPlayer.value.state
    if (actionType === 'use_item' && state === 'COMBAT') return true

    // 如果是死亡状态，仅允许复活（或其他特定逻辑，这里暂不处理，由后端或getMessage处理）
    if (state === 'DEAD') {
      return true
    }

    // 检查是否在忙碌状态
    const stateName = stateNames[state]
    if (stateName) {
      addLog(`你正在${stateName}，待会再说`)
      return false
    }

    return true
  }

  const doAction = (actionType) => {
    if (!checkCanAct(actionType)) return

    if (actionType === 'relive') {
      sendGameCommand("command", "relive", getPlayerId(), {})
    } else {
      sendGameCommand("command", actionType, getPlayerId(), {})
    }
  }

  const doRoomAction = (actionType) => {
    if (!checkCanAct(actionType)) return
    sendGameCommand("command", "do_action", getPlayerId(), { action: actionType })
  }

  const getHealthStatusDescription = (currentHp, maxHp) => {
    if (!maxHp) return '看起来深不可测。'
    const percent = (currentHp / maxHp) * 100
    if (percent >= 100) return '看起来气血充盈，精力旺盛。'
    if (percent >= 80) return '看起来受了点轻伤，不过并无大碍。'
    if (percent >= 60) return '看起来受伤不轻，动作有些迟缓。'
    if (percent >= 40) return '看起来伤痕累累，状况不太妙。'
    if (percent >= 20) return '看起来摇摇欲坠，随时可能倒下。'
    return '看起来已经奄奄一息，离死不远了。'
  }

  const viewEntity = (entity) => {
    if (entity.state === 'DEAD') {
      const name = entity.isCorpse ? entity.name : entity.name + '的尸体'
      addLog(`${name}\n${name}，静静地躺在这里`)
      return
    }

    let info = `${entity.name}\n`
    if (entity.description) {
      info += `${entity.description}\n`
    }

    if (entity.attributes) {
      const hpDesc = getHealthStatusDescription(entity.attributes.currentHp, entity.attributes.maxHp)
      info += `${entity.name}${hpDesc}`
    } else {
      info += `${entity.name}看起来深不可测。`
    }

    addLog(info)
  }

  const talkEntity = (entity) => {
    if (!checkCanAct('talk')) return
    if (!entity?.isNpc) {
      addLog('只能和 NPC 交谈。')
      return
    }
    sendGameCommand("command", "talk", getPlayerId(), { npcId: entity.id })
  }

  const killEntity = (entity) => {
    if (!checkCanAct('kill')) return
    if (!entity) {
      addLog('找不到目标。')
      return
    }
    sendGameCommand("command", "kill", getPlayerId(), { targetId: entity.id })
  }

  const sparEntity = (entity) => {
    if (!checkCanAct('spar')) return
    if (!entity || entity.state === undefined) {
      addLog('当前版本暂不支持与 NPC 切磋。')
      return
    }
    sendGameCommand("command", "spar", getPlayerId(), { targetId: entity.id })
  }

  return {
    selectedEntityKey,
    combinedEntities,
    selectedEntityObj,
    handleEntitySelect,
    checkCanAct,
    doAction,
    doRoomAction,
    viewEntity,
    talkEntity,
    killEntity,
    sparEntity
  }
}
