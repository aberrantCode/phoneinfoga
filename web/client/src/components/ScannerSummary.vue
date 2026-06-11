<template>
  <div class="scanner-summary">
    <div class="scanner-summary-status">
      <b-badge :variant="badge.variant" class="scanner-summary-badge">
        <b-icon
          v-if="badge.icon"
          :icon="badge.icon"
          aria-hidden="true"
          class="mr-1"
        />
        {{ badge.label }}
      </b-badge>
      <span v-if="headline" class="scanner-summary-headline">
        {{ headline }}
      </span>
      <span v-if="subtext" class="scanner-summary-subtext text-muted">
        {{ subtext }}
      </span>
    </div>

    <slot>
      <b-row v-if="groups.length > 0" class="scanner-summary-groups">
        <b-col
          v-for="group in groups"
          :key="group.key"
          cols="12"
          :md="colMd"
          class="mb-3"
        >
          <b-card no-body class="scanner-summary-group h-100">
            <b-card-header class="scanner-summary-group-header">
              {{ group.label }}
            </b-card-header>
            <b-list-group flush>
              <b-list-group-item
                v-for="row in group.rows"
                :key="row.label"
                class="scanner-summary-row"
                :class="{ 'scanner-summary-row-changed': row.changed }"
              >
                <span class="scanner-summary-row-label text-muted">
                  {{ row.label }}
                </span>
                <span class="scanner-summary-row-value">
                  {{ row.value }}
                  <b-badge
                    v-if="row.changed"
                    variant="warning"
                    class="ml-2 scanner-summary-row-badge"
                  >
                    changed
                  </b-badge>
                </span>
              </b-list-group-item>
            </b-list-group>
          </b-card>
        </b-col>
      </b-row>
      <p v-else-if="emptyText" class="text-muted mb-0">{{ emptyText }}</p>
    </slot>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from "vue-property-decorator";

export interface SummaryBadge {
  variant: string;
  label: string;
  icon?: string;
}

export interface SummaryRow {
  label: string;
  value: string;
  changed?: boolean;
}

export interface SummaryGroup {
  key: string;
  label: string;
  rows: SummaryRow[];
}

@Component
export default class ScannerSummary extends Vue {
  @Prop({ required: true }) badge!: SummaryBadge;
  @Prop({ default: "" }) headline!: string;
  @Prop({ default: "" }) subtext!: string;
  @Prop({ default: () => [] }) groups!: SummaryGroup[];
  @Prop({ default: 6 }) colMd!: number;
  @Prop({ default: "" }) emptyText!: string;
}
</script>

<style scoped>
.scanner-summary-status {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
}

.scanner-summary-badge {
  align-items: center;
  display: inline-flex;
  font-size: 0.9rem;
  padding: 0.45rem 0.75rem;
}

.scanner-summary-headline {
  font-size: 1.25rem;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.scanner-summary-subtext {
  font-size: 0.95rem;
}

.scanner-summary-group {
  border: 1px solid #e9ecef;
}

.scanner-summary-group-header {
  background-color: #f8f9fa;
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.scanner-summary-row {
  align-items: baseline;
  display: flex;
  justify-content: space-between;
  padding: 0.6rem 1rem;
}

.scanner-summary-row-label {
  font-size: 0.85rem;
  margin-right: 1rem;
  white-space: nowrap;
}

.scanner-summary-row-value {
  font-weight: 600;
  text-align: right;
  word-break: break-word;
}

.scanner-summary-row-changed {
  background-color: #fff8e1;
}

.scanner-summary-row-changed .scanner-summary-row-value {
  color: #8a6d3b;
  font-weight: 700;
}

.scanner-summary-row-badge {
  font-size: 0.65rem;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}
</style>
