<template>
  <main class="home">
    <h1 class="home__title">
      Arqetra
    </h1>
  </main>
</template>

<style scoped>
.home {
  display: grid;
  min-height: 100dvh;
  place-items: center;
  padding: env(safe-area-inset-top) env(safe-area-inset-right) env(safe-area-inset-bottom) env(safe-area-inset-left);
}

.home__title {
  margin: 0;
  color: var(--color-primary);
  font-size: var(--fs-h1);
  font-weight: 700;
}
</style>
