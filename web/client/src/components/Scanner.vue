<template>
  <b-card no-body class="mb-3 scanner-panel text-left">
    <b-card-header
      class="bg-white scanner-panel-header"
      @click="expanded = !expanded"
    >
      <div class="d-flex align-items-center justify-content-between">
        <button
          class="scanner-panel-toggle"
          type="button"
          :aria-expanded="expanded ? 'true' : 'false'"
          :aria-controls="collapseId"
          @click.stop="expanded = !expanded"
        >
          <span class="scanner-panel-chevron" aria-hidden="true">
            {{ expanded ? "-" : "+" }}
          </span>
          <span>{{ displayName }}</span>
        </button>

        <div class="d-flex align-items-center">
          <b-spinner
            v-if="loading && !error"
            small
            type="grow"
            class="mr-2"
          ></b-spinner>
          <b-badge v-if="hasData && !loading" variant="success" class="mr-2">
            {{ readyLabel }}
          </b-badge>
          <b-badge
            v-if="dryrunError && !loading"
            variant="warning"
            class="mr-2"
          >
            Unavailable
          </b-badge>
          <b-button
            v-if="isGoogleSearch && hasData && !launcherRunning"
            @click.stop="startLauncher(allGoogleDorks)"
            variant="outline-primary"
            size="sm"
            class="mr-2"
            >Open All</b-button
          >
          <b-button
            v-if="isSearXNGSearch && hasData && !launcherRunning"
            @click.stop="startLauncher(matchedSearXNGQueries)"
            variant="outline-primary"
            size="sm"
            class="mr-2"
            >Open Matches</b-button
          >
          <b-button
            v-if="(isGoogleSearch || isSearXNGSearch) && launcherRunning"
            @click.stop="stopLauncher"
            variant="outline-danger"
            size="sm"
            class="mr-2"
            >Stop</b-button
          >
          <b-button
            v-if="loading"
            @click.stop="cancelScan"
            variant="outline-danger"
            size="sm"
            class="mr-2"
            >Cancel</b-button
          >
          <b-button
            v-if="!error && !loading && !hasData"
            @click.stop="runScan"
            variant="dark"
            size="sm"
            >Run</b-button
          >
          <b-button
            v-if="error && !loading && !dryrunError"
            @click.stop="runScan"
            variant="danger"
            size="sm"
            >Retry</b-button
          >
        </div>
      </div>
    </b-card-header>

    <b-collapse :id="collapseId" v-model="expanded">
      <b-card-body>
        <b-alert v-if="error && !loading" show variant="danger" fade>
          {{ error }}
        </b-alert>

        <div v-else-if="isGoogleSearch && hasData">
          <p class="text-muted mb-3">
            Generated search actions grouped by investigation intent. Open a
            single query, copy it for adjustment, or launch timed batches.
          </p>

          <b-card
            v-for="group in googleGroups"
            :key="group.key"
            no-body
            class="google-action-group"
          >
            <b-card-header
              class="google-action-group-header"
              @click="toggleGoogleCategory(group.key)"
            >
              <div
                class="d-flex flex-wrap align-items-center justify-content-between"
              >
                <div class="google-action-group-title">
                  <h4 class="h5 mb-1">
                    <span class="scanner-panel-chevron" aria-hidden="true">
                      {{ isGoogleCategoryExpanded(group.key) ? "-" : "+" }}
                    </span>
                    {{ group.label }}
                    <b-badge variant="secondary" class="ml-2">
                      {{ group.items.length }}
                    </b-badge>
                  </h4>
                  <p class="text-muted mb-0">{{ group.description }}</p>
                </div>
                <div class="d-flex align-items-center mt-2 mt-sm-0">
                  <b-badge
                    v-if="launcherRunning"
                    variant="info"
                    class="mr-2"
                  >
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
              :visible="isGoogleCategoryExpanded(group.key)"
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
                    <code class="google-query">{{ row.item.dork }}</code>
                  </template>
                  <template #cell(actions)="row">
                    <b-button
                      :id="actionButtonId('open', group.key, row.index)"
                      :href="row.item.url"
                      target="_blank"
                      rel="noopener"
                      variant="dark"
                      size="sm"
                      class="google-action-button mr-1 mb-1"
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
                      class="google-action-button mr-1 mb-1"
                      aria-label="Copy query text"
                      @click="copyText(row.item.dork)"
                    >
                      <b-icon-clipboard aria-hidden="true" />
                    </b-button>
                    <b-tooltip
                      :target="
                        actionButtonId('copy-query', group.key, row.index)
                      "
                    >
                      Copy query text
                    </b-tooltip>
                    <b-button
                      :id="actionButtonId('copy-url', group.key, row.index)"
                      variant="outline-secondary"
                      size="sm"
                      class="google-action-button mb-1"
                      aria-label="Copy Google search URL"
                      @click="copyText(row.item.url)"
                    >
                      <b-icon-link45deg aria-hidden="true" />
                    </b-button>
                    <b-tooltip
                      :target="
                        actionButtonId('copy-url', group.key, row.index)
                      "
                    >
                      Copy Google search URL
                    </b-tooltip>
                  </template>
                </b-table>
              </b-card-body>
            </b-collapse>
          </b-card>
        </div>

        <div v-else-if="isSearXNGSearch && hasData">
          <p class="text-muted mb-3">
            SearXNG checked each search action and returned match counts with
            top results. Open matching searches, inspect hits inline, or copy a
            query for adjustment.
          </p>

          <b-card
            v-for="group in searxngGroups"
            :key="group.key"
            no-body
            class="google-action-group"
          >
            <b-card-header
              class="google-action-group-header"
              @click="toggleSearchCategory(group.key)"
            >
              <div
                class="d-flex flex-wrap align-items-center justify-content-between"
              >
                <div class="google-action-group-title">
                  <h4 class="h5 mb-1">
                    <span class="scanner-panel-chevron" aria-hidden="true">
                      {{ isSearchCategoryExpanded(group.key) ? "-" : "+" }}
                    </span>
                    {{ group.label }}
                    <b-badge variant="secondary" class="ml-2">
                      {{ group.items.length }}
                    </b-badge>
                    <b-badge
                      :variant="groupMatchCount(group.items) > 0 ? 'success' : 'light'"
                      class="ml-1"
                    >
                      {{ groupMatchCount(group.items) }} matched
                    </b-badge>
                  </h4>
                  <p class="text-muted mb-0">{{ group.description }}</p>
                </div>
                <div class="d-flex align-items-center mt-2 mt-sm-0">
                  <b-badge
                    v-if="launcherRunning"
                    variant="info"
                    class="mr-2"
                  >
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
              :visible="isSearchCategoryExpanded(group.key)"
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
                    <code class="google-query">{{ row.item.dork }}</code>
                    <p v-if="row.item.error" class="text-danger small mb-0 mt-2">
                      {{ row.item.error }}
                    </p>
                  </template>
                  <template #cell(result_count)="row">
                    <b-badge
                      :variant="row.item.error ? 'warning' : row.item.result_count > 0 ? 'success' : 'secondary'"
                    >
                      {{ row.item.error ? "Error" : row.item.result_count }}
                    </b-badge>
                  </template>
                  <template #cell(results)="row">
                    <div
                      v-if="row.item.results && row.item.results.length > 0"
                      class="searxng-results"
                    >
                      <div
                        v-for="(result, resultIndex) in row.item.results"
                        :key="`${row.item.url}-${resultIndex}`"
                        class="searxng-result"
                      >
                        <a :href="result.url" target="_blank" rel="noopener">
                          {{ result.title || result.url }}
                        </a>
                        <span v-if="result.engine" class="text-muted small">
                          {{ result.engine }}
                        </span>
                        <p v-if="result.content" class="text-muted small mb-0">
                          {{ result.content }}
                        </p>
                      </div>
                    </div>
                    <span v-else class="text-muted">No inline hits</span>
                  </template>
                  <template #cell(actions)="row">
                    <b-button
                      :id="actionButtonId('searxng-open', group.key, row.index)"
                      :href="row.item.url"
                      target="_blank"
                      rel="noopener"
                      variant="dark"
                      size="sm"
                      class="google-action-button mr-1 mb-1"
                      aria-label="Open query in SearXNG"
                    >
                      <b-icon-box-arrow-up-right aria-hidden="true" />
                    </b-button>
                    <b-tooltip
                      :target="
                        actionButtonId('searxng-open', group.key, row.index)
                      "
                    >
                      Open query in SearXNG
                    </b-tooltip>
                    <b-button
                      :id="
                        actionButtonId('searxng-copy-query', group.key, row.index)
                      "
                      variant="outline-secondary"
                      size="sm"
                      class="google-action-button mr-1 mb-1"
                      aria-label="Copy query text"
                      @click="copyText(row.item.dork)"
                    >
                      <b-icon-clipboard aria-hidden="true" />
                    </b-button>
                    <b-tooltip
                      :target="
                        actionButtonId(
                          'searxng-copy-query',
                          group.key,
                          row.index
                        )
                      "
                    >
                      Copy query text
                    </b-tooltip>
                    <b-button
                      :id="
                        actionButtonId('searxng-copy-url', group.key, row.index)
                      "
                      variant="outline-secondary"
                      size="sm"
                      class="google-action-button mb-1"
                      aria-label="Copy SearXNG search URL"
                      @click="copyText(row.item.url)"
                    >
                      <b-icon-link45deg aria-hidden="true" />
                    </b-button>
                    <b-tooltip
                      :target="
                        actionButtonId(
                          'searxng-copy-url',
                          group.key,
                          row.index
                        )
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

        <JsonViewer v-else-if="hasData" :value="data"></JsonViewer>

        <p v-else-if="!loading" class="text-muted mb-0">
          Run this scanner to view its results.
        </p>
      </b-card-body>
    </b-collapse>
  </b-card>
</template>

<script lang="ts">
import { Component, Vue, Prop } from "vue-property-decorator";
import axios, { CancelTokenSource } from "axios";
import { mapState, mapMutations } from "vuex";
import JsonViewer from "vue-json-viewer";
import config from "@/config";

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

interface SearXNGResultItem {
  title?: string;
  url: string;
  content?: string;
  engine?: string;
}

interface SearXNGQueryResult {
  number: string;
  dork: string;
  url: string;
  result_count: number;
  results?: SearXNGResultItem[];
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

@Component({
  components: {
    JsonViewer,
  },
})
export default class Scanner extends Vue {
  data: unknown = null;
  loading = false;
  dryrunError = false;
  error: unknown = null;
  expanded = false;
  googleCategoryExpanded: { [key: string]: boolean } = {};
  searchCategoryExpanded: { [key: string]: boolean } = {};
  launcherQueue: LaunchableSearch[] = [];
  launcherTimer: number | null = null;
  launcherRunning = false;
  cancelSource: CancelTokenSource | null = null;
  scanStartedAt = 0;
  etaTimer: number | null = null;
  openedUrls = new Set<string>();
  googleFields = [
    { key: "dork", label: "Query", thClass: "google-query-column" },
    { key: "actions", label: "Actions", thClass: "google-actions-column" },
  ];
  searxngFields = [
    { key: "dork", label: "Query", thClass: "searxng-query-column" },
    { key: "result_count", label: "Results", thClass: "searxng-count-column" },
    { key: "results", label: "Top results", thClass: "searxng-results-column" },
    { key: "actions", label: "Actions", thClass: "searxng-actions-column" },
  ];
  computed = {
    ...mapState(["number"]),
    ...mapMutations(["pushError"]),
  };

  @Prop() scanId!: string;
  @Prop() name!: string;
  @Prop({ default: false }) autoRun!: boolean;
  @Prop({ default: () => ({}) }) scanOptions!: { [key: string]: number };

  get collapseId(): string {
    return `scanner-collapse-${this.scanId}`;
  }

  get displayName(): string {
    if (this.scanId === "googlesearch") {
      return "Google search";
    }

    if (this.scanId === "googlecse") {
      return "Google custom search";
    }

    if (this.scanId === "searxng") {
      return "SearXNG search";
    }

    return this.name;
  }

  get hasData(): boolean {
    return this.data !== null && this.data !== undefined;
  }

  get isGoogleSearch(): boolean {
    return this.scanId === "googlesearch";
  }

  get isSearXNGSearch(): boolean {
    return this.scanId === "searxng";
  }

  get searchGroupDefinitions(): Omit<GoogleSearchGroup, "items">[] {
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

  get allGoogleDorks(): GoogleSearchDork[] {
    return this.googleGroups.reduce(
      (items: GoogleSearchDork[], group: GoogleSearchGroup) =>
        items.concat(group.items),
      []
    );
  }

  get searxngGroups(): SearXNGGroup[] {
    const response = (this.data || {}) as SearXNGResponse;

    return this.searchGroupDefinitions
      .map((group) => ({
        ...group,
        items: response[group.key] || [],
      }))
      .filter((group) => group.items.length > 0);
  }

  get matchedSearXNGQueries(): SearXNGQueryResult[] {
    return this.searxngGroups.reduce(
      (items: SearXNGQueryResult[], group: SearXNGGroup) =>
        items.concat(this.matchedRows(group.items)),
      []
    );
  }

  get launcherRemaining(): number {
    return this.launcherQueue.length;
  }

  get readyLabel(): string {
    return this.isGoogleSearch ? "Queries ready" : "Results ready";
  }

  get estimatedDurationMs(): number {
    if (this.scanId === "searxng") {
      const delay = Number(this.scanOptions.SEARXNG_DELAY_MS || 0);
      return 35 * Math.max(delay, 0) + 10000;
    }

    if (this.scanId === "googlesearch") {
      return 1000;
    }

    return 10000;
  }

  async mounted(): Promise<void> {
    const available = await this.dryRun();
    if (available && this.autoRun) {
      this.runScan();
    }
  }

  beforeDestroy(): void {
    this.cancelScan();
    this.stopLauncher();
    this.stopEtaTicker();
  }

  private async dryRun(): Promise<boolean> {
    try {
      const res = await axios.post(
        `${config.apiUrl}/v2/scanners/${this.scanId}/dryrun`,
        {
          number: this.$store.state.number,
        },
        {
          validateStatus: () => true,
        }
      );

      if (!res.data.success && res.data.error) {
        throw res.data.error;
      }
      return true;
    } catch (error: unknown) {
      this.dryrunError = true;
      this.error = error;
      return false;
    }
  }

  private async runScan(): Promise<void> {
    this.error = null;
    this.loading = true;
    this.cancelSource = axios.CancelToken.source();
    this.scanStartedAt = Date.now();
    this.startEtaTicker();
    this.emitStatus(
      "running",
      `Running ${this.displayName}`,
      this.remainingEtaMs()
    );
    try {
      const res = await axios.post(
        `${config.apiUrl}/v2/scanners/${this.scanId}/run`,
        {
          number: this.$store.state.number,
          options: this.scanOptions,
        },
        {
          cancelToken: this.cancelSource.token,
          validateStatus: () => true,
        }
      );

      if (!res.data.success && res.data.error) {
        throw res.data.error;
      }
      this.data = res.data.result;
      this.expanded = true;
      this.emitStatus(
        "complete",
        this.isGoogleSearch
          ? `${this.displayName} queries ready`
          : `${this.displayName} finished`
      );
    } catch (error) {
      if (axios.isCancel(error)) {
        this.error = "Canceled";
        this.emitStatus("canceled", `${this.displayName} canceled`);
      } else {
        this.error = error;
        this.emitStatus("error", `${this.displayName} failed`);
      }
      this.expanded = true;
    }

    this.loading = false;
    this.cancelSource = null;
    this.stopEtaTicker();
  }

  cancelScan(): void {
    if (this.cancelSource !== null) {
      this.cancelSource.cancel("Canceled");
      this.cancelSource = null;
    }
  }

  startEtaTicker(): void {
    this.stopEtaTicker();
    this.etaTimer = window.setInterval(() => {
      if (this.loading) {
        this.emitStatus(
          "running",
          `Running ${this.displayName}`,
          this.remainingEtaMs()
        );
      }
    }, 1000);
  }

  stopEtaTicker(): void {
    if (this.etaTimer !== null) {
      window.clearInterval(this.etaTimer);
      this.etaTimer = null;
    }
  }

  remainingEtaMs(): number {
    if (this.scanStartedAt === 0) {
      return this.estimatedDurationMs;
    }

    return Math.max(0, this.estimatedDurationMs - (Date.now() - this.scanStartedAt));
  }

  emitStatus(status: string, message: string, etaMs?: number): void {
    this.$emit("status", {
      scanId: this.scanId,
      scanner: this.displayName,
      status,
      message,
      etaMs,
    });
  }

  categoryCollapseId(key: string): string {
    return `${this.collapseId}-${key}`;
  }

  isGoogleCategoryExpanded(key: string): boolean {
    return this.googleCategoryExpanded[key] === true;
  }

  toggleGoogleCategory(key: string): void {
    this.$set(
      this.googleCategoryExpanded,
      key,
      !this.isGoogleCategoryExpanded(key)
    );
  }

  isSearchCategoryExpanded(key: string): boolean {
    return this.searchCategoryExpanded[key] === true;
  }

  toggleSearchCategory(key: string): void {
    this.$set(
      this.searchCategoryExpanded,
      key,
      !this.isSearchCategoryExpanded(key)
    );
  }

  actionButtonId(action: string, groupKey: string, index: number): string {
    return `${this.collapseId}-${groupKey}-${index}-${action}`;
  }

  groupMatchCount(rows: SearXNGQueryResult[]): number {
    return this.matchedRows(rows).length;
  }

  matchedRows(rows: SearXNGQueryResult[]): SearXNGQueryResult[] {
    return rows.filter((row) => row.result_count > 0 && !row.error);
  }

  startLauncher(dorks: LaunchableSearch[]): void {
    const queued = dorks.filter((dork) => !this.openedUrls.has(dork.url));
    if (queued.length === 0) {
      return;
    }

    this.stopLauncher();
    this.launcherQueue = queued;
    this.launcherRunning = true;
    this.launchNext();
  }

  stopLauncher(): void {
    if (this.launcherTimer !== null) {
      window.clearTimeout(this.launcherTimer);
      this.launcherTimer = null;
    }
    this.launcherRunning = false;
    this.launcherQueue = [];
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
.scanner-panel-header,
.google-action-group-header {
  cursor: pointer;
}

.scanner-panel-toggle {
  align-items: center;
  background: transparent;
  border: 0;
  color: #212529;
  display: inline-flex;
  font-size: 1.15rem;
  font-weight: 600;
  line-height: 1.2;
  padding: 0;
  text-align: left;
}

.scanner-panel-chevron {
  display: inline-block;
  font-size: 1rem;
  margin-right: 0.5rem;
  width: 1rem;
}

.google-action-group {
  margin-bottom: 1rem;
}

.google-action-group-title {
  min-width: 16rem;
}

.google-query {
  white-space: normal;
  word-break: break-word;
}

::v-deep .google-query-column {
  width: 82%;
}

::v-deep .google-actions-column {
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

.google-action-button {
  align-items: center;
  display: inline-flex;
  height: 2.25rem;
  justify-content: center;
  width: 2.25rem;
}

.searxng-results {
  min-width: 18rem;
}

.searxng-result + .searxng-result {
  border-top: 1px solid #e9ecef;
  margin-top: 0.75rem;
  padding-top: 0.75rem;
}
</style>
