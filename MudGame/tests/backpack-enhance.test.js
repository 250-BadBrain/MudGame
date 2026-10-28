import test from 'node:test'
import assert from 'node:assert/strict'
import { ref } from 'vue'
import { useBackpackActions } from '../src/composables/useBackpackActions.js'

const item = () => ({ id: 'iron_sword', equipmentSlot: 'WEAPON', enhancement: { level: 0, atMax: false, ownedMoney: 500, ownedBronze: 2, nextMoney: 200, nextBronze: 1 } })
const setup = () => {
  const calls = []
  const actions = useBackpackActions({ backpackItems: ref([]), getPlayerId: () => 'p', checkCanAct: () => true, sendGameCommand: (...args) => calls.push(args) })
  return { calls, actions }
}

test('enhancement waits for confirmation and a double click sends only one command', () => {
  const { calls, actions } = setup()
  actions.initiateEnhance(item())
  assert.equal(calls.length, 0)
  actions.confirmEnhance(); actions.confirmEnhance()
  assert.equal(calls.length, 1)
  assert.deepEqual(calls[0], ['command', 'enhance_item', 'p', { itemId: 'iron_sword' }])
})

test('cancel and insufficient materials send no enhancement command', () => {
  const { calls, actions } = setup()
  actions.initiateEnhance(item()); actions.cancelEnhance(); actions.confirmEnhance()
  const poor = item(); poor.enhancement.ownedBronze = 0
  actions.initiateEnhance(poor); actions.confirmEnhance()
  assert.equal(calls.length, 0)
})

test('legacy reset requires eligibility and confirmation, cancellation sends nothing', () => {
  const { calls, actions } = setup()
  actions.initiateEnhancementReset(item())
  actions.confirmEnhancementReset()
  assert.equal(calls.length, 0)
  const legacy = item()
  Object.assign(legacy.enhancement, { level: 3, canReset: true, resetMoney: 1200, resetBronze: 6 })
  actions.initiateEnhancementReset(legacy)
  assert.equal(calls.length, 0)
  actions.cancelEnhancementReset(); actions.confirmEnhancementReset()
  assert.equal(calls.length, 0)
  actions.initiateEnhancementReset(legacy)
  actions.confirmEnhancementReset(); actions.confirmEnhancementReset()
  assert.deepEqual(calls, [['command', 'reset_enhancement', 'p', { itemId: 'iron_sword' }]])
})
