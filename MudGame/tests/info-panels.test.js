import assert from 'node:assert/strict'
import test from 'node:test'
import { ref } from 'vue'
import { useInfoPanels } from '../src/composables/useInfoPanels.js'

const panels = () => useInfoPanels({
  room: ref(null), sendGameCommand: () => {}, getPlayerId: () => 'test', checkCanAct: () => true,
})

test('background attribute replies do not consume a requested skills panel', () => {
  const p = panels()
  p.sendSilentRefresh('get_attributes')
  p.toggleSkills()
  assert.equal(p.consumeSilentRefresh('get_skills'), false)
  assert.equal(p.consumeSilentRefresh('get_attributes'), true)
})

test('same-command requests preserve background and explicit intent in order', () => {
  const p = panels()
  p.sendSilentRefresh('get_skills')
  p.toggleSkills()
  assert.equal(p.consumeSilentRefresh('get_skills'), true)
  assert.equal(p.consumeSilentRefresh('get_skills'), false)
})

test('reconnect drops missing replies and unsolicited refreshes remain silent', () => {
  const p = panels()
  p.sendSilentRefresh('get_skills')
  p.resetPanelRequests()
  p.toggleSkills()
  assert.equal(p.consumeSilentRefresh('get_skills'), false)
  assert.equal(p.consumeSilentRefresh('get_attributes'), true)
})

test('a panel requested while reconnecting still opens when its reply arrives', () => {
  const p = panels()
  p.sendSilentRefresh('get_skills')
  p.toggleSkills()
  p.resetPanelRequests()
  assert.equal(p.consumeSilentRefresh('get_skills'), false)
})
