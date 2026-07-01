<template>
  <div class="provider-comparison text-left">
    <p v-if="columns.length === 0" class="text-muted mb-0">
      No provider results yet. Run the validation scanners to compare them here.
    </p>

    <template v-else>
      <p class="provider-comparison-hint text-muted">
        One row per property, one column per provider.
        <span class="provider-comparison-hint-flag">
          <b-icon icon="exclamation-triangle-fill" aria-hidden="true" />
          Highlighted rows are where the providers disagree.
        </span>
      </p>

      <div class="provider-comparison-scroll">
        <table class="provider-comparison-table">
          <thead>
            <tr>
              <th scope="col" class="provider-comparison-corner">Property</th>
              <th
                v-for="column in columns"
                :key="column.key"
                scope="col"
                class="provider-comparison-provider"
              >
                {{ column.label }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in rows"
              :key="row.key"
              :class="{ 'provider-comparison-row-disagree': row.disagreement }"
            >
              <th scope="row" class="provider-comparison-label">
                <b-icon
                  v-if="row.disagreement"
                  icon="exclamation-triangle-fill"
                  aria-hidden="true"
                  class="provider-comparison-warn"
                />
                {{ row.label }}
              </th>
              <td
                v-for="cell in row.cells"
                :key="cell.columnKey"
                class="provider-comparison-value"
                :class="{ 'provider-comparison-value-empty': !cell.filled }"
              >
                <template v-if="cell.filled">{{ cell.value }}</template>
                <span v-else aria-hidden="true">&mdash;</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from "vue-property-decorator";
import {
  buildComparison,
  Comparison,
  ComparisonColumn,
  ComparisonRow,
} from "@/utils/providerComparison";

type Raw = Record<string, unknown>;

@Component
export default class ProviderComparison extends Vue {
  // The offline libphonenumber metadata (Scan.vue's localData), or null to omit
  // the baseline column.
  @Prop({ default: null }) baseline!: Raw | null;
  // Live scanner results keyed by scanId (e.g. { numverify: {...}, twilio: {...} }).
  @Prop({ default: () => ({}) }) results!: { [scanId: string]: Raw };

  get comparison(): Comparison {
    return buildComparison(this.baseline, this.results);
  }

  get columns(): ComparisonColumn[] {
    return this.comparison.columns;
  }

  get rows(): ComparisonRow[] {
    return this.comparison.rows;
  }
}
</script>

<style scoped>
.provider-comparison-hint {
  font-size: 0.85rem;
  margin-bottom: 0.85rem;
}

.provider-comparison-hint-flag {
  color: var(--led-warn);
  margin-left: 0.35rem;
  white-space: nowrap;
}

/* Horizontal scroll keeps the table usable on narrow screens without breaking
   the print layout, where the container expands to fit. */
.provider-comparison-scroll {
  overflow-x: auto;
}

.provider-comparison-table {
  border-collapse: collapse;
  width: 100%;
}

.provider-comparison-table th,
.provider-comparison-table td {
  border-bottom: 1px solid var(--rule-soft);
  padding: 0.55rem 0.85rem;
  text-align: left;
  vertical-align: top;
}

.provider-comparison-table thead th {
  border-bottom: 1px solid var(--rule);
  color: var(--muted);
  font-size: 0.78rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

/* The property column is pinned so values stay associated with their row when
   the table scrolls sideways. */
.provider-comparison-corner,
.provider-comparison-label {
  background: var(--surface-1);
  left: 0;
  position: sticky;
  z-index: 1;
}

.provider-comparison-label {
  color: var(--ink);
  font-weight: 600;
  white-space: nowrap;
}

.provider-comparison-provider {
  white-space: nowrap;
}

.provider-comparison-value {
  overflow-wrap: anywhere;
}

.provider-comparison-value-empty {
  color: var(--muted);
}

/* A disagreeing row is tinted as a whole — the majority value is not assumed to
   be correct, so no single cell is branded as the outlier. */
.provider-comparison-row-disagree td,
.provider-comparison-row-disagree .provider-comparison-label {
  background: color-mix(in oklch, var(--led-warn) 12%, transparent);
}

.provider-comparison-row-disagree .provider-comparison-label {
  color: var(--led-warn);
}

.provider-comparison-warn {
  color: var(--led-warn);
  margin-right: 0.35rem;
}
</style>
