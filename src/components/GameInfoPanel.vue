<template>
  <div v-if="visible" class="info-panel">
    <div class="panel-header">
      <h3>{{ title }}</h3>
      <button @click="$emit('close')" class="close-btn">×</button>
    </div>
    <div class="panel-content">
      <div v-if="currentPanel === 'backpack'" class="backpack-panel-container">
        <div class="money-header">{{ formatMoney(playerMoney) }}</div>
        <div class="backpack-list">
          <ul v-if="backpackItems && backpackItems.length > 0">
            <li v-for="item in backpackItems" :key="item.uniqueId || item.id" class="item-entry"
              :class="{ 'selected': (selectedBackpackItem?.uniqueId || selectedBackpackItem?.id) === (item.uniqueId || item.id) }"
              @click="$emit('selectBackpackItem', item)">
              <div class="item-row">
                <div class="item-row-content">
                  <span class="item-name" :style="{ color: getItemColor(item.rarity) }">{{ item.name }}</span>
                  <span v-if="item.stackable && item.count > 1" class="item-count">{{ item.count }}{{ item.unit }}</span>
                </div>
              </div>
              <div v-if="(selectedBackpackItem?.uniqueId || selectedBackpackItem?.id) === (item.uniqueId || item.id)" class="item-actions">
                <button @click.stop="$emit('viewItem', item)" class="action-btn">查看</button>
                <button @click.stop="$emit('discardItem', item)" class="action-btn discard-btn">丢弃</button>
                <button v-if="item.equipmentSlot" @click.stop="$emit('equipItem', item)" class="action-btn">装备</button>
                <button v-if="item.sellable && hasShopNpc" @click.stop="$emit('sellItem', item)" class="action-btn">出售</button>
              </div>
            </li>
          </ul>
          <p v-else class="empty-msg">背包里空空如也。</p>
        </div>
        <div class="backpack-bottom-area">
          <div v-if="viewingItem" class="item-details-view">
            <div class="detail-header">
              <span class="detail-name" :style="{ color: getItemColor(viewingItem.rarity) }">{{ viewingItem.name }}</span>
              <span class="detail-sell-price" v-if="viewingItem.sellable">{{ formatPrice(Math.floor(viewingItem.value / 2)) }}/{{ viewingItem.unit || '个' }}</span>
            </div>
            <div class="detail-desc">{{ viewingItem.description }}</div>
            <div v-if="viewingItem.equipmentSlot" class="detail-slot">部位: {{ equipmentSlotLabels[viewingItem.equipmentSlot] || viewingItem.equipmentSlot }}</div>
          </div>
          <div v-else-if="discardingItem" class="discard-prompt">
            <div class="discard-info">丢弃 {{ discardingItem.name }} ?</div>
            <div class="amount-selector" v-if="discardShowAmount">
              <button @click="$emit('changeDiscardAmount', -1)" :disabled="discardAmount <= 1" class="small-btn">-</button>
              <input type="number" :value="discardAmount" readonly />
              <button @click="$emit('changeDiscardAmount', 1)" :disabled="discardAmount >= discardMax" class="small-btn">+</button>
            </div>
            <div class="discard-actions">
              <button @click="$emit('confirmDiscard')" class="confirm-btn">确认</button>
              <button @click="$emit('cancelDiscard')" class="cancel-btn">取消</button>
            </div>
          </div>
          <div v-else-if="sellingItem" class="discard-prompt">
            <div class="discard-info">出售 {{ sellingItem.name }}</div>
            <div class="sell-price-info">单价: {{ formatPrice(Math.floor(sellingItem.value / 2)) }}</div>
            <div class="amount-selector" v-if="sellingItem.stackable && sellingItem.count > 1">
              <button @click="$emit('changeSellAmount', 'min')" :disabled="sellAmount <= 1" class="small-btn">最少</button>
              <button @click="$emit('changeSellAmount', -10)" :disabled="sellAmount <= 10" class="small-btn">-10</button>
              <input type="number" :value="sellAmount" readonly />
              <button @click="$emit('changeSellAmount', 10)" :disabled="sellAmount >= sellMax" class="small-btn">+10</button>
              <button @click="$emit('changeSellAmount', 'max')" :disabled="sellAmount >= sellMax" class="small-btn">最多</button>
            </div>
            <div class="total-price-info">总价: {{ formatPrice(Math.floor(sellingItem.value / 2) * sellAmount) }}</div>
            <div class="discard-actions">
              <button @click="$emit('confirmSell')" class="confirm-btn">确认出售</button>
              <button @click="$emit('cancelSell')" class="cancel-btn">取消</button>
            </div>
          </div>
        </div>
      </div>
      <div v-if="currentPanel === 'skills'" class="backpack-panel-container">
        <div class="skill-tabs">
          <span :class="{ active: skillTab === 'basic' }" @click="$emit('changeSkillTab', 'basic')">基础武功</span>
          <span class="tab-separator">|</span>
          <span :class="{ active: skillTab === 'special' }" @click="$emit('changeSkillTab', 'special')">特殊武功</span>
        </div>
        <div class="backpack-list">
          <ul v-if="filteredSkills && filteredSkills.length > 0">
            <li v-for="skill in filteredSkills" :key="skill.id" class="item-entry"
              :class="{ selected: selectedSkill && selectedSkill.id === skill.id }"
              @click="$emit('selectSkill', skill)">
              <div class="item-row">
                <div class="item-row-content">
                  <span class="item-name" :style="{ color: getItemColor(skill.rarity) }">{{ skill.name }}</span>
                  <span class="item-count">Lv.{{ skill.level }}</span>
                </div>
              </div>
              <div v-if="selectedSkill && selectedSkill.id === skill.id" class="item-actions">
                <button @click.stop="$emit('viewSkill', skill)" class="action-btn">查看</button>
              </div>
            </li>
          </ul>
          <p v-else class="empty-msg">没有相关技能。</p>
        </div>
        <div class="backpack-bottom-area" v-if="viewingSkill">
          <div class="item-details-view">
            <div class="detail-header">
              <span class="detail-name" :style="{ color: getItemColor(viewingSkill.rarity) }">{{ viewingSkill.name }}</span>
              <span class="detail-sell-price">Lv.{{ viewingSkill.level }}</span>
            </div>
            <div class="detail-desc">{{ viewingSkill.description }}</div>
            <div class="detail-bonuses" v-if="getSkillBonusText(viewingSkill).length > 0">
              <div class="bonus-title">加成效果:</div>
              <div v-for="(bonus, index) in getSkillBonusText(viewingSkill)" :key="index" class="bonus-item">{{ bonus }}</div>
            </div>
          </div>
        </div>
      </div>
      <div v-if="currentPanel === 'equipment'" class="equipment-panel-container">
        <div class="equipment-slots">
          <div v-for="(slotName, slotKey) in equipmentSlotLabels" :key="slotKey" class="equipment-slot-wrapper">
            <div class="equipment-slot"
              :class="{ 'selected': selectedEquipmentSlot === slotKey }"
              @click="$emit('selectEquipSlot', slotKey)">
              <span class="slot-label">{{ slotName }}</span>
              <div class="slot-content">
                <span v-if="equipment[slotKey]" class="equipped-item" :style="{ color: getItemColor(equipment[slotKey].rarity) }">{{ equipment[slotKey].name }}</span>
                <span v-else class="empty-slot">空</span>
              </div>
            </div>
            <div v-if="selectedEquipmentSlot === slotKey && equipment[slotKey]" class="slot-actions">
              <button @click="$emit('unequipSlot', slotKey)" class="action-btn unequip-btn">脱下</button>
            </div>
          </div>
        </div>
        <div class="backpack-bottom-area" v-if="selectedEquipmentSlot && equipment[selectedEquipmentSlot]">
          <div class="item-details-view">
            <div class="detail-header">
              <span class="detail-name" :style="{ color: getItemColor(equipment[selectedEquipmentSlot].rarity) }">{{ equipment[selectedEquipmentSlot].name }}</span>
            </div>
            <div class="detail-desc">{{ equipment[selectedEquipmentSlot].description }}</div>
          </div>
        </div>
      </div>
      <div v-if="currentPanel === 'status'">
        <div v-if="statusAttributes" class="status-list">
          <div class="status-item"><span class="label">经验:</span> <span class="value">{{ statusAttributes.experience || 0 }}</span></div>
          <div class="status-item"><span class="label">潜能:</span> <span class="value">{{ statusAttributes.potential || 0 }}</span></div>
          <hr class="status-divider">
          <div class="status-item"><span class="label">气血:</span> <span class="value">{{ statusAttributes.currentHp }} / {{ statusAttributes.maxHp }}</span></div>
          <div class="status-item"><span class="label">内力:</span> <span class="value">{{ statusAttributes.currentMp }} / {{ statusAttributes.trainedMaxMp }}</span></div>
          <div class="status-item"><span class="label">内力上限:</span> <span class="value">{{ statusAttributes.maxMp }}</span></div>
          <div class="status-item"><span class="label">心神:</span> <span class="value">{{ statusAttributes.spirit }} / {{ statusAttributes.maxSpirit }}</span></div>
          <hr class="status-divider">
          <div class="status-grid">
            <div class="status-item"><span class="label">臂力:</span> <span class="value">{{ statusAttributes.strength }} ({{ statusAttributes.totalPostStrength || statusAttributes.postStrength || 0 }})</span></div>
            <div class="status-item"><span class="label">根骨:</span> <span class="value">{{ statusAttributes.constitution }} ({{ statusAttributes.totalPostConstitution || statusAttributes.postConstitution || 0 }})</span></div>
            <div class="status-item"><span class="label">身法:</span> <span class="value">{{ statusAttributes.agility }} ({{ statusAttributes.totalPostAgility || statusAttributes.postAgility || 0 }})</span></div>
            <div class="status-item"><span class="label">悟性:</span> <span class="value">{{ statusAttributes.perception }} ({{ statusAttributes.totalPostPerception || statusAttributes.postPerception || 0 }})</span></div>
            <div class="status-item"><span class="label">心志:</span> <span class="value">{{ statusAttributes.willpower }} ({{ statusAttributes.totalPostWillpower || statusAttributes.postWillpower || 0 }})</span></div>
            <div class="status-item"><span class="label">魅力:</span> <span class="value">{{ statusAttributes.charm || '?' }}</span></div>
            <div class="status-item"><span class="label">福禄:</span> <span class="value">{{ statusAttributes.luck || '?' }}</span></div>
          </div>
          <hr class="status-divider">
          <div class="status-grid">
            <div class="status-item"><span class="label">攻击:</span> <span class="value">{{ statusAttributes.attack }}</span></div>
            <div class="status-item"><span class="label">防御:</span> <span class="value">{{ statusAttributes.defense }}</span></div>
            <div class="status-item"><span class="label">命中:</span> <span class="value">{{ statusAttributes.hitRate || 100 }}</span></div>
            <div class="status-item"><span class="label">躲闪:</span> <span class="value">{{ statusAttributes.dodge || 100 }}</span></div>
            <div class="status-item"><span class="label">招架:</span> <span class="value">{{ statusAttributes.parry || 100 }}</span></div>
            <div class="status-item"><span class="label">暴击:</span> <span class="value">{{ statusAttributes.critRate || 1 }}%</span></div>
            <div class="status-item full-width"><span class="label">攻速:</span> <span class="value">{{ statusAttributes.attackSpeed || 4 }}</span></div>
          </div>
        </div>
      </div>
      <div v-if="currentPanel === 'maps'" class="maps-panel-container">
        <div class="map-list-area">
          <ul v-if="worldsList && worldsList.length > 0" class="map-list">
            <li v-for="world in worldsList" :key="world.id" class="map-item"
              :class="{ 'selected': selectedWorldId === world.id }"
              @click="$emit('selectWorld', world.id)">
              <span class="map-name">{{ world.name }}</span>
            </li>
          </ul>
          <p v-else class="empty-msg">暂无可用地图。</p>
        </div>
        <div class="map-description-area">
          <div class="map-desc-text">{{ selectedWorld ? selectedWorld.description : '请选择一个世界查看详情。' }}</div>
          <button @click="$emit('enterWorld')" class="enter-world-btn" :disabled="!selectedWorld">进入世界</button>
        </div>
      </div>
      <div v-if="currentPanel === 'dungeons'" class="maps-panel-container">
        <div class="map-list-area">
          <ul v-if="dungeonsList && dungeonsList.length > 0" class="map-list">
            <li v-for="dungeon in dungeonsList" :key="dungeon.id" class="map-item"
              :class="{ 'selected': selectedDungeonId === dungeon.id }"
              @click="$emit('selectDungeon', dungeon.id)">
              <div class="map-item-line">
                <span class="map-name">{{ dungeon.name }}</span>
              </div>
            </li>
          </ul>
          <p v-else class="empty-msg">暂无可用副本。</p>
        </div>
        <div class="map-description-area">
          <div class="map-desc-text">{{ selectedDungeon ? selectedDungeon.description : '请选择一个副本查看详情。' }}</div>
          <div v-if="selectedDungeon" class="map-sub-line">满额奖励：{{ selectedDungeon.rewardBase || 0 }}经验 {{ selectedDungeon.rewardBase || 0 }}潜能</div>
          <div v-if="isInDungeon && dungeonRoomId === selectedDungeon?.id" class="map-sub-line">当前进度：{{ dungeonProgress ?? 0 }}%</div>
          <button @click="$emit('enterDungeon')" class="enter-world-btn" :disabled="!selectedDungeon || isInDungeon">进入副本</button>
        </div>
      </div>
      <GameQuestPanel v-if="currentPanel === 'quests'" :quests="questsList" />
      <GameFactionPanel v-if="currentPanel === 'factions'" :factions="factionsList" />
    </div>
  </div>
</template>

<script setup>
import GameFactionPanel from './GameFactionPanel.vue'
import GameQuestPanel from './GameQuestPanel.vue'

const props = defineProps({
  visible: Boolean,
  title: String,
  currentPanel: String,
  backpackItems: { type: Array, default: () => [] },
  selectedBackpackItem: Object,
  viewingItem: Object,
  discardingItem: Object,
  discardAmount: { type: Number, default: 1 },
  discardMax: { type: Number, default: 1 },
  discardShowAmount: Boolean,
  sellingItem: Object,
  sellAmount: { type: Number, default: 1 },
  sellMax: { type: Number, default: 1 },
  playerMoney: { type: Number, default: 0 },
  hasShopNpc: Boolean,
  skillTab: { type: String, default: 'basic' },
  filteredSkills: { type: Array, default: () => [] },
  selectedSkill: Object,
  viewingSkill: Object,
  equipmentSlotLabels: { type: Object, default: () => ({}) },
  equipment: { type: Object, default: () => ({}) },
  selectedEquipmentSlot: String,
  statusAttributes: Object,
  worldsList: { type: Array, default: () => [] },
  selectedWorldId: String,
  selectedWorld: Object,
  dungeonsList: { type: Array, default: () => [] },
  selectedDungeonId: String,
  selectedDungeon: Object,
  isInDungeon: Boolean,
  dungeonRoomId: String,
  dungeonProgress: { type: Number, default: 0 },
  questsList: { type: Array, default: () => [] },
  factionsList: { type: Array, default: () => [] }
})

defineEmits(['close', 'selectBackpackItem', 'viewItem', 'discardItem', 'equipItem', 'sellItem',
  'changeDiscardAmount', 'confirmDiscard', 'cancelDiscard',
  'changeSellAmount', 'confirmSell', 'cancelSell',
  'changeSkillTab', 'selectSkill', 'viewSkill',
  'selectEquipSlot', 'unequipSlot',
  'selectWorld', 'enterWorld',
  'selectDungeon', 'enterDungeon'])

const getItemColor = (rarity) => {
  const colors = { 1: '#888', 2: '#2ecc71', 3: '#3498db', 4: '#f1c40f', 5: '#8e44ad', 6: '#e67e22', 7: '#e74c3c' }
  return colors[rarity] || '#888'
}

const formatMoney = (money) => {
  if (!money && money !== 0) return ''
  const gold = Math.floor(money / 10000)
  const silver = Math.floor((money % 10000) / 100)
  const copper = money % 100
  let parts = []
  if (gold > 0) parts.push(gold + '两黄金')
  if (silver > 0) parts.push(silver + '两银子')
  if (copper > 0 || parts.length === 0) parts.push(copper + '枚铜板')
  return parts.join(' ')
}

const formatPrice = (value) => {
  if (!value && value !== 0) return ''
  return formatMoney(value)
}

const getSkillBonusText = (skill) => {
  if (Array.isArray(skill?.bonusText)) return skill.bonusText
  if (!skill || !skill.bonuses) return []
  const bonusMap = { 'strength': '臂力', 'constitution': '根骨', 'agility': '身法', 'perception': '悟性', 'willpower': '心志', 'charm': '魅力', 'luck': '福缘', 'attack': '攻击', 'defense': '防御', 'maxHp': '气血上限', 'maxMp': '内力上限', 'spirit': '心神', 'maxSpirit': '心神上限', 'hitRate': '命中', 'dodge': '躲闪', 'parry': '招架', 'critRate': '暴击率' }
  return Object.entries(skill.bonuses).map(([key, val]) => (bonusMap[key] || key) + ' +' + val)
}
</script>

<style scoped>
.info-panel { background: rgba(20,20,20,0.95); border: 1px solid #333; border-radius: 6px; padding: 10px; margin-bottom: 8px; overflow: hidden; min-width: 0; }
.panel-header { display: flex; justify-content: space-between; align-items: center; gap: 8px; margin-bottom: 8px; min-width: 0; }
.panel-header h3 { margin: 0; font-size: 14px; color: #f1c40f; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.close-btn { background: none; border: none; color: #e74c3c; font-size: 18px; cursor: pointer; padding: 0 4px; }
.panel-content { max-height: 50vh; overflow-y: auto; overflow-x: hidden; min-width: 0; }
.money-header { text-align: right; color: #f1c40f; margin-bottom: 6px; font-weight: bold; font-size: 13px; }
.backpack-list ul { list-style: none; padding: 0; margin: 0; }
.item-entry { padding: 6px 6px; cursor: pointer; border-bottom: 1px solid #2a2a2a; }
.item-entry.selected { background: rgba(241,196,15,0.1); border-left: 3px solid #f1c40f; }
.item-row-content { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; min-width: 0; }
.item-name { font-weight: bold; font-size: 12px; overflow-wrap: anywhere; }
.item-count { color: #888; font-size: 11px; background: #333; padding: 1px 6px; border-radius: 8px; }
.item-actions { display: flex; gap: 4px; margin-top: 4px; flex-wrap: wrap; }
.action-btn { padding: 4px 8px; background: #444; color: #ecf0f1; border: 1px solid #666; cursor: pointer; font-size: 11px; border-radius: 3px; font-family: inherit; }
.action-btn:hover { background: #555; }
.discard-btn { color: #e74c3c; }
.empty-msg { color: #666; text-align: center; padding: 15px; font-style: italic; font-size: 12px; }
.backpack-bottom-area { margin-top: 8px; padding: 8px; background: rgba(0,0,0,0.3); border-radius: 4px; }
.item-details-view { }
.detail-header { display: flex; justify-content: space-between; gap: 8px; margin-bottom: 4px; flex-wrap: wrap; }
.detail-name { font-weight: bold; font-size: 13px; overflow-wrap: anywhere; }
.detail-sell-price { color: #f1c40f; font-size: 12px; }
.detail-desc { font-size: 11px; color: #b0b0b0; margin-bottom: 4px; }
.detail-slot { font-size: 11px; color: #888; }
.detail-bonuses { margin-top: 6px; }
.bonus-title { font-size: 11px; color: #888; margin-bottom: 2px; }
.bonus-item { font-size: 11px; color: #2ecc71; padding-left: 8px; }
.discard-prompt { }
.discard-info { font-size: 12px; margin-bottom: 6px; }
.sell-price-info { font-size: 11px; color: #f1c40f; margin-bottom: 4px; }
.total-price-info { font-size: 11px; color: #f1c40f; margin: 4px 0; }
.amount-selector { display: flex; align-items: center; justify-content: center; gap: 4px; margin-bottom: 6px; flex-wrap: wrap; }
.amount-selector input { width: 40px; text-align: center; background: #222; color: #ecf0f1; border: 1px solid #555; padding: 3px; }
.small-btn { padding: 3px 6px; background: #444; color: #ecf0f1; border: 1px solid #666; cursor: pointer; font-size: 11px; border-radius: 2px; font-family: inherit; }
.small-btn:hover:not(:disabled) { background: #555; }
.small-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.discard-actions { display: flex; gap: 6px; flex-wrap: wrap; justify-content: center; }
.confirm-btn { padding: 5px 12px; background: #e74c3c; color: #fff; border: none; cursor: pointer; font-size: 12px; border-radius: 3px; font-family: inherit; }
.cancel-btn { padding: 5px 12px; background: #555; color: #ecf0f1; border: none; cursor: pointer; font-size: 12px; border-radius: 3px; font-family: inherit; }
.skill-tabs { display: flex; gap: 4px; margin-bottom: 8px; font-size: 12px; }
.skill-tabs span { cursor: pointer; color: #888; padding: 2px 4px; }
.skill-tabs span.active { color: #f1c40f; font-weight: bold; }
.tab-separator { color: #444; cursor: default !important; }
.equipment-slots { display: flex; flex-direction: column; gap: 4px; }
.equipment-slot-wrapper { display: flex; align-items: center; gap: 8px; min-width: 0; }
.equipment-slot { flex: 1; display: flex; align-items: center; gap: 8px; padding: 6px 8px; border-bottom: 1px solid #2a2a2a; cursor: pointer; }
.equipment-slot.selected { background: rgba(241,196,15,0.1); }
.slot-label { min-width: 50px; color: #888; font-size: 12px; }
.slot-content { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; }
.equipped-item { font-weight: bold; font-size: 12px; }
.empty-slot { color: #555; font-style: italic; font-size: 12px; }
.slot-actions { }
.unequip-btn { background: #e67e22; color: #fff; border: none; padding: 4px 8px; cursor: pointer; font-size: 11px; border-radius: 3px; font-family: inherit; }
.status-list { font-size: 12px; }
.status-item { display: flex; justify-content: space-between; padding: 3px 6px; }
.status-item .label { color: #888; }
.status-item .value { color: #ecf0f1; }
.status-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 2px; }
.status-item.full-width { grid-column: 1 / -1; }
.status-divider { border: none; border-top: 1px solid #333; margin: 6px 0; }
.maps-panel-container { display: flex; flex-direction: column; gap: 8px; }
.map-list { list-style: none; padding: 0; margin: 0; }
.map-list-area { max-height: 200px; overflow-y: auto; overflow-x: hidden; }
.map-item { padding: 6px 8px; cursor: pointer; border-bottom: 1px solid #2a2a2a; font-size: 12px; }
.map-item.selected { background: rgba(241,196,15,0.1); border-left: 3px solid #f1c40f; color: #f1c40f; }
.map-name { font-weight: bold; }
.map-description-area { padding: 8px; background: rgba(0,0,0,0.3); border-radius: 4px; font-size: 12px; }
.map-desc-text { color: #b0b0b0; margin-bottom: 8px; }
.map-sub-line { color: #888; font-size: 11px; margin-bottom: 4px; }
.enter-world-btn { width: 100%; padding: 6px; background: #3498db; color: #fff; border: none; cursor: pointer; border-radius: 4px; font-family: inherit; }
.enter-world-btn:disabled { background: #444; color: #888; cursor: not-allowed; }
</style>
