// Content-depth E2E in an independent headless browser and synthetic account.
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
  const root = path.resolve(__dirname, '../..')
  const rooms = {}
  for (const file of fs.readdirSync(path.join(root, 'MudGame-server/resources/worlds'))) {
    Object.assign(rooms, JSON.parse(fs.readFileSync(path.join(root, 'MudGame-server/resources/worlds', file), 'utf8')).rooms)
  }
  const browser = await chromium.launch({ channel: 'chrome', headless: true })
  let page
  try {
    page = await browser.newPage({ viewport: { width: 1440, height: 1000 } })
    const errors = [], received = [], checks = []
    page.on('pageerror', error => errors.push(error.message))
    page.on('websocket', socket => socket.on('framereceived', ({ payload }) => {
      try { received.push(JSON.parse(String(payload))) } catch {}
    }))
    await page.goto(url.href)
    await page.getByPlaceholder('请输入用户名').fill('quick_review_ui')
    await page.getByPlaceholder('请输入密码').fill('temporary-quick-pass')
    await page.getByRole('button', { name: '登录', exact: true }).click()
    await page.getByText('见闻页面侠', { exact: true }).click()
    await page.getByRole('button', { name: '进入江湖', exact: true }).click()
    let position = 'ly_old_alley'
    async function movePublic(target) {
      const queue = [[position, []]], seen = new Set([position])
      while (queue.length) {
        const [rid, route] = queue.shift()
        if (rid === target) {
          for (const next of route) {
            await page.locator('.room-exits-grid').getByRole('button', { name: rooms[next].name, exact: true }).click()
            await page.locator('.room-header h3').filter({ hasText: rooms[next].name }).waitFor()
            position = next
          }
          return
        }
        for (const next of Object.values(rooms[rid].exits || {})) {
          if (!seen.has(next)) { seen.add(next); queue.push([next, [...route, next]]) }
        }
      }
      throw new Error('No route: ' + target)
    }
    async function talk(name) {
      const actor = page.locator('.entity-block').filter({ has: page.locator('.entity-name', { hasText: new RegExp('^' + name + '$') }) })
      await actor.locator('.entity-row').click()
      await actor.getByRole('button', { name: '交谈', exact: true }).click()
    }
    async function choose(label) {
      const count = received.filter(message => message.subtype === 'choose_dialogue' && message.flag).length
      await page.getByRole('button', { name: label }).last().click()
      await poll(() => received.filter(message => message.subtype === 'choose_dialogue' && message.flag).length === count + 1, 'dialogue choice')
    }
    await talk('秦三娘')
    assert.equal(received.some(message => message.results?.journal?.some(entry => entry.id === 'discovery_qingyi_spy_specialty')), false)
    await choose(/^专长：反向追踪/)
    await movePublic('ly_00'); await talk('洛阳府卫'); await choose(/^专长：辨伪路引/)
    await movePublic('ly_north_office'); await talk('韩录事')
    await choose(/^密信应怎样入卷：隐去受胁者/)
    const opposite = page.getByRole('button', { name: /^密信应怎样入卷：保留完整名单/ })
    assert.equal(await opposite.evaluateAll(buttons => buttons.every(button => button.disabled)), true)
    checks.push('different witnesses require explicit questions; a genuine item enables one mutually exclusive case conclusion')
    await page.getByRole('button', { name: '令', exact: true }).click()
    const panel = page.locator('.game-info-panel')
    await panel.getByRole('button', { name: /^见闻 / }).click()
    await panel.locator('.journal').waitFor()
    const entry = panel.locator('.journal details').filter({ has: page.locator('summary', { hasText: /韩录事.*密信应怎样入卷/ }) })
    await entry.locator('summary').click()
    assert.match(await entry.innerText(), /证词被匿名保全/)
    await page.screenshot({ path: path.join(output, 'journal.png'), fullPage: true })
    checks.push('existing quest panel replays only acquired witness/conclusion text')
    await panel.locator('.close-btn').click()
    await movePublic('ly_east_market')
    await page.getByRole('button', { name: '囊', exact: true }).click()
    const clothes = panel.locator('.item-entry').filter({ has: page.locator('.item-name', { hasText: /^粗布衣$/ }) })
    await clothes.click(); await clothes.getByRole('button', { name: '装备', exact: true }).click()
    await poll(() => received.some(message => message.subtype === 'equip' && message.flag), 'clothes equip')
    await panel.locator('.close-btn').click()
    await talk('杜掌柜'); await choose(/^出示粗布衣/)
    await page.screenshot({ path: path.join(output, 'equipped-evidence.png'), fullPage: true })
    checks.push('an equipped ordinary item has its own specialist interpretation without consuming it')
    await page.getByRole('button', { name: '境', exact: true }).click()
    await panel.locator('.map-item').filter({ has: page.locator('.map-name', { hasText: /^运河追击$/ }) }).click()
    assert.match(await panel.innerText(), /已扩写：往返复核/)
    await panel.getByRole('button', { name: '进入副本', exact: true }).click()
    await page.locator('.room-header h3').filter({ hasText: '追粮码头' }).waitFor()
    if (await panel.isVisible()) await panel.locator('.close-btn').click()
    await page.locator('.room-exits-grid').getByRole('button', { name: '断缆纤道', exact: true }).click()
    await page.getByRole('button', { name: '以粗布衣校验前段痕迹（不消耗）', exact: true }).click()
    await poll(() => received.some(message => message.subtype === 'do_action' && message.results?.room?.dungeon?.puzzleStates?.content_trace === 'known'), 'evidence aid updates instance')
    assert.equal(await page.getByRole('button', { name: /以深海寒珠校验/ }).count(), 0)
    await page.screenshot({ path: path.join(output, 'item-aid.png'), fullPage: true })
    checks.push('expanded chapter advertises its mechanic; interpreted held evidence changes a real puzzle state; unavailable aids stay hidden')
    assert.deepEqual(errors, [])
    const report = { status: 'PASS', checks, runtimeErrors: errors }
    fs.writeFileSync(path.join(output, 'content-depth-browser.json'), JSON.stringify(report, null, 2))
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
