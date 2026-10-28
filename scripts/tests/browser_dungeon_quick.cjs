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
  let page
  try {
    page = await browser.newPage({ viewport: { width: 1440, height: 1000 } })
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
    await quick.waitFor({ state: 'visible' })
    await poll(() => received.filter(message => message.subtype === 'get_dungeons').length >= 3, 'second directory refresh')
    await page.getByRole('button', { name: '打坐', exact: true }).click()
    await poll(() => received.some(message => message.subtype === 'meditate' && message.flag), 'meditation accepted')
    await page.waitForFunction(() => [...document.querySelectorAll('button')].find(button => button.textContent.trim() === '快速完成')?.disabled === true)
    await page.getByRole('button', { name: '停止', exact: true }).click()
    await poll(() => received.some(message => message.subtype === 'stop' && message.flag), 'stop accepted')
    await page.waitForFunction(() => [...document.querySelectorAll('button')].find(button => button.textContent.trim() === '快速完成')?.disabled === false)
    checks.push('busy state disables quick completion; stopping restores it without reopening the panel')
    await panel.locator('.map-item').filter({ has: page.locator('.map-name', { hasText: /雪原猎狼/ }) }).click()
    assert.match(await panel.innerText(), /已扩写：双证推理/)
    await page.screenshot({ path: path.join(output, 'content-status.png'), fullPage: true })
    checks.push('retained late dungeon displays its expanded interaction and clue rules')
    await page.getByRole('button', { name: '囊', exact: true }).click()
    const bronze = panel.locator('.item-entry').filter({ has: page.locator('.item-name', { hasText: /^古铜残片$/ }) })
    await bronze.click()
    await bronze.getByRole('button', { name: '查看', exact: true }).click()
    assert.equal(await panel.locator('.detail-sell-price').innerText({ timeout: 5000 }), '2两银子/枚')
    await page.screenshot({ path: path.join(output, 'resale-price.png'), fullPage: true })
    checks.push('backpack shows server-provided 20% resale price: 200 for a 1000-value bronze fragment')
    await page.getByRole('button', { name: '退出游戏', exact: true }).click()
    await page.getByRole('button', { name: '创建新角色', exact: true }).click()
    const longName = '江湖中文长名' + Date.now().toString(36)
    await page.getByPlaceholder('请输入角色名称 (2-16字符)').fill(longName)
    assert.equal(await page.getByRole('button', { name: '创建角色', exact: true }).isEnabled(), true)
    await page.getByRole('button', { name: '创建角色', exact: true }).click()
    await page.getByText(longName, { exact: true }).waitFor()
    checks.push('a long Chinese character name is accepted through the actual creation form')
    assert.deepEqual(errors, [])
    const report = { status: 'PASS', checks, runtimeErrors: errors, quickCommands: sent.filter(message => message.subtype === 'quick_complete_dungeon').length }
    fs.writeFileSync(path.join(output, 'browser.json'), JSON.stringify(report, null, 2))
    console.log(JSON.stringify(report))
  } catch (error) {
    if (page) {
      await page.screenshot({ path: path.join(output, 'failure.png'), fullPage: true })
      fs.writeFileSync(path.join(output, 'failure.html'), await page.content())
    }
    throw error
  } finally {
    await browser.close()
  }
}
run().catch(error => { console.error(error); process.exitCode = 1 })
