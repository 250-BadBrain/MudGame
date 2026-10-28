"""Author existing entities only. Preserve IDs, reward budgets and skill sources.

Uses 71085f3 as the documented content baseline for original text and gate data;
refuses changed entity catalogs/rewards. Re-running replaces only this pass's
content_* additions. Never reads or writes player saves.
"""
import copy
import json
import subprocess
from pathlib import Path

from content_depth_chapters import DEPTH, EPILOGUES, RELIC_NAMES
from content_depth_people import PEOPLE, RETURNING_PAIRS, RETURNING_PUBLIC

ROOT=Path(__file__).resolve().parents[1]
RES=ROOT/'MudGame-server/resources'
BASE='71085f3'

def read(p): return json.loads(p.read_text(encoding='utf-8'))
def original(path): return json.loads(subprocess.check_output(['git','show',BASE+':MudGame-server/resources/'+path],cwd=ROOT).decode('utf-8'))
def original_files(directory):
    prefix='MudGame-server/resources/'+directory+'/'
    paths=subprocess.check_output(['git','ls-tree','-r','--name-only',BASE,'--',prefix],cwd=ROOT).decode('utf-8').splitlines()
    return sorted(p.removeprefix(prefix) for p in paths if p.endswith('.json'))
def write(p,v):
    text=json.dumps(v,ensure_ascii=False,indent=2)+'\n'
    p.write_bytes(text.replace('\n','\r\n').encode('utf-8') if p.name=='npcs.json' else text.encode('utf-8'))

def main():
    items,npcs,dialogs,skills=[read(RES/(s+'.json')) for s in ['items','npcs','dialogues','skills']]
    original_items, original_npcs=[original(s+'.json') for s in ['items','npcs']]
    worlds={p:read(p) for p in sorted((RES/'worlds').glob('*.json'))}
    chapters={p:read(p) for p in sorted((RES/'dungeons').glob('*.json'))}
    assert sorted(p.name for p in worlds)==original_files('worlds'), 'world catalog changed'
    assert sorted(p.name for p in chapters)==original_files('dungeons'), 'chapter catalog changed'
    original_worlds={p:original('worlds/'+p.name) for p in worlds}
    originals={d['id']:original('dungeons/'+p.name) for p,d in chapters.items()}
    by_id={d['id']:d for d in chapters.values()}
    baseline=dict(items=sorted(original_items),npcs=sorted(original_npcs),skills=sorted(original('skills.json')),quests=sorted(original('quests.json')),factions=sorted(original('factions.json')),
       worlds=sorted(w['id'] for w in original_worlds.values()),rooms=sorted(r for w in original_worlds.values() for r in w['rooms']),
       dungeons=sorted(originals),dungeonRooms={k:sorted(v['rooms']) for k,v in originals.items()},
       rewards={k:v['rewards'] for k,v in originals.items()}, quickRewards={k:v.get('quickRewards',{}) for k,v in originals.items()})
    assert sorted(items)==baseline['items'] and sorted(npcs)==baseline['npcs']
    assert sorted(skills)==baseline['skills'] and sorted(read(RES/'quests.json'))==baseline['quests']
    assert sorted(read(RES/'factions.json'))==baseline['factions']
    assert sorted(w['id'] for w in worlds.values())==baseline['worlds']
    assert sorted(r for w in worlds.values() for r in w['rooms'])==baseline['rooms']
    for d in chapters.values():
        old=originals[d['id']]
        assert sorted(d['rooms'])==sorted(old['rooms']) and d['rewards']==old['rewards'] and d.get('quickRewards',{})==old.get('quickRewards',{})
        d['puzzleActions']={k:v for k,v in d.get('puzzleActions',{}).items() if not k.startswith('content_')}
        d['puzzleInitialStates']={k:v for k,v in d.get('puzzleInitialStates',{}).items() if not k.startswith('content_')}
        for r in d['rooms'].values(): r['actions']=[a for a in r.get('actions',[]) if not a.startswith('content_')]
    for d in dialogs.values(): d['options']=[o for o in d.get('options',[]) if not o['id'].startswith('content_')]
    people=copy.deepcopy(PEOPLE)
    public_rooms={rid:(w,room) for w in worlds.values() for rid,room in w['rooms'].items()}
    # Restore all unused public actors to existing locations, with distinct roles.
    for line in RETURNING_PUBLIC.strip().splitlines():
        nid,rid,name,specialty,cid=line.split('|')
        npc=npcs[nid];npc.update(name=name,instanceOnly=False,nonCombat=True,combatMode='protected',aggressive=False)
        room=public_rooms[rid][1]; room.setdefault('npcs',[])
        if nid not in room['npcs']: room['npcs'].append(nid)
        s=DEPTH.get(cid)
        testimony=(s['a']+' '+s['b']) if s else EPILOGUES[cid][3]
        people[nid]=dict(specialty=specialty,testimony=testimony,chapter=cid)
    for cid,pair in RETURNING_PAIRS.items():
        d=by_id[cid];room=d['rooms'][d['spawnRoomId']]
        for index in [0,2]:
            nid,name=pair[index:index+2]
            npc=npcs[nid];npc.update(name=name,instanceOnly=True,nonCombat=True,combatMode='protected',aggressive=False)
            if nid not in room.setdefault('npcs',[]):room['npcs'].append(nid)
            s=DEPTH.get(cid)
            testimony=(s['a'] if index==0 else s['b']) if s else EPILOGUES[cid][3]
            people[nid]=dict(specialty=('现场痕迹的原始见证' if index==0 else '旧制与误判的校对'),testimony=testimony,chapter=cid)
    places={}
    for rid,(w,room) in public_rooms.items():
        for nid in room.get('npcs',[]):places.setdefault(nid,[]).append((w['name'],room['name'],None))
    for cid,d in by_id.items():
        for rid,room in d['rooms'].items():
            scene=DEPTH[cid]['names'][list(d['rooms']).index(rid)] if cid in DEPTH else room['name']
            for nid in room.get('npcs',[]):places.setdefault(nid,[]).append((d['name'],scene,cid))
        for a in d['puzzleActions'].values():
            for nid in a.get('spawnNpcs',[])+a.get('failureSpawnNpcs',[]):places.setdefault(nid,[]).append((d['name'],'支线出场',cid))
    assert set(places)==set(npcs), 'NPC has no actual location: '+str(set(npcs)-set(places))
    # Every existing actor has an explicit, reviewable role and context-specific
    # observation. Existing quest/teaching/trade options remain in place.
    for nid,n in npcs.items():
        old_n=original_npcs[nid]
        if not old_n.get('combatMode') and not old_n.get('nonCombat') and not old_n.get('dialogueId') and not old_n.get('shopItems') and not old_n.get('skills') and not old_n.get('questIds') and nid not in {x for pair in RETURNING_PAIRS.values() for x in (pair[0],pair[2])} and nid not in {line.split('|')[0] for line in RETURNING_PUBLIC.strip().splitlines()}:
            n['combatMode']='kill'
        if nid not in people:
            context,scene,cid=places[nid][0]
            s=DEPTH.get(cid)
            ep=EPILOGUES.get(cid)
            active=not n.get('nonCombat') and n.get('combatMode')!='protected'
            clue=(s['b'] if 'boss' in nid or n.get('combatMode')=='defeat' else s['a']) if s else (ep[3] if ep else n.get('description',''))
            people[nid]=dict(specialty=(f'{context}·{scene}的武备与守位' if active else f'{context}·{scene}的在场证词'),testimony=f'你仔细辨认{n["name"]}在{scene}留下的痕迹。'+clue,chapter=cid or 'abandoned_temple_rain')
        p=people[nid]
        n['specialty']=p['specialty']
        n['description']=original_npcs[nid].get('description','').split('【专长】')[0]+'【专长】'+p['specialty']+'。交谈后明确选择“专长”可记录见闻。'
        if nid in {x for pair in RETURNING_PAIRS.values() for x in (pair[0],pair[2])}: n['description']=n['name']+'亲历旧事，不代替本章首领作战。【专长】'+p['specialty']+'。'
        did=n.get('dialogueId') or 'content_npc_'+nid
        if not n.get('dialogueId'):n['dialogueId']=did
        if did not in dialogs:dialogs[did]=dict(id=did,npcId=nid,title=n['name']+'的在地专长',opening=n['name']+'将注意力转向眼前的事：“问具体的事，才有具体的答复。”',options=[])
        assert dialogs[did]['npcId']==nid, (did,nid)
        dialogs[did]['options'].append(dict(id='content_specialty',text='专长：'+p['specialty'],response=p['testimony'],explicit=True,setFlags={'discovery_'+nid+'_specialty':True}))
        if n.get('role') in ['enemy','boss'] and not n.get('nonCombat') and n.get('combatMode')!='protected':
            n['description']=n['description'].replace('选择“专长”','选择对应观察项')
            if did.startswith('content_npc_'):dialogs[did]['opening']='你保持距离，观察'+n['name']+'的武备，并未主动开战。'
            observation=dialogs[did]['options'][-1]
            observation['text']='观察武备：'+p['specialty']
            observation['response']='你观察'+n['name']+'，核对其所处位置与装备痕迹。'+p['testimony']
    # Eight city cases use different witnesses and real objects. These are
    # dialogue continuations, not new quest entities or free reward sources.
    cases=[
     ('office_clerk','sealed_letter',['qingyi_spy','luoyang_guard'],'密信应怎样入卷','隐去受胁者姓名，只追转运主使','保留完整名单，提交可复查原卷','证词被匿名保全，秦三娘以后只替证人转达不涉及姓名的消息。','原卷保留了每个签名，韩录事承诺每次调阅都留查阅记录。','old_refugee'),
     ('bs_tang_steward','iron_pickaxe',['bs_jiange_clerk','bs_wine_maker'],'栈道应先修还是先封','先支旧桩，给返乡人留一条窄路','先封危险段，公开所有松索位置','唐家坳的修索办法交给驿丞，茶棚不再把缺席的赶路人直接算作遇匪。','栈道暂时封存，酒娘替留宿者辨草药，没人再被催着踏上尚未压实的新桩。','bs_tea_house'),
     ('ca2_monk','bronze_box_rubbing',['academy_teacher','ca2_erci_soldier'],'异本证词该如何流传','附上异文公开抄本，不替古人补字','先封存抄本，逐个核对见证人的口述','字形与异文一并公开，任何后来者都能指出抄错的一笔。','抄本暂不外传，知客把每个不相同的说法分开记录，不让整齐口径盖过事实。','ca2_west_keeper'),
     ('dd2_pool_poet','old_map_fragment',['dd2_palace_guard','dd2_silk_merchant'],'晓月与账册谁作见证','以诗指方位，以原账认责任','先保全原账，暂不公布桥下入口','诗没有被当成口供，账册也没有被当成谜语，桥下的来历终于可以各自查证。','入口暂不示众，锦绸庄把押运次序交给有名字可追的承运人。','dd2_silk_merchant'),
     ('dm_old_captain','dry_ration',['dm_jiaoren','dm_pearl_diver'],'沉船该捞货还是留证','先标出活梁，给后来潜水者留通路','先封住脆弱舱口，保留沉船原状','采珠人把压舱放水的先后留给所有下水的人，船货不再比活人优先。','船骸暂不打捞，珠娘守着原来的潮门盐线，让下一次调查仍能核对现场。','dm_sailor'),
     ('jn_pool_keeper','iron_sword',['jn_zither_player','jn_calligrapher'],'剑池应该怎样传给后来者','开清泉，留养剑方法给所有来者','守旧泉，公开为何不能急于拔剑','守池人的木牌多了一行公开的养剑次序，琴师愿意替不会读字的人传唱。','名剑暂留池底，临帖者只拓原有刻痕，不替不曾见过剑的人编造胜负。','jn_silk_merchant'),
     ('sb_centurion','dry_ration',['sb_scout_captain','sb_beacon_guard'],'兵册上失联者该怎样记','先按归队路线寻人，暂不填阵亡','保留失联记录，同时公开最后更牌','空格没有被草率填成战死，斥候与马商把反穿靴的假迹标在同一张旧路图上。','更牌随失联记录一同公布，年轻戍卒终于不必独自承担假烽的责任。','sb_horse_trader'),
     ('yz_yamen_clerk','reed_root',['yz_boat_girl','yz_salt_merchant'],'三联账怎样才算对清','保全脚夫的匿名实账，逐担核对','三联同时封存，交不同见证人复查','匿名尾数被保留下来，程盐商也须解释自己的损耗，不再由最弱的人独担亏空。','三联分交不同的人保存；船娘只证明水路，茶博士只证明来客时刻，各自不能冒认别人的见证。','yz_tea_master'),
    ]
    for actor,iid,witnesses,topic,left,right,left_result,right_result,aftermath in cases:
        flag='discovery_case_'+actor
        req={'presentItems':{iid:1},'flags':{**{'discovery_'+n+'_specialty':True for n in witnesses},flag+'_left':False,flag+'_right':False}}
        instructions=' 若要继续“'+topic+'”，请携'+items[iid]['name']+'，先向'+'、'.join(npcs[n]['name'] for n in witnesses)+'明确询问各自专长，再回来作出选择；出示不耗物，也没有额外物品奖励。'
        next(o for o in dialogs[npcs[actor]['dialogueId']]['options'] if o['id']=='content_specialty')['response']+=instructions
        for side,label,result in [('left',left,left_result),('right',right,right_result)]:
            dialogs[npcs[actor]['dialogueId']]['options'].append(dict(id='content_case_'+side,text=topic+'：'+label+'（此事只选一种）',response=result,explicit=True,requires=copy.deepcopy(req),setFlags={flag+'_'+side:True}))
            dialogs[npcs[aftermath]['dialogueId']]['options'].append(dict(id='content_after_'+actor+'_'+side,text='追问“'+topic+'”之后的变化',response=npcs[aftermath]['name']+'说起后续：'+result,explicit=True,requires={'flags':{flag+'_'+side:True}},setFlags={'discovery_aftermath_'+actor+'_'+side:True}))
    # Private copies keep existing clue stock from locking five players out of
    # optional city cases; full face-value payment still respects resale rules.
    for actor,iid in [('academy_teacher','bronze_box_rubbing'),('station_master_luo','old_map_fragment')]:
        price=items[iid]['value']
        dialogs[npcs[actor]['dialogueId']]['options'].append(dict(id='content_copy_clue_'+iid,text='誊取'+items[iid]['name']+'（'+str(price)+'铜，本人一次）',response='你按原价支付誊取费用，获得既有的'+items[iid]['name']+'。这只是核证材料，不会发放任何武学资格。',explicit=True,takeMoney=price,giveItems={iid:1}))
    # Public rooms become distinct sources, preserving all existing quest clues.
    for rid,(w,room) in public_rooms.items():
        old=original('worlds/'+next(p.name for p,v in worlds.items() if v['id']==w['id']))['rooms'][rid]
        actors=[nid for nid in room.get('npcs',[]) if nid in people]
        preferred={'ly_north_office':'office_clerk','ly_south_gate':'escort_bureau_chief'}.get(rid)
        actor=preferred if preferred in actors else (actors[0] if actors else None)
        clue=people[actor]['testimony'] if actor else room.get('rumors',[room['description']])[0]
        if not old.get('searchFlag'):
            room['searchFlag']='discovery_place_'+rid
            room['searchMessage']=f'你在{room["name"]}核对现场：'+room['description']+' 进一步可核验的是：'+clue
        if 'search' not in room.setdefault('actions',[]): room['actions'].append('search')
        room['contentRole']=f'{w["name"]}的{room["name"]}：'+(people[actors[0]]['specialty'] if actors else '从现场痕迹校验传闻')
    # Rebuild thin gates with seven genuinely different operation graphs.
    for cid,s in DEPTH.items():
        d=by_id[cid];old=originals[cid];rids=list(old['rooms']); a,b,c,last=rids
        d['description']=s['a']+' '+s['b']+' 这一次要先判断现场，再决定怎样走完这条路。'
        for rid,name in zip(rids,s['names']):
            d['rooms'][rid]['name']=name
        descriptions=[f'{s["names"][0]}保留来路，调查所得可以带回这里复核。',s['a'],s['b'],s['end']]
        for rid,text in zip(rids,descriptions):d['rooms'][rid]['description']=text
        d['puzzleActions']['open_gate']=copy.deepcopy(old['puzzleActions']['open_gate'])
        gate=d['puzzleActions']['open_gate']; gate.update(label='推断：'+s['right'].split('，')[0],nextState='considered',message=s['a']+' '+s['b']+' 你准备怎样处理？',failureMessage='线索还不完整。先察看前段痕迹与关口细节，再作判断。')
        old_flags=gate.pop('setFlags',{})
        d['puzzleInitialStates'].update(content_trace='unseen',content_detail='unseen',content_compare='unseen',content_support='unseen')
        solution=[]
        def add(aid,label,rid,puzzle,start,target,message,**extra):
            d['puzzleActions'][aid]=dict(label=label,puzzleId=puzzle,rooms=[rid],fromStates=[start],nextState=target,message=message,repeatMessage=message,**extra)
            if aid not in d['rooms'][rid].setdefault('actions',[]):d['rooms'][rid]['actions'].append(aid)
        add('content_trace','察看：'+s['names'][1],b,'content_trace','unseen','known',s['a'])
        add('content_detail','核验：'+s['names'][2],c,'content_detail','unseen','known',s['b'])
        solution.append(dict(roomId=b,actionId='content_trace'))
        if s['mode']=='rescue':
            guards=list(dict.fromkeys(n for n in d['rooms'][b].get('npcs',[]) if not npcs[n].get('nonCombat') and npcs[n].get('combatMode')!='protected'))
            d['puzzleActions']['content_trace']['requiredDefeatNpcs']=guards
            d['puzzleActions']['content_trace']['failureMessage']='先让本处的拦路者失去战意，再救助与查验；不必消耗稀有物品。'
        if s['mode']=='traversal':
            add('content_compare','回头复核原始记录',a,'content_compare','unseen','checked',s['a']+' 你把后来见到的细节与来处旧记核在一起。',requiredPuzzleStates={'content_trace':'known'})
            solution.append(dict(roomId=a,actionId='content_compare'))
        solution.append(dict(roomId=c,actionId='content_detail'))
        required={'content_trace':'known','content_detail':'known'}
        if s['mode']=='traversal':required['content_compare']='checked'
        if s['mode']=='tool':
            add('content_brace','先做人工支护',c,'content_support','unseen','bracing','你先找稳固的着力点，支护尚需最后一道固定。',requiredPuzzleStates={'content_detail':'known'})
            add('content_secure','再固定支护',c,'content_support','bracing','ready','支护已经稳固，现在可以不冒险地判断机关。')
            add('content_pickaxe','用铁镐校稳支点（不消耗）',c,'content_support','unseen','ready','你以铁镐作短暂杠杆校稳支点，随后把铁镐收回。',presentItems={'iron_pickaxe':1},requiredPuzzleStates={'content_detail':'known'})
            required['content_support']='ready'
            solution += [dict(roomId=c,actionId='content_brace'),dict(roomId=c,actionId='content_secure')]
        gate['requiredPuzzleStates']=required
        gate['choices']=[dict(id='content_decide_good',label=s['right']),dict(id='content_decide_wrong',label=s['wrong']),dict(id='content_decide_other',label=s['other'])]
        # Rotate positions so choosing the first button is not a universal solve.
        shift=list(DEPTH).index(cid)%3;gate['choices']=gate['choices'][shift:]+gate['choices'][:shift]
        solution.append(dict(roomId=c,actionId='open_gate'))
        for aid in ['content_decide_good','content_decide_wrong','content_decide_other']:
            good=aid=='content_decide_good'; alternative=s['mode']=='polarity' and aid=='content_decide_wrong'
            target='open' if good or alternative else 'closed'
            text=s['end'] if good else (s['end']+' 你选择另一种承诺，后果由自己承担；守关者仍保留完整招势。' if alternative else '这个判断忽略了现场的反证：'+s['b']+' 你及时收手，没有损失物品；可以重新作出判断。')
            branch=dict(label=next(x['label'] for x in gate['choices'] if x['id']==aid),puzzleId='gate',rooms=[c],fromStates=['considered'],nextState=target,message=text,repeatMessage=text)
            if good:branch['setFlags']=dict(old_flags,**{'discovery_chapter_'+cid:True})
            if alternative:branch['setFlags']={'discovery_alternative_'+cid:True}
            d['puzzleActions'][aid]=branch
        if s['mode'] in ['sequence','cipher']:
            first=d['puzzleActions']['content_decide_good']; first['nextState']='aligned';first.pop('setFlags',None)
            first['message']='第一步已与线索相符；仍须决定下一步，不能一口气把所有机关推到底。'
            first['choices']=[dict(id='content_finish',label=s['right'].split('，')[-1]),dict(id='content_reverse',label='反过来，再做一次最先的动作')]
            d['puzzleActions']['content_finish']=dict(label=s['right'].split('，')[-1],puzzleId='gate',rooms=[c],fromStates=['aligned'],nextState='open',message=s['end'],setFlags=dict(old_flags,**{'discovery_chapter_'+cid:True}))
            d['puzzleActions']['content_reverse']=dict(label='颠倒次序',puzzleId='gate',rooms=[c],fromStates=['aligned'],nextState='closed',message='先后次序被颠倒了，你撤回到安全位置。重新判断即可，没有物品损失。')
            solution += [dict(roomId=c,actionId='content_decide_good'),dict(roomId=c,actionId='content_finish')]
        else: solution.append(dict(roomId=c,actionId='content_decide_good'))
        d['contentStatus']='enriched';d['contentNote']='已扩写：'+dict(deduction='双证推理',sequence='分步次序',cipher='留白与对照',polarity='两种立场',rescue='先稳现场再判断',traversal='往返复核',tool='手工支护或器物辅助')[s['mode']]+'；线索可现场取得，失败可重试，不要求稀有掉落。'
        d['contentFeature']=dict(kind=s['mode'],solution=solution,mistake='content_decide_other',summary=s['end'])
        for objective in d.get('objectives',[]):
            if objective.get('targetId')=='gate:open':objective['description']='核验现场并完成'+s['names'][2]+'的判断'
        # Replace generic briefings without introducing new bound story actors.
        for aid,action in d['puzzleActions'].items():
            if action.get('puzzleId')=='briefing':action['message']=s['a']+' 此处不能只凭武力开门；沿路观察，并核验'+s['names'][2]+'的细节。'
    for cid,ep in EPILOGUES.items():
        d=by_id[cid]; rooms=list(d['rooms']); last=rooms[-1]
        boss_ids=[o['targetId'] for o in d.get('objectives',[]) if o['type']=='kill']
        for rid,room in d['rooms'].items():
            if boss_ids and boss_ids[-1] in room.get('npcs',[]):last=rid
        p='content_epilogue';d['puzzleInitialStates'][p]='unread'
        main=dict(label='复盘：'+ep[0],puzzleId=p,rooms=[last],fromStates=['unread'],nextState='considered',message=ep[3]+' 你准备怎样记录？',choices=[dict(id='content_record_people',label=ep[1]),dict(id='content_record_route',label=ep[2])])
        present=[n for n in d['rooms'][last].get('npcs',[]) if n in boss_ids and npcs[n].get('combatMode')!='protected' and not npcs[n].get('nonCombat')]
        if present:main['requiredDefeatNpcs']=list(dict.fromkeys(present));main['failureMessage']='先结束这里的战斗，再整理战后留下的证词。'
        d['puzzleActions'][p]=main;d['rooms'][last].setdefault('actions',[]).append(p)
        for aid,text,target in [('content_record_people',ep[1],'people'),('content_record_route',ep[2],'route')]:
            d['puzzleActions'][aid]=dict(label=text,puzzleId=p,rooms=[last],fromStates=['considered'],nextState=target,message=ep[3]+' 你选择：'+text+'。这份记录不会额外发放物品或改变通关奖励。',setFlags={'discovery_'+cid+'_'+target:True})
        if cid in ['shuanglong_hui','silk_road_hugong']:
            d['contentStatus']='enriched';d['contentNote']='已扩写：保留本章分段战斗，以专属复盘选择区分护送/赴会中的职责，不套用通用开门题。'
        d['contentFeature']=dict(kind='epilogue',summary=ep[3],solution=[dict(roomId=last,actionId=p),dict(roomId=last,actionId='content_record_people')])
    # Give every item a specialist reading and an optional, nonconsumptive use.
    sources={}
    for cid,d in by_id.items():
        for iid in d['rewards'].get('items',{}):sources.setdefault(iid,cid)
    regional={
     'datang':['office_clerk','guild_steward','military_instructor','dd2_pool_poet','ca2_monk','heluo_blacksmith'],
     'fuyu':['storyteller','dock_foreman','qingyi_spy','jn_zither_player'],
     'posui':['ancient_scroll_keeper','academy_teacher','jn_calligrapher','r204_npc_3'],
     'bianhuang':['sb_centurion','tongguan_officer','sb_iron_smith','sb_scout_captain'],
     'luoshui':['yz_boat_girl','dm_pearl_diver','dd2_pool_poet','dm_jiaoren'],
     'yangzhou':['yz_yamen_clerk','yz_salt_merchant','yz_tea_master','yz_musician'],
     'bashu':['bs_tang_steward','bs_herbalist','bs_jiange_clerk','bs_wine_maker'],
     'saibei':['sb_horse_trader','sb_centurion','sb_iron_smith','sb_s2'],
     'jiangnan':['jn_pool_keeper','jn_zither_player','jn_calligrapher','jn_silk_merchant'],
     'dongming':['dm_old_captain','dm_jiaoren','dm_mirage_keeper','dm_pearl_diver']}
    special={'iron_pickaxe':'bs_jiange_clerk','old_map_fragment':'station_master_luo','bronze_box_rubbing':'academy_teacher','sealed_letter':'office_clerk','bronze_fragment':'heluo_blacksmith','herb':'bs_herbalist','reed_root':'herbalist_sun','dry_ration':'nan_market_peddler','smoke_powder':'black_market_broker','common_clothes':'east_market_keeper','common_shoes':'east_market_keeper','yg_baoku_pearl':'guild_steward'}
    for index,(iid,item) in enumerate(items.items()):
        if iid in RELIC_NAMES:item['name']=RELIC_NAMES[iid]
        cid=sources.get(iid)
        sid=item.get('effects',{}).get('learnSkill')
        if sid:cid=skills[sid].get('sourceDungeon') or cid
        pool=regional.get((cid or '').split('_')[0],['heluo_blacksmith','weapon_school_master','academy_teacher','herbalist_sun'])
        actor=special.get(iid) or (skills.get(sid,{}).get('teacherId') if sid else None) or pool[index%len(pool)]
        if actor not in people:actor='ancient_scroll_keeper'
        if not cid:cid=people[actor]['chapter']
        support=cid if cid in DEPTH else next((x for x in DEPTH if x.split('_')[0]==cid.split('_')[0]),'datang_yunhe')
        p=people[actor]; chapter=by_id[cid]
        context=DEPTH[cid]['end'] if cid in DEPTH else EPILOGUES[cid][3]
        question=DEPTH[cid]['a'] if cid in DEPTH else EPILOGUES[cid][0]
        reading=original_items[iid].get('description','')+' '+p['testimony']+' '+npcs[actor]['name']+'建议把'+item['name']+'与《'+chapter['name']+'》的这个问题互证：'+question+' 这是一条调查线索，最后的判断仍须亲历现场。'
        item['contentUse']=dict(npcId=actor,chapterId=cid,aidDungeon=support,reading=reading)
        item['description']=original_items[iid].get('description','')+'【器物线索】可向'+npcs[actor]['name']+'出示，核对《'+chapter['name']+'》的来历；不消耗，穿戴中也可出示。'
        flag='discovery_item_'+iid
        dialogs[npcs[actor]['dialogueId']]['options'].append(dict(id='content_show_'+iid,text='出示'+item['name']+'：核对其独有来历',response=reading,explicit=True,requires={'presentItems':{iid:1}},setFlags={flag:True}))
        d=by_id[support];rid=list(d['rooms'])[1]
        aid='content_aid_'+iid
        d['puzzleActions'][aid]=dict(label='以'+item['name']+'校验前段痕迹（不消耗）',puzzleId='content_trace',rooms=[rid],fromStates=['unseen'],nextState='known',message='你把'+npcs[actor]['name']+'辨明的器物来历与现场互证。'+DEPTH[support]['a'],requiredFlags={flag:True},presentItems={iid:1})
        d['rooms'][rid].setdefault('actions',[]).append(aid)
    # Retired books can be copied only after their original teacher taught the
    # art, preserving all martial gates. No new manuals or skills are created.
    for sid,s in skills.items():
        actor=s.get('teacherId');mid=s.get('manualId')
        if actor and mid:
            dialogs[npcs[actor]['dialogueId']]['options'].append(dict(id='content_copy_'+sid,text='誊录已学的'+s['name']+'，留作比对',response='老师只在你已正式入门后允许誊录；这份秘本不替代任何修炼或传承门槛。',explicit=True,requires={'skills':{sid:1}},giveItems={mid:1}))
            items[mid]['source']='原指定老师入门后可誊录一次；旧档持有继续兼容。'
    # Existing retired blade gains a deliberate reforge route, not a free gift.
    raw={'gongbian_mingzhu':1}
    while sum(items[k]['value']*v for k,v in raw.items())<=items['gongbian_jian']['value']:raw['bronze_fragment']=raw.get('bronze_fragment',0)+1
    dialogs[npcs['heluo_blacksmith']['dialogueId']]['options'].append(dict(id='content_reforge_old_palace',text='以宫变夜明珠改铸宫变密剑（交出凭证，单次）',response='钟铁衣依夜明珠的宫印重铸旧式密剑。你交出明确列出的材料，换回的是另一条可辨认的旧线索。',explicit=True,requires={'items':raw},takeItems=raw,giveItems={'gongbian_jian':1}))
    items['gongbian_jian']['source']='钟铁衣以宫变夜明珠按旧制改铸，单次明确选择；旧档兼容。'
    # Persist and emit a human-readable map, all from the same source of truth.
    write(RES/'items.json',items);write(RES/'npcs.json',npcs);write(RES/'dialogues.json',dialogs)
    for path,w in worlds.items():write(path,w)
    for path,d in chapters.items():write(path,d)
    testdata=ROOT/'MudGame-server/internal/gamedata/testdata';testdata.mkdir(exist_ok=True)
    write(testdata/'existing_content_baseline.json',baseline)
    lines=['# 既有资源用途与差异化覆盖','',f'基线{BASE}；实体ID保持不变。下列内容对应实际JSON与对话/机关，不是待做清单。','', '## 人物职责（313个既有NPC）','', '| ID / 人物 | 实际出场 | 专长与不可互代的信息 |','|---|---|---|']
    for nid,n in npcs.items():lines.append('| `'+nid+'` '+n['name']+' | '+'；'.join('/'.join(x[:2]) for x in places[nid])+' | '+people[nid]['specialty']+'：'+people[nid]['testimony']+' |')
    lines+=['','## 器物用途（135个既有物品）','','每次出示须真实持有（含合法穿戴），不消耗；只补在地线索，不替代通关、武学门槛或奖励。','', '| 物品 / ID | 专属解读者 | 互证章节 | 可辅助的现场 |','|---|---|---|---|']
    for iid,item in items.items():
        u=item['contentUse'];lines.append(f'| {item["name"]} / `{iid}` | {npcs[u["npcId"]]["name"]} | {by_id[u["chapterId"]]["name"]} | {by_id[u["aidDungeon"]]["name"]}前段痕迹 |')
    lines+=['','## 章节玩法与复盘（78关）','','| 章节 | 交互类型 | 判断/复盘的具体问题 |','|---|---|---|']
    for d in sorted(by_id.values(),key=lambda d:(d['order'],d['id'])):lines.append(f'| {d["name"]} / `{d["id"]}` | {d["contentFeature"]["kind"]} | {d["contentFeature"]["summary"]} |')
    lines+=['','## 八城互证事件与互斥结论','','这些是既有人物的对话延续，不增加任务定义。选定结论后不能在同一角色上再选相反结论，后续可在见闻中回看。','','| 办理者 / 事件 | 原始凭证（不消耗） | 必须分别询问的见证人 | 两种结论 | 追问后续 |','|---|---|---|---|---|']
    for actor,iid,witnesses,topic,left,right,_,_,aftermath in cases:lines.append(f'| {npcs[actor]["name"]}：{topic} | {items[iid]["name"]} | '+ '、'.join(npcs[n]['name'] for n in witnesses)+f' | {left} / {right} | {npcs[aftermath]["name"]} |')
    (ROOT/'docs/content/existing_resource_roles.md').write_text('\n'.join(lines)+'\n',encoding='utf-8')
    print(f'Enriched {len(DEPTH)} gates + {len(EPILOGUES)} distinct epilogues; {len(npcs)} NPCs placed, {len(items)} item uses, {len(public_rooms)} investigated public rooms; frozen entity sets/rewards preserved.')

if __name__=='__main__':main()
