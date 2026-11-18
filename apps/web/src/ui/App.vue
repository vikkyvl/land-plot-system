<script setup lang="ts">
import { onMounted } from 'vue'

import AppHeader from './components/AppHeader.vue'
import { useSessionStore } from './stores/session'

const session = useSessionStore()

onMounted(() => session.refresh())
</script>

<template>
  <AppHeader />
  <RouterView />
</template>
