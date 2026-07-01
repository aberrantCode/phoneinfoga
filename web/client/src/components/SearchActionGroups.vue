<template>
  <div>
    <div v-if="mode === 'google'">
      <p class="text-muted mb-3">
        Generated search actions grouped by investigation intent. Open a single
        query, copy it for adjustment, or launch timed batches.
      </p>

      <b-card
        v-for="group in googleGroups"
        :key="group.key"
        no-body
        class="search-action-group"
      >
        <b-card-header
          class="search-action-group-header"
          @click="toggleCategory(group.key)"
        >
          <div
            class="d-flex flex-wrap align-items-center justify-content-between"
          >
            <div class="search-action-group-title">
              <h4 class="h5 mb-1">
                <span class="search-action-chevron" aria-hidden="true">
                  {{ isCategoryExpanded(group.key) ? "-" : "+" }}
                </span>
                {{ group.label }}
                <b-badge variant="secondary" class="ml-2">
                  {{ group.items.length }}
                </b-badge>
              </h4>
              <p class="text-muted mb-0">{{ group.description }}</p>
            </div>
            <div class="d-flex align-items-center mt-2 mt-sm-0">
              <b-badge v-if="launcherRunning" variant="info" class="mr-2">
                {{ launcherRemaining }} queued
              </b-badge>
              <b-button
                v-if="!launcherRunning"
                variant="outline-primary"
                size="sm"
                @click.stop="startLauncher(group.items)"
              >
                Open All
              </b-button>
              <b-button
                v-else
                variant="outline-danger"
                size="sm"
                @click.stop="stopLauncher"
              >
                Stop
              </b-button>
            </div>
          </div>
        </b-card-header>

        <b-collapse
          :id="categoryCollapseId(group.key)"
          :visible="isCategoryExpanded(group.key)"
        >
          <b-card-body>
            <b-table
              small
              responsive
              outlined
              :items="group.items"
              :fields="googleFields"
            >
              <template #cell(dork)="row">
                <code class="search-query">{{ row.item.dork }}</code>
              </template>
              <template #cell(actions)="row">
                <b-button
                  :id="actionButtonId('open', group.key, row.index)"
                  :href="row.item.url"
                  target="_blank"
                  rel="noopener"
                  variant="dark"
                  size="sm"
                  class="search-action-button mr-1 mb-1"
                  aria-label="Open query in Google"
                >
                  <b-icon-box-arrow-up-right aria-hidden="true" />
                </b-button>
                <b-tooltip
                  :target="actionButtonId('open', group.key, row.index)"
                >
                  Open query in Google
                </b-tooltip>
                <b-button
                  :id="actionButtonId('copy-query', group.key, row.index)"
                  variant="outline-secondary"
                  size="sm"
                  class="search-action-button mr-1 mb-1"
                  aria-label="Copy query text"
                  @click="copyText(row.item.dork)"
                >
                  <b-icon-clipboard aria-hidden="true" />
                </b-button>
                <b-tooltip
                  :target="actionButtonId('copy-query', group.key, row.index)"
                >
                  Copy query text
                </b-tooltip>
                <b-button
                  :id="actionButtonId('copy-url', group.key, row.index)"
                  variant="outline-secondary"
                  size="sm"
                  class="search-action-button mb-1"
                  aria-label="Copy Google search URL"
                  @click="copyText(row.item.url)"
                >
                  <b-icon-link45deg aria-hidden="true" />
                </b-button>
                <b-tooltip
                  :target="actionButtonId('copy-url', group.key, row.index)"
                >
                  Copy Google search URL
                </b-tooltip>
              </template>
            </b-table>
          </b-card-body>
        </b-collapse>
      </b-card>
    </div>

    <div v-else>
      <p class="text-muted mb-3">{{ resultsIntro }}</p>

      <b-card
        v-for="group in resultsGroups"
        :key="group.key"
        no-body
        class="search-action-group"
      >
        <b-card-header
          class="search-action-group-header"
          @click="toggleCategory(group.key)"
        >
          <div
            class="d-flex flex-wrap align-items-center justify-content-between"
          >
            <div class="search-action-group-title">
              <h4 class="h5 mb-1">
                <span class="search-action-chevron" aria-hidden="true">
                  {{ isCategoryExpanded(group.key) ? "-" : "+" }}
                </span>
                {{ group.label }}
                <b-badge variant="secondary" class="ml-2">
                  {{ group.items.length }}
                </b-badge>
                <b-badge
                  :variant="
                    groupMatchCount(group.items) > 0 ? 'success' : 'light'
                  "
                  class="ml-1"
                >
                  {{ groupMatchCount(group.items) }} matched
                </b-badge>
              </h4>
              <p class="text-muted mb-0">{{ group.description }}</p>
            </div>
            <div
              v-if="supportsLauncher"
              class="d-flex align-items-center mt-2 mt-sm-0"
            >
              <b-badge v-if="launcherRunning" variant="info" class="mr-2">
                {{ launcherRemaining }} queued
              </b-badge>
              <b-button
                v-if="!launcherRunning"
                variant="outline-primary"
                size="sm"
                :disabled="groupMatchCount(group.items) === 0"
                @click.stop="startLauncher(matchedRows(group.items))"
              >
                Open Matches
              </b-button>
              <b-button
                v-else
                variant="outline-danger"
                size="sm"
                @click.stop="stopLauncher"
              >
                Stop
              </b-button>
            </div>
          </div>
        </b-card-header>

        <b-collapse
          :id="categoryCollapseId(group.key)"
          :visible="isCategoryExpanded(group.key)"
        >
          <b-card-body>
            <b-table
              small
              responsive
              outlined
              :items="group.items"
              :fields="searxngFields"
            >
              <template #cell(dork)="row">
                <code class="search-query">{{ row.item.dork }}</code>
                <b-badge
                  v-if="row.item.engine"
                  variant="light"
                  class="ml-2 search-engine-badge"
                >
                  {{ row.item.engine }}
                </b-badge>
                <p v-if="row.item.error" class="text-danger small mb-0 mt-2">
                  {{ row.item.error }}
                </p>
              </template>
              <template #cell(result_count)="row">
                <b-badge
                  :variant="
                    row.item.error
                      ? 'warning'
                      : row.item.result_count > 0
                      ? 'success'
                      : 'secondary'
                  "
                >
                  {{ row.item.error ? "Error" : row.item.result_count }}
                </b-badge>
              </template>
              <template #cell(results)="row">
                <SearchResultList
                  :results="row.item.results || []"
                  :id-prefix="`${idPrefix}-${group.key}-${row.index}`"
                  :show-engine="true"
                  empty-text="No inline hits"
                />
              </template>
              <template #cell(actions)="row">
                <b-button
                  v-if="row.item.url"
                  :id="actionButtonId('searxng-open', group.key, row.index)"
                  :href="row.item.url"
                  target="_blank"
                  rel="noopener"
                  variant="dark"
                  size="sm"
                  class="search-action-button mr-1 mb-1"
                  aria-label="Open query in SearXNG"
                >
                  <b-icon-box-arrow-up-right aria-hidden="true" />
                </b-button>
                <b-tooltip
                  v-if="row.item.url"
                  :target="actionButtonId('searxng-open', group.key, row.index)"
                >
                  Open query in SearXNG
                </b-tooltip>
                <b-button
                  :id="
                    actionButtonId('searxng-copy-query', group.key, row.index)
                  "
                  variant="outline-secondary"
                  size="sm"
                  class="search-action-button mr-1 mb-1"
                  aria-label="Copy query text"
                  @click="copyText(row.item.dork)"
                >
                  <b-icon-clipboard aria-hidden="true" />
                </b-button>
                <b-tooltip
                  :target="
                    actionButtonId('searxng-copy-query', group.key, row.index)
                  "
                >
                  Copy query text
                </b-tooltip>
                <b-button
                  v-if="row.item.url"
                  :id="actionButtonId('searxng-copy-url', group.key, row.index)"
                  variant="outline-secondary"
                  size="sm"
                  class="search-action-button mb-1"
                  aria-label="Copy SearXNG search URL"
                  @click="copyText(row.item.url)"
                >
                  <b-icon-link45deg aria-hidden="true" />
                </b-button>
                <b-tooltip
                  v-if="row.item.url"
                  :target="
                    actionButtonId('searxng-copy-url', group.key, row.index)
                  "
                >
                  Copy SearXNG search URL
                </b-tooltip>
              </template>
            </b-table>
          </b-card-body>
        </b-collapse>
      </b-card>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from "vue-property-decorator";
import SearchResultList from "./SearchResultList.vue";

interface GoogleSearchDork {
  number: string;
  dork: string;
  url: string;
}

interface GoogleSearchResponse {
  social_media?: GoogleSearchDork[];
  disposable_providers?: GoogleSearchDork[];
  reputation?: GoogleSearchDork[];
  individuals?: GoogleSearchDork[];
  general?: GoogleSearchDork[];
}

interface GoogleSearchGroup {
  key: keyof GoogleSearchResponse;
  label: string;
  description: string;
  items: GoogleSearchDork[];
}

// Shared shape for the "results" modes (searxng, serpapi). SearXNG carries a
// per-query `url` to reopen the search; SerpAPI has none (results are already
// fetched) but carries the `engine` that produced them. url-dependent controls
// are rendered conditionally so the same layout serves both.
interface SearXNGQueryResult {
  number: string;
  dork: string;
  url?: string;
  engine?: string;
  result_count: number;
  results?: {
    title?: string;
    url: string;
    content?: string;
    engine?: string;
  }[];
  error?: string;
}

interface SearXNGResponse {
  social_media?: SearXNGQueryResult[];
  disposable_providers?: SearXNGQueryResult[];
  reputation?: SearXNGQueryResult[];
  individuals?: SearXNGQueryResult[];
  general?: SearXNGQueryResult[];
}

interface SearXNGGroup {
  key: keyof SearXNGResponse;
  label: string;
  description: string;
  items: SearXNGQueryResult[];
}

interface LaunchableSearch {
  url: string;
}

interface SearchGroupDefinition {
  key: keyof GoogleSearchResponse;
  label: string;
  description: string;
}

@Component({
  components: {
    SearchResultList,
  },
})
export default class SearchActionGroups extends Vue {
  @Prop({ required: true }) mode!: "google" | "searxng" | "serpapi";
  @Prop({ default: () => ({}) }) data!: GoogleSearchResponse | SearXNGResponse;
  @Prop({ default: "search" }) idPrefix!: string;

  categoryExpanded: { [key: string]: boolean } = {};
  launcherQueue: LaunchableSearch[] = [];
  launcherTimer: number | null = null;
  launcherRunning = false;
  openedUrls = new Set<string>();
  googleFields = [
    { key: "dork", label: "Query", thClass: "search-query-column" },
    { key: "actions", label: "Actions", thClass: "search-actions-column" },
  ];
  searxngFields = [
    { key: "dork", label: "Query", thClass: "searxng-query-column" },
    { key: "result_count", label: "Results", thClass: "searxng-count-column" },
    { key: "results", label: "Top results", thClass: "searxng-results-column" },
    { key: "actions", label: "Actions", thClass: "searxng-actions-column" },
  ];

  beforeDestroy(): void {
    this.stopLauncher();
  }

  get searchGroupDefinitions(): SearchGroupDefinition[] {
    return [
      {
        key: "general",
        label: "General footprints",
        description: "Broad exact-number searches and common web mentions.",
      },
      {
        key: "social_media",
        label: "Social networks",
        description: "Profile and post searches across major social platforms.",
      },
      {
        key: "individuals",
        label: "People and identity",
        description: "Queries aimed at personal listings and identity clues.",
      },
      {
        key: "reputation",
        label: "Reputation and spam",
        description: "Spam reports, complaints, and abuse-related mentions.",
      },
      {
        key: "disposable_providers",
        label: "Temporary number providers",
        description: "Disposable-number provider searches and public listings.",
      },
    ];
  }

  get googleGroups(): GoogleSearchGroup[] {
    const response = (this.data || {}) as GoogleSearchResponse;

    return this.searchGroupDefinitions
      .map((group) => ({
        ...group,
        items: response[group.key] || [],
      }))
      .filter((group) => group.items.length > 0);
  }

  // Serves both the searxng and serpapi modes: each returns per-query result
  // counts and inline hits grouped by investigation intent.
  get resultsGroups(): SearXNGGroup[] {
    const response = (this.data || {}) as SearXNGResponse;

    return this.searchGroupDefinitions
      .map((group) => ({
        ...group,
        items: response[group.key] || [],
      }))
      .filter((group) => group.items.length > 0);
  }

  get resultsIntro(): string {
    if (this.mode === "serpapi") {
      return "SerpApi ran each search action through live search engines and returned match counts with top results. Inspect hits inline or copy a query for adjustment.";
    }
    return "SearXNG checked each search action and returned match counts with top results. Open matching searches, inspect hits inline, or copy a query for adjustment.";
  }

  // Only searxng exposes a per-query search URL, so the batch launcher (Open
  // Matches / Stop) applies to searxng alone. SerpAPI results are pre-fetched.
  get supportsLauncher(): boolean {
    return this.mode === "searxng";
  }

  get launcherRemaining(): number {
    return this.launcherQueue.length;
  }

  get allGoogleDorks(): GoogleSearchDork[] {
    return this.googleGroups.reduce(
      (items: GoogleSearchDork[], group: GoogleSearchGroup) =>
        items.concat(group.items),
      []
    );
  }

  get matchedSearXNGQueries(): SearXNGQueryResult[] {
    return this.resultsGroups.reduce(
      (items: SearXNGQueryResult[], group: SearXNGGroup) =>
        items.concat(this.matchedRows(group.items)),
      []
    );
  }

  categoryCollapseId(key: string): string {
    return `${this.idPrefix}-${key}`;
  }

  isCategoryExpanded(key: string): boolean {
    return this.categoryExpanded[key] === true;
  }

  toggleCategory(key: string): void {
    this.$set(this.categoryExpanded, key, !this.isCategoryExpanded(key));
  }

  actionButtonId(action: string, groupKey: string, index: number): string {
    return `${this.idPrefix}-${groupKey}-${index}-${action}`;
  }

  groupMatchCount(rows: SearXNGQueryResult[]): number {
    return this.matchedRows(rows).length;
  }

  matchedRows(rows: SearXNGQueryResult[]): SearXNGQueryResult[] {
    return rows.filter((row) => row.result_count > 0 && !row.error);
  }

  // Public methods invoked by the parent panel header buttons via a ref.
  openAll(): void {
    this.startLauncher(this.allGoogleDorks);
  }

  openMatches(): void {
    this.startLauncher(this.matchedSearXNGQueries);
  }

  stop(): void {
    this.stopLauncher();
  }

  // Accepts any row that may carry a search URL (Google dorks always do;
  // SearXNG query results type it as optional). Rows without a usable URL are
  // dropped — they were never launchable.
  startLauncher(dorks: Array<{ url?: string }>): void {
    const queued: LaunchableSearch[] = dorks.filter(
      (dork): dork is LaunchableSearch =>
        typeof dork.url === "string" &&
        dork.url !== "" &&
        !this.openedUrls.has(dork.url)
    );
    if (queued.length === 0) {
      return;
    }

    this.stopLauncher();
    this.launcherQueue = queued;
    this.launcherRunning = true;
    this.$emit("launcher-change", true);
    this.launchNext();
  }

  stopLauncher(): void {
    if (this.launcherTimer !== null) {
      window.clearTimeout(this.launcherTimer);
      this.launcherTimer = null;
    }
    this.launcherRunning = false;
    this.launcherQueue = [];
    this.$emit("launcher-change", false);
  }

  launchNext(): void {
    const next = this.launcherQueue.shift();
    if (!next) {
      this.stopLauncher();
      return;
    }

    this.openedUrls.add(next.url);
    window.open(next.url, "_blank", "noopener");

    if (this.launcherQueue.length === 0) {
      this.stopLauncher();
      return;
    }

    const delay = 1500 + Math.floor(Math.random() * 3500);
    this.launcherTimer = window.setTimeout(() => this.launchNext(), delay);
  }

  async copyText(text: string): Promise<void> {
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      const el = document.createElement("textarea");
      el.value = text;
      el.setAttribute("readonly", "true");
      el.style.position = "absolute";
      el.style.left = "-9999px";
      document.body.appendChild(el);
      el.select();
      document.execCommand("copy");
      document.body.removeChild(el);
    }
  }
}
</script>

<style scoped>
.search-action-group-header {
  cursor: pointer;
}

.search-action-chevron {
  display: inline-block;
  font-size: 1rem;
  margin-right: 0.5rem;
  width: 1rem;
}

.search-action-group {
  margin-bottom: 1rem;
}

.search-action-group-title {
  min-width: 16rem;
}

.search-query {
  white-space: normal;
  word-break: break-word;
}

.search-engine-badge {
  letter-spacing: 0.02em;
  text-transform: uppercase;
}

::v-deep .search-query-column {
  width: 82%;
}

::v-deep .search-actions-column {
  width: 10rem;
}

::v-deep .searxng-query-column {
  width: 42%;
}

::v-deep .searxng-count-column {
  width: 6rem;
}

::v-deep .searxng-results-column {
  width: 38%;
}

::v-deep .searxng-actions-column {
  width: 10rem;
}

.search-action-button {
  align-items: center;
  display: inline-flex;
  height: 2.25rem;
  justify-content: center;
  width: 2.25rem;
}
</style>
