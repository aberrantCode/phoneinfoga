<template>
  <div v-if="results.length > 0" class="search-result-list">
    <div
      v-for="(result, index) in results"
      :key="`${idPrefix}-${index}`"
      class="search-result"
    >
      <div class="search-result-main">
        <a :href="result.url" target="_blank" rel="noopener">
          {{ result.title || result.url }}
        </a>
        <span v-if="showEngine && result.engine" class="text-muted small">
          {{ result.engine }}
        </span>
        <p v-if="result.content" class="text-muted small mb-0">
          {{ result.content }}
        </p>
      </div>
      <div v-if="showActions" class="search-result-actions">
        <b-button
          :id="`${idPrefix}-open-${index}`"
          :href="result.url"
          target="_blank"
          rel="noopener"
          variant="dark"
          size="sm"
          class="search-result-button mr-1 mb-1"
          aria-label="Open result in new tab"
        >
          <b-icon-box-arrow-up-right aria-hidden="true" />
        </b-button>
        <b-tooltip :target="`${idPrefix}-open-${index}`">
          Open result in new tab
        </b-tooltip>
        <b-button
          :id="`${idPrefix}-copy-${index}`"
          variant="outline-secondary"
          size="sm"
          class="search-result-button mb-1"
          aria-label="Copy result URL"
          @click="$emit('copy', result.url)"
        >
          <b-icon-link45deg aria-hidden="true" />
        </b-button>
        <b-tooltip :target="`${idPrefix}-copy-${index}`">
          Copy result URL
        </b-tooltip>
      </div>
    </div>
  </div>
  <span v-else-if="emptyText" class="text-muted">{{ emptyText }}</span>
</template>

<script lang="ts">
import { Component, Vue, Prop } from "vue-property-decorator";

export interface SearchResult {
  title?: string;
  url: string;
  content?: string;
  engine?: string;
}

@Component
export default class SearchResultList extends Vue {
  @Prop({ default: () => [] }) results!: SearchResult[];
  @Prop({ default: "result" }) idPrefix!: string;
  @Prop({ default: false }) showActions!: boolean;
  @Prop({ default: false }) showEngine!: boolean;
  @Prop({ default: "" }) emptyText!: string;
}
</script>

<style scoped>
.search-result-list {
  min-width: 18rem;
}

.search-result {
  align-items: flex-start;
  display: flex;
  gap: 1rem;
  justify-content: space-between;
}

.search-result + .search-result {
  border-top: 1px solid #e9ecef;
  margin-top: 0.75rem;
  padding-top: 0.75rem;
}

.search-result-main {
  min-width: 0;
  word-break: break-word;
}

.search-result-actions {
  flex-shrink: 0;
  white-space: nowrap;
}

.search-result-button {
  align-items: center;
  display: inline-flex;
  height: 2.25rem;
  justify-content: center;
  width: 2.25rem;
}
</style>
