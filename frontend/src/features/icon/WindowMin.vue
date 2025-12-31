<script setup lang="ts">
const props = withDefaults(defineProps<{ size: number | string | undefined }>(), { size: 14 })
const show:number = 1
</script>

<template>
    <svg v-if="show === 1" :height="props.size" :width="props.size" fill="none" viewBox="0 0 48 48">
      <path
        d="M8 24L40 24"
        stroke="currentColor"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="4" />
    </svg>
  <svg
    v-if="show === 2"
    stroke="currentColor"
    fill="currentColor"
    stroke-width="0"
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 512 512"
    :height="props.size"
    :width="props.size">
    <path
      d="M32 416c-17.7 0-32 14.3-32 32s14.3 32 32 32H480c17.7 0 32-14.3 32-32s-14.3-32-32-32H32z"></path>
  </svg>
    <svg
      v-if="show === 3"
      stroke="currentColor"
      fill="currentColor"
      stroke-width="0"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 512 512"
      :width="props.size"
      :height="props.size">
      <path
        d="M480 480H32c-17.7 0-32-14.3-32-32s14.3-32 32-32h448c17.7 0 32 14.3 32 32s-14.3 32-32 32z"></path>
    </svg>
</template>

<style lang="scss" scoped></style>
