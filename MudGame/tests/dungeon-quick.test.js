import assert from 'node:assert/strict'
import test from 'node:test'
import { ref } from 'vue'
import { useDungeonView } from '../src/composables/useDungeonView.js'

test('quick completion sends one intent with server sequence and refreshes after reply', () => {
  const sent = [], refreshed = [], logs = []
  const d = useDungeonView({ room: ref({ worldId: 'luoyang' }), addLog: x => logs.push(x), addDungeonLeaveButtonLog: () => {},
    sendGameCommand: (...args) => sent.push(args), checkCanAct: () => true, getPlayerId: () => 'player',
    getPlayerState: () => 'IDLE', refreshAfterQuickComplete: () => refreshed.push(true) })
  d.dungeonsList.value = [{ id: 'temple', cleared: true, canQuickComplete: true, quickCompleteSequence: 7 }]
  d.selectedDungeonId.value = 'temple'
  d.quickCompleteDungeon(); d.quickCompleteDungeon()
  assert.equal(sent.length, 1)
  assert.deepEqual(sent[0], ['command', 'quick_complete_dungeon', 'player', { dungeonId: 'temple', expectedSequence: 7 }])
  assert.equal(d.quickCompleting.value, true)
  d.finishQuickComplete()
  assert.equal(d.quickCompleting.value, false)
  assert.equal(refreshed.length, 1)
})

test('uncleared, busy and dungeon characters cannot quick complete from the UI', () => {
  const sent = [], state = ref('IDLE'), room = ref({ worldId: 'luoyang' })
  const d = useDungeonView({ room, addLog: () => {}, addDungeonLeaveButtonLog: () => {}, sendGameCommand: (...args) => sent.push(args),
    checkCanAct: () => true, getPlayerId: () => 'player', getPlayerState: () => state.value, refreshAfterQuickComplete: () => {} })
  d.dungeonsList.value = [{ id: 'temple', canQuickComplete: false, quickCompleteSequence: 0 }]; d.selectedDungeonId.value = 'temple'
  d.quickCompleteDungeon()
  d.dungeonsList.value[0].canQuickComplete = true; state.value = 'COMBAT'; d.quickCompleteDungeon()
  state.value = 'IDLE'; room.value.worldId = 'dungeon_temple'; d.quickCompleteDungeon()
  assert.equal(sent.length, 0)
})
