// Actual clicks against an isolated synthetic account and independent server.
const { chromium } = require('playwright')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')

async function poll(test, label) {
  for (let i=0; i<150; i++) { if (test()) return; await new Promise(resolve => setTimeout(resolve,100)) }
  throw new Error('Timed out: '+label)
}

async function run() {
  const url = new URL(process.env.MUD_TEST_UI_URL)
  assert.equal(url.hostname,'127.0.0.1')
  const output=process.env.MUD_TEST_UI_OUTPUT, root=path.resolve(__dirname,'../..'), rooms={}
  for (const file of fs.readdirSync(path.join(root,'MudGame-server/resources/worlds'))) {
    Object.assign(rooms,JSON.parse(fs.readFileSync(path.join(root,'MudGame-server/resources/worlds',file),'utf8')).rooms)
  }
  const browser=await chromium.launch({channel:'chrome',headless:true})
  let page
  try {
    page=await browser.newPage({viewport:{width:1440,height:1000}})
    const received=[],errors=[],checks=[]
    let liveRoom
    page.on('pageerror',e=>errors.push(e.message))
    page.on('websocket',socket=>socket.on('framereceived',({payload})=>{
      try {
        const message=JSON.parse(String(payload)); received.push(message)
        if(message.results?.room) liveRoom=message.results.room
        if(message.type==='event'&&message.subtype==='room_update'&&liveRoom&&message.results?.npcs) {
          liveRoom={...liveRoom,npcs:message.results.npcs||liveRoom.npcs}
        }
      }catch{}
    }))
    await page.goto(url.href)
    await page.getByPlaceholder('请输入用户名').fill('quick_review_ui')
    await page.getByPlaceholder('请输入密码').fill('temporary-quick-pass')
    await page.getByRole('button',{name:'登录',exact:true}).click()
    await page.getByText('衔接页面侠',{exact:true}).click()
    await page.getByRole('button',{name:'进入江湖',exact:true}).click()
    await page.locator('.room-header h3').filter({hasText:rooms.ly_old_alley.name}).waitFor()
    const panel=page.locator('.game-info-panel')
    async function openJournal() {
      if (!(await panel.isVisible())) await page.getByRole('button',{name:'令',exact:true}).click()
      await panel.getByRole('button',{name:/^见闻 /}).click()
      await panel.locator('.journal').waitFor()
    }
    async function closePanel() { if (await panel.isVisible()) await panel.locator('.close-btn').click() }
    function card() { return panel.locator('.journey-entry').filter({hasText:'密信与运河转运'}) }
    async function talk(name) {
      const actor=page.locator('.entity-block').filter({has:page.locator('.entity-name',{hasText:new RegExp('^'+name+'$')})}).first()
      await actor.locator('.entity-row').click(); await actor.getByRole('button',{name:'交谈',exact:true}).click()
    }
    async function choose(label) {
      const count=received.filter(m=>m.subtype==='choose_dialogue'&&m.flag).length
      await page.getByRole('button',{name:label}).last().click()
      await poll(()=>received.filter(m=>m.subtype==='choose_dialogue'&&m.flag).length===count+1,'dialogue choice')
    }
    let position='ly_old_alley'
    async function movePublic(target) {
      const queue=[[position,[]]],seen=new Set([position])
      while(queue.length) {
        const [rid,route]=queue.shift()
        if(rid===target) {
          for(const next of route) {
            await page.locator('.room-exits-grid').getByRole('button',{name:rooms[next].name,exact:true}).click()
            await page.locator('.room-header h3').filter({hasText:rooms[next].name}).waitFor(); position=next
          }
          return
        }
        for(const next of Object.values(rooms[rid].exits||{})) if(!seen.has(next)){seen.add(next);queue.push([next,[...route,next]])}
      }
      throw new Error('No route to '+target)
    }
    await openJournal(); assert.equal(await panel.locator('.journey-entry').count(),0); await closePanel()
    await talk('秦三娘'); await choose(/^专长：反向追踪/)
    await openJournal(); await card().getByText(/下一步：.*洛阳府卫/).waitFor()
    assert.doesNotMatch(await card().innerText(),/原卷更次|匿名口信辨/)
    await page.screenshot({path:path.join(output,'next-witness.png'),fullPage:true}); await closePanel()
    checks.push('unseen events hidden; known event points to the missing witness without branch answers')
    await movePublic('ly_00'); await talk('洛阳府卫'); await choose(/^专长：辨伪路引/)
    await movePublic('ly_north_office'); await talk('韩录事'); await choose(/^密信应怎样入卷：隐去受胁者/)
    await openJournal(); await card().getByText(/下一步：.*秦三娘.*追问旧事/).waitFor(); await closePanel()
    await movePublic('ly_old_alley'); await talk('秦三娘'); await choose(/^追问旧事：密信与运河转运的后续$/)
    await openJournal(); await card().getByText(/下一步：.*断缆纤道.*凭匿名口信/).waitFor()
    assert.match(await card().innerText(),/3\/4/)
    await page.screenshot({path:path.join(output,'branch-followup.png'),fullPage:true}); await closePanel()
    checks.push('explicit city choice changes the next person and unlocks one concrete scene route')
    await page.getByRole('button',{name:'境',exact:true}).click()
    await panel.locator('.map-item').filter({has:page.locator('.map-name',{hasText:/^运河追击$/})}).click()
    await panel.getByRole('button',{name:'进入副本',exact:true}).click()
    await page.locator('.room-header h3').filter({hasText:'追粮码头'}).waitFor(); await closePanel()
    await page.locator('.room-exits-grid').getByRole('button',{name:'断缆纤道',exact:true}).click()
    const actionCount=()=>received.filter(m=>m.subtype==='do_action'&&m.flag).length
    async function action(label) {
      const count=actionCount(); await page.getByRole('button',{name:label,exact:true}).click()
      await poll(()=>actionCount()===count+1,'room action')
      return received.filter(m=>m.subtype==='do_action'&&m.flag).at(-1)
    }
    let result=await action('凭匿名口信辨转运绳迹')
    assert.equal(result.results.room.dungeon.puzzleStates.content_trace,'known')
    assert.equal(await page.getByRole('button',{name:'凭原卷更次核发船簿',exact:true}).count(),0)
    await openJournal(); await card().getByText(/已在现场互证/).waitFor()
    assert.match(await card().innerText(),/4\/4/)
    await page.screenshot({path:path.join(output,'scene-consequence.png'),fullPage:true}); await closePanel()
    checks.push('chosen intelligence changes a real instance state and updates guidance to completed')
    // Clear the room through actual combat buttons; no direct protocol injection.
    while(true) {
      const target=liveRoom?.npcs.find(n=>['kill','defeat'].includes(n.combatMode)&&!['DEAD','DEFEATED'].includes(n.state))
      if(!target) break
      const actor=page.locator('.entity-block').filter({has:page.locator('.entity-name',{hasText:new RegExp('^'+target.name+'$')})}).first()
      await actor.locator('.entity-row').click(); await actor.getByRole('button',{name:'击杀',exact:true}).click()
      await poll(()=>liveRoom?.npcs.some(n=>n.id===target.id&&['DEAD','DEFEATED'].includes(n.state)),'combat outcome')
      await page.locator('.entity-name').filter({hasText:target.name+'的尸体'}).first().waitFor()
    }
    await page.locator('.room-exits-grid').getByRole('button',{name:'双闸绞盘',exact:true}).click()
    await page.locator('.room-header h3').filter({hasText:'双闸绞盘'}).waitFor()
    result=await action('以踏浪诀沿潮痕核更次')
    assert.equal(result.results.room.dungeon.puzzleStates.content_compare,'checked')
    assert.equal(result.results.room.dungeon.puzzleStates.gate,'closed')
    await page.screenshot({path:path.join(output,'martial-route.png'),fullPage:true})
    checks.push('learned martial art has a clickable alternate investigation step while final gate stays closed')
    assert.deepEqual(errors,[])
    const report={status:'PASS',checks,runtimeErrors:errors}
    fs.writeFileSync(path.join(output,'connected-browser.json'),JSON.stringify(report,null,2)); console.log(JSON.stringify(report))
  } catch(error) {
    if(page){await page.screenshot({path:path.join(output,'failure.png'),fullPage:true});fs.writeFileSync(path.join(output,'failure.html'),await page.content())}
    throw error
  } finally { await browser.close() }
}
run().catch(error=>{console.error(error);process.exitCode=1})
