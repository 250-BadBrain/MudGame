<template>
  <div class="room-exits-grid">
    <h3>出口</h3>

    <div class="exits-map-grid">
      <div class="grid-cell">
        <template v-if="room?.exits?.northwest">
          <button @click="$emit('go', 'northwest')">{{ room.exits.northwest.targetRoomName }}</button>
        </template>
      </div>
      <div class="grid-cell">
        <template v-if="room?.exits?.north">
          <button @click="$emit('go', 'north')">{{ room.exits.north.targetRoomName }}</button>
        </template>
      </div>
      <div class="grid-cell">
        <template v-if="room?.exits?.northeast">
          <button @click="$emit('go', 'northeast')">{{ room.exits.northeast.targetRoomName }}</button>
        </template>
      </div>

      <div class="grid-cell">
        <template v-if="room?.exits?.west">
          <button @click="$emit('go', 'west')">{{ room.exits.west.targetRoomName }}</button>
        </template>
      </div>
      <div class="grid-cell current-room-center">
        <span class="room-abbr">{{ room?.name || '?' }}</span>
      </div>
      <div class="grid-cell">
        <template v-if="room?.exits?.east">
          <button @click="$emit('go', 'east')">{{ room.exits.east.targetRoomName }}</button>
        </template>
      </div>

      <div class="grid-cell">
        <template v-if="room?.exits?.southwest">
          <button @click="$emit('go', 'southwest')">{{ room.exits.southwest.targetRoomName }}</button>
        </template>
      </div>
      <div class="grid-cell">
        <template v-if="room?.exits?.south">
          <button @click="$emit('go', 'south')">{{ room.exits.south.targetRoomName }}</button>
        </template>
      </div>
      <div class="grid-cell">
        <template v-if="room?.exits?.southeast">
          <button @click="$emit('go', 'southeast')">{{ room.exits.southeast.targetRoomName }}</button>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  room: { type: Object, default: null }
})

defineEmits(['go'])
</script>

<style scoped>
.room-exits-grid {
    background-color: var(--color-background-secondary);
    border: 1px solid var(--color-border-subtle);
    padding: 8px;
    border-radius: 8px;
    flex: 0 0 auto;
    margin-top: auto;
    overflow: hidden;
}

.room-exits-grid h3 {
    margin: 0 0 6px;
    padding-bottom: 4px;
    font-size: 0.95em;
}

/* Exits Map Grid 样式 */
.exits-map-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    grid-template-rows: repeat(3, 1fr);

    gap: 3px;
    border: none;

    width: 100%;
    max-width: none;
    margin: 0 auto;
}

.grid-cell {
    min-height: 28px;

    display: flex;
    justify-content: center;
    align-items: center;
    padding: 0;
    box-sizing: border-box;
    border: none;
    background-color: transparent;
}

.grid-cell button {
    width: 100%;
    height: 100%;

    padding: 2px 2px;
    min-height: 28px;

    cursor: pointer;
    white-space: normal;
    text-align: center;
    border: none;
    background-color: var(--color-background-dark, #111);
    color: var(--color-text-light, #fff);
    font-size: 0.78em;
    line-height: 1.1;

    border-radius: 5px;
    border: 1px solid var(--color-border-subtle);
}

.current-room-center {
    width: 95%;
    height: 95%;
    min-height: 28px;

    background-color: var(--color-background-dark);
    color: var(--color-accent-gold);
    border: 2px solid var(--color-accent-gold);
    border-radius: 5px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
}

.room-abbr {
    font-size: 0.78em;
    word-break: break-word;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    max-width: 80%;
}
</style>
