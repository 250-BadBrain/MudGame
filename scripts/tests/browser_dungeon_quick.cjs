// Isolated headless browser E2E. Requires Playwright and an existing local
// Chrome/Edge installation; never connects to a user's running browser.
const { chromium } = require('playwright')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')

async function poll(condition, label) {
  for (let i = 0; i < 100; i++) {
    if (condition()) return
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  throw new Error('Timed out: ' + label)
}

async function run() {
  const url = new URL(process.env.MUD_TEST_UI_URL)
  assert.equal(url.hostname, '127.0.0.1')
  const output = process.env.MUD_TEST_UI_OUTPUT
  fs.mkdirSync(output, { recursive: true })
  const browser = await chromium.launch({ channel: process.env.MUD_TEST_BROWSER_CHANNEL || 'chrome', headless: true })
  try {
    const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } })
    const errors = [], sent = [], received = []
    page.on('pageerror', error => errors.push(error.message))
    page.on('websocket', socket => {
      socket.on('framesent', ({ payload }) => { try { sent.push(JSON.parse(String(payload))) } catch {} })
      socket.on('framereceived', ({ payload }) => { try { received.push(JSON.parse(String(payload))) } catch {} })
    })
    await page.goto(url.href)
    await page.getByPlaceholder('请输入用户名').fill('quick_review_ui')
    await page.getByPlaceholder('请输入密码').fill('temporary-quick-pass')
    await page.getByRole('button', { name: '登录', exact: true }).click()
    await page.getByText('快速验收侠', { exact: true }).click()
    await page.getByRole('button', { name: '进入江湖', exact: true }).click()
    await page.getByRole('button', { name: '境', exact: true }).click()
    const panel = page.locator('.game-info-panel')
    await panel.locator('.map-item').filter({ has: page.locator('.map-name', { hasText: /^荒庙夜雨$/ }) }).click()
    const quick = panel.getByRole('button', { name: '快速完成', exact: true })
    assert.equal(await quick.isDisabled(), true)
    const checks = ['uncleared dungeon shows disabled quick button']
    await panel.locator('.map-item').filter({ has: page.locator('.map-name', { hasText: /扬子江惊变/ }) }).click()
    assert.equal(await quick.isEnabled(), true)
    assert.match(await panel.innerText(), /当前免费/)
    assert.match(await panel.innerText(), /15%/)
    const before = sent.filter(message => message.subtype === 'quick_complete_dungeon').length
    // Two click events in one UI event turn exercise the synchronous pending guard.
    await quick.evaluate(button => { button.click(); button.click() })
    await poll(() => received.some(message => message.subtype === 'quick_complete_dungeon' && message.flag), 'quick result')
    assert.equal(sent.filter(message => message.subtype === 'quick_complete_dungeon').length, before + 1)
    const first = received.find(message => message.subtype === 'quick_complete_dungeon' && message.flag)
    assert.equal(first.results.settlement.rewardItems.herb, 4)
    const request = sent.filter(message => message.subtype === 'quick_complete_dungeon').at(-1)
    await poll(() => received.some(message => message.subtype === 'get_dungeons' && message.results?.dungeons?.[0]?.quickCompleteSequence === request.args.expectedSequence + 1), 'directory refresh')
    const log = page.locator('.right-log-area').getByText(/快速完成《扬子江惊变》/)
    await log.first().waitFor()
    assert.match((first.logs || []).join(''), /止血草×4/)
    checks.push('cleared button shows free/rate rules; double click sends one command; actual loot appears')
    await quick.click()
    await poll(() => received.filter(message => message.subtype === 'quick_complete_dungeon' && message.flag).length === 2, 'second completion')
    checks.push('next intentional click completes again with refreshed sequence')
    await page.screenshot({ path: path.join(output, 'quick-complete.png'), fullPage: true })
    assert.deepEqual(errors, [])
    const report = { status: 'PASS', checks, runtimeErrors: errors, quickCommands: sent.filter(message => message.subtype === 'quick_complete_dungeon').length }
    fs.writeFileSync(path.join(output, 'browser.json'), JSON.stringify(report, null, 2))
    console.log(JSON.stringify(report))
  } finally {
    await browser.close()
  }
}
run().catch(error => { console.error(error); process.exitCode = 1 })
