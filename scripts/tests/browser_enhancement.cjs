// Isolated headless acceptance; uses only synthetic roles and a local test server.
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
  const browser = await chromium.launch({ channel: 'chrome', headless: true })
  let page
  try {
    page = await browser.newPage({ viewport: { width: 1440, height: 1000 } })
    const errors = [], sent = [], received = [], checks = []
    page.on('pageerror', error => errors.push(error.message))
    page.on('websocket', socket => {
      socket.on('framesent', ({ payload }) => { try { sent.push(JSON.parse(String(payload))) } catch {} })
      socket.on('framereceived', ({ payload }) => { try { received.push(JSON.parse(String(payload))) } catch {} })
    })
    await page.goto(url.href)
    await page.getByPlaceholder('请输入用户名').fill('quick_review_ui')
    await page.getByPlaceholder('请输入密码').fill('temporary-quick-pass')
    await page.getByRole('button', { name: '登录', exact: true }).click()
    await page.getByText('强化验收侠', { exact: true }).click()
    await page.getByRole('button', { name: '进入江湖', exact: true }).click()
    await page.getByRole('button', { name: '囊', exact: true }).click()
    const panel = page.locator('.game-info-panel')
    const relic = panel.locator('.item-entry').filter({ has: page.locator('.item-name', { hasText: /^唐门宝鉴$/ }) })
    await relic.click()
    await relic.getByRole('button', { name: /^强化/ }).click()
    const enhance = page.getByRole('dialog', { name: '强化确认', exact: true })
    assert.match(await enhance.innerText(), /攻击 \+1/)
    assert.match(await enhance.innerText(), /暴击率 \+0\.1个百分点/)
    await page.screenshot({ path: path.join(output, 'coefficients.png'), fullPage: true })
    await enhance.getByRole('button', { name: '取消', exact: true }).click()
    checks.push('multi-affix preview shows separate coefficients, Chinese crit label and decimal percentage points')
    const sword = panel.locator('.item-entry').filter({ has: page.locator('.item-name', { hasText: /^铁剑$/ }) })
    await sword.click()
    const reset = page.getByRole('dialog', { name: '强化重置确认', exact: true })
    await sword.getByRole('button', { name: '免费重置', exact: true }).click()
    assert.match(await reset.innerText(), /古铜残片×3/)
    assert.match(await reset.innerText(), /→ \+0/)
    await reset.getByRole('button', { name: '取消', exact: true }).click()
    assert.equal(sent.filter(message => message.subtype === 'reset_enhancement').length, 0)
    await sword.getByRole('button', { name: '免费重置', exact: true }).click()
    await page.screenshot({ path: path.join(output, 'reset-confirmation.png'), fullPage: true })
    const money = received.filter(message => message.subtype === 'get_backpack' && message.flag).at(-1).results.money
    await reset.getByRole('button', { name: '确认重置', exact: true }).evaluate(button => { button.click(); button.click() })
    await poll(() => received.some(message => message.subtype === 'get_backpack' && message.results?.money === money + 600), 'refund refresh')
    assert.equal(sent.filter(message => message.subtype === 'reset_enhancement').length, 1)
    await sword.getByRole('button', { name: '免费重置', exact: true }).waitFor({ state: 'hidden' })
    const result = received.filter(message => message.subtype === 'get_backpack' && message.flag).at(-1)
    const updated = result.results.items.find(item => item.id === 'iron_sword')
    assert.equal(updated.enhancement.level, 0)
    assert.equal(updated.enhancement.canReset, false)
    checks.push('reset cancellation sends nothing; double confirmation sends once; 600/3 refunded and reset button disappears')
    await panel.locator('.close-btn').click()
    const merchant = page.locator('.entity-block').filter({ has: page.locator('.entity-name', { hasText: /^裴玄$/ }) })
    await merchant.locator('.entity-row').click()
    await merchant.getByRole('button', { name: '交易', exact: true }).click()
    const shop = page.locator('.shop-panel')
    const smoke = shop.locator('.shop-item').filter({ has: page.locator('.item-name', { hasText: /^烟尘粉$/ }) })
    assert.match(await smoke.innerText(), /无限供应/)
    await smoke.click()
    await shop.getByRole('button', { name: '+10', exact: true }).click()
    assert.equal(await shop.locator('input[type=number]').inputValue(), '11')
    await shop.getByRole('button', { name: '购买', exact: true }).click()
    await poll(() => received.some(message => message.subtype === 'buy' && message.flag && message.results?.boughtAmount === 11), 'buy eleven smoke')
    const request = sent.filter(message => message.subtype === 'buy').at(-1)
    assert.equal(request.args.amount, 11)
    await smoke.waitFor()
    assert.match(await smoke.innerText(), /无限供应/)
    await page.screenshot({ path: path.join(output, 'unlimited-smoke.png'), fullPage: true })
    checks.push('shop labels unlimited smoke and permits a real purchase of 11 packs without old five-pack cap')
    assert.deepEqual(errors, [])
    const report = { status: 'PASS', checks, runtimeErrors: errors }
    fs.writeFileSync(path.join(output, 'enhancement-browser.json'), JSON.stringify(report, null, 2))
    console.log(JSON.stringify(report))
  } catch (error) {
    if (page) {
      await page.screenshot({ path: path.join(output, 'failure.png'), fullPage: true })
      fs.writeFileSync(path.join(output, 'failure.html'), await page.content())
    }
    throw error
  } finally { await browser.close() }
}
run().catch(error => { console.error(error); process.exitCode = 1 })
