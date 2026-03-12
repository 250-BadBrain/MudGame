<template>
  <div class="character-create-container">
    <h2>创建新角色</h2>

    <div class="form-group">
      <label>角色名称:</label>
      <input v-model="characterName" placeholder="请输入角色名称 (2-16字符)" />
    </div>

    <div class="attributes-section">
      <h3>先天属性 (总和必须为80)</h3>
      <div class="attribute-row">
        <label>臂力 (Strength): {{ strength }}</label>
        <input type="range" v-model.number="strength" min="15" max="30" @input="validateSum" />
      </div>
      <div class="attribute-row">
        <label>根骨 (Constitution): {{ constitution }}</label>
        <input type="range" v-model.number="constitution" min="15" max="30" @input="validateSum" />
      </div>
      <div class="attribute-row">
        <label>身法 (Agility): {{ agility }}</label>
        <input type="range" v-model.number="agility" min="15" max="30" @input="validateSum" />
      </div>
      <div class="attribute-row">
        <label>悟性 (Intelligence): {{ intelligence }}</label>
        <input type="range" v-model.number="intelligence" min="15" max="30" @input="validateSum" />
      </div>
      
      <div class="sum-display" :class="{ 'valid-sum': currentSum === 80, 'invalid-sum': currentSum !== 80 }">
        当前总和: {{ currentSum }} / 80
      </div>

      <button @click="randomizeAttributes" class="btn-random">随机分配</button>
    </div>

    <div class="actions">
      <button @click="goBack" class="btn-cancel">返回</button>
      <button @click="createCharacter" class="btn-create" :disabled="!isValid">创建角色</button>
    </div>

    <p v-if="error" class="error-msg">{{ error }}</p>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { usePlayerStore } from '@/stores/player'
import { initWebSocket, removeMessageHandler, sendMessage } from '../utils/ws.js'

const router = useRouter()
const playerStore = usePlayerStore()

const characterName = ref('')
const strength = ref(20)
const constitution = ref(20)
const agility = ref(20)
const intelligence = ref(20)
const error = ref('')
const loading = ref(false)

const currentSum = computed(() => strength.value + constitution.value + agility.value + intelligence.value)

const isValid = computed(() => {
  return characterName.value.length >= 2 && 
         characterName.value.length <= 16 && 
         currentSum.value === 80 &&
         !loading.value
})

const validateSum = () => {
  // Optional: Logic to auto-adjust sliders if needed, but for now just validation
}

const randomizeAttributes = () => {
  // Simple randomizer that ensures sum is 80 and each is 15-30
  let s, c, a, i;
  do {
    s = Math.floor(Math.random() * 16) + 15;
    c = Math.floor(Math.random() * 16) + 15;
    a = Math.floor(Math.random() * 16) + 15;
    i = 80 - s - c - a;
  } while (i < 15 || i > 30);
  
  strength.value = s;
  constitution.value = c;
  agility.value = a;
  intelligence.value = i;
}

const createCharacter = () => {
  if (!isValid.value) return;
  loading.value = true;
  error.value = '';

  const msg = {
    type: "system",
    subtype: "create_character",
    args: { 
      userId: playerStore.userId,
      characterName: characterName.value.trim(),
      strength: strength.value,
      constitution: constitution.value,
      agility: agility.value,
      intelligence: intelligence.value
    }
  }
  sendMessage(msg);
}

const goBack = () => {
  router.push({ name: 'CharacterSelect' });
}

const handleCreateCharacterMessage = (msg) => {
  if (msg.type === 'system' && msg.subtype === 'create_character') {
    loading.value = false;
    if (msg.flag) {
      router.push({ name: 'CharacterSelect' });
    } else {
      error.value = msg.results.error || '创建失败';
    }
  }
}

onMounted(() => {
  randomizeAttributes(); // Start with valid random values
  initWebSocket(handleCreateCharacterMessage);
})

onUnmounted(() => {
  removeMessageHandler(handleCreateCharacterMessage)
})
</script>

<style scoped>
.character-create-container {
  width: 500px;
  margin: 50px auto;
  padding: 30px;
  background-color: var(--color-background-dark);
  color: var(--color-text-light);
  border: 2px solid var(--color-accent-gold);
  border-radius: 8px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group input {
  width: 100%;
  padding: 8px;
  margin-top: 5px;
  background-color: #333;
  color: white;
  border: 1px solid #555;
}

.attribute-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.attribute-row input {
  width: 60%;
}

.sum-display {
  text-align: center;
  font-weight: bold;
  margin: 15px 0;
}

.valid-sum { color: #2ecc71; }
.invalid-sum { color: #e74c3c; }

.actions {
  display: flex;
  justify-content: space-between;
  margin-top: 20px;
}

.btn-random {
  width: 100%;
  padding: 8px;
  background-color: var(--color-accent-blue);
  color: white;
  border: none;
  cursor: pointer;
  margin-bottom: 10px;
}

.btn-create {
  background-color: var(--color-accent-gold);
  color: black;
  padding: 10px 20px;
  border: none;
  cursor: pointer;
}

.btn-create:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-cancel {
  background-color: #7f8c8d;
  color: white;
  padding: 10px 20px;
  border: none;
  cursor: pointer;
}

.error-msg {
  color: #e74c3c;
  margin-top: 10px;
  text-align: center;
}
</style>
