<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'

import AppHeader from './components/AppHeader.vue'
import BottomNav from './components/BottomNav.vue'
import OfflineBanner from './components/OfflineBanner.vue'
import ToastHost from './components/ToastHost.vue'
import { useSessionStore } from './stores/session'

const session = useSessionStore()
const { signedIn } = storeToRefs(session)
const route = useRoute()
const showBottomNav = computed(() => signedIn.value && route.meta.bottomNav !== false)

onMounted(() => session.refresh())
</script>

<template>
  <div
    class="app"
    :class="{ 'app--own-mobile-header': route.meta.ownMobileHeader, 'app--bottom-nav': showBottomNav }"
  >
    <OfflineBanner />
    <AppHeader class="app__header" />
    <RouterView />
    <BottomNav v-if="showBottomNav" />
    <ToastHost />
  </div>
</template>

<style scoped>
@media (max-width: 767.98px) {
  .app--own-mobile-header .app__header {
    display: none;
  }

  .app--bottom-nav {
    --bottom-nav-height: 73px;

    padding-bottom: calc(var(--bottom-nav-height) + env(safe-area-inset-bottom));
  }
}
</style>
