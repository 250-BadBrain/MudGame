<template>
  <div class="shop-panel">
    <div class="shop-header">
      <h3>{{ title }}</h3>
      <button @click="$emit('close')" class="close-btn">×</button>
    </div>
    <div class="shop-content">
      <div class="money-display" v-if="showMoney">
        {{ formatMoney(playerMoney) }}
      </div>
      <div class="shop-items-list">
        <div v-for="item in items" :key="item.id" class="shop-item" :class="{ 'selected': selectedItem?.id === item.id }" @click="$emit('selectItem', item)">
          <div class="shop-item-info-row">
            <span class="item-name" :style="{ color: getItemColor(item.rarity) }">{{ item.name }}</span>
          </div>
          <div class="shop-item-price-row">
            <span class="item-price">{{ formatPrice(item.value) }}</span>
            <span v-if="item.limit !== undefined && item.limit === 0" class="item-sold-out-tag"> 已售罄</span>
            <span v-else-if="item.limit" class="item-limit-tag"> 库存:{{ item.limit }}</span>
            <span v-else class="item-stock-tag"> 库存充足</span>
          </div>
        </div>
      </div>
      <div class="buy-controls" v-if="selectedItem">
        <div class="selected-item-info">
          <span class="selected-item-name" :style="{ color: getItemColor(selectedItem.rarity) }">{{ selectedItem.name }}</span>
          <span class="selected-item-price">{{ formatPrice(selectedItem.value) }}</span>
        </div>
        <div class="amount-selector" v-if="selectedItem.stackable">
          <button @click="$emit('changeAmount', 'min')" :disabled="buyAmount <= 1" class="small-btn">最少</button>
          <button @click="$emit('changeAmount', -10)" :disabled="buyAmount <= 10" class="small-btn">-10</button>
          <input type="number" :value="buyAmount" min="1" readonly />
          <button @click="$emit('changeAmount', 10)" :disabled="isMaxReached" class="small-btn">+10</button>
          <button @click="$emit('changeAmount', 'max')" :disabled="isMaxReached" class="small-btn">最多</button>
        </div>
        <div class="stock-info">
          <span v-if="selectedItem.limit">库存: {{ selectedItem.limit }}{{ selectedItem.unit }}</span>
          <span v-else>库存充足</span>
        </div>
        <button @click="$emit('buy')" class="buy-btn">{{ buyButtonText }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: String,
  items: Array,
  selectedItem: Object,
  buyAmount: Number,
  isMaxReached: Boolean,
  playerMoney: { type: Number, default: 0 },
  showMoney: { type: Boolean, default: true },
  buyButtonText: { type: String, default: '购买' }
})

defineEmits(['close', 'selectItem', 'changeAmount', 'buy'])

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
</script>

<style scoped>
.shop-panel { height: 100%; display: flex; flex-direction: column; }
.shop-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.shop-header h3 { margin: 0; border: none; padding: 0; font-size: 14px; }
.close-btn { background: none; border: none; color: #e74c3c; font-size: 20px; cursor: pointer; padding: 0 5px; }
.shop-content { flex: 1; overflow-y: auto; display: flex; flex-direction: column; }
.money-display { text-align: right; color: #f1c40f; margin-bottom: 8px; font-weight: bold; }
.shop-items-list { flex: 1; overflow-y: auto; }
.shop-item { padding: 8px; cursor: pointer; border-bottom: 1px solid #333; transition: background 0.2s; }
.shop-item:hover { background: rgba(255,255,255,0.05); }
.shop-item.selected { background: rgba(241,196,15,0.15); border-left: 3px solid #f1c40f; }
.shop-item-info-row { display: flex; align-items: center; gap: 8px; }
.item-name { font-weight: bold; }
.shop-item-price-row { display: flex; align-items: center; gap: 8px; margin-top: 4px; font-size: 12px; }
.item-price { color: #f1c40f; }
.item-sold-out-tag { color: #e74c3c; font-weight: bold; }
.item-limit-tag { color: #e67e22; }
.item-stock-tag { color: #2ecc71; }
.buy-controls { margin-top: 12px; padding-top: 12px; border-top: 1px solid #444; }
.selected-item-info { display: flex; justify-content: space-between; margin-bottom: 8px; }
.selected-item-name { font-weight: bold; }
.selected-item-price { color: #f1c40f; }
.amount-selector { display: flex; align-items: center; gap: 4px; margin-bottom: 8px; }
.amount-selector input { width: 50px; text-align: center; background: #222; color: #ecf0f1; border: 1px solid #555; padding: 4px; }
.amount-selector button { padding: 4px 8px; font-size: 12px; }
.small-btn { background: #444; color: #ecf0f1; border: 1px solid #666; cursor: pointer; border-radius: 2px; padding: 4px 8px; font-size: 12px; font-family: inherit; }
.small-btn:hover:not(:disabled) { background: #555; }
.small-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.stock-info { margin-bottom: 8px; font-size: 12px; color: #888; }
.buy-btn { width: 100%; padding: 8px; background: #f1c40f; color: #111; border: none; cursor: pointer; font-weight: bold; font-family: inherit; border-radius: 4px; }
.buy-btn:hover { filter: brightness(1.2); }
</style>
