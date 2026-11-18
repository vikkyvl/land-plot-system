<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
const isPlots = () => route.path.startsWith('/plots') && !route.path.endsWith('/edit')
</script>

<template>
  <nav
    class="bottom-nav"
    aria-label="Основна навігація"
  >
    <RouterLink
      class="bottom-nav__item"
      to="/"
      :aria-current="route.path === '/' || route.path.endsWith('/edit') ? 'page' : undefined"
    >
      <svg
        width="22"
        height="22"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <path
          d="M4 18 L8 6 L19 8 L17 19 Z"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linejoin="round"
        />
      </svg>
      Розрахунок
    </RouterLink>
    <RouterLink
      class="bottom-nav__item"
      to="/plots"
      :aria-current="isPlots() ? 'page' : undefined"
    >
      <svg
        width="22"
        height="22"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <rect
          x="4"
          y="4"
          width="16"
          height="16"
          rx="3"
          fill="none"
          stroke="currentColor"
          stroke-width="2.2"
        />
        <path
          d="M8 9 H16 M8 13 H16 M8 17 H12"
          stroke="currentColor"
          stroke-width="2"
        />
      </svg>
      Ділянки
    </RouterLink>
  </nav>
</template>

<style scoped>
.bottom-nav {
  display: none;
}

@media (max-width: 767.98px) {
  .bottom-nav {
    position: fixed;
    z-index: 1500;
    right: 0;
    bottom: 0;
    left: 0;
    display: grid;
    grid-auto-columns: minmax(0, 1fr);
    grid-auto-flow: column;
    padding: 6px var(--space-2) calc(14px + env(safe-area-inset-bottom));
    border-top: 1px solid var(--color-border);
    background: var(--color-surface);
  }

  .bottom-nav__item {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 3px;
    min-height: var(--control-h-md);
    color: var(--color-ink-3);
    font-size: var(--fs-xs);
    font-weight: 600;
    text-decoration: none;
  }

  .bottom-nav__item[aria-current='page'] {
    color: var(--color-primary);
    font-weight: 800;
  }
}
</style>
