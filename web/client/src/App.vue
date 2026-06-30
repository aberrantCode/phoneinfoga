<template>
  <div id="app" style="padding-bottom: 50px">
    <div>
      <b-navbar toggleable="lg" type="dark" variant="dark">
        <b-container>
          <b-navbar-brand to="/" class="ac-app-brand">
            <span class="ac-brand-logo">
              <img src="@/assets/logo.svg" width="48" height="48" alt="logo" />
            </span>
            <span class="ac-app-brand-text">
              <span class="ac-title"
                >{{ config.appName }} <span class="ac-title__sub">OSINT</span></span
              >
              <span class="ac-app-brand-desc ac-mono">{{
                config.appDescription
              }}</span>
            </span>
          </b-navbar-brand>

          <b-navbar-nav class="ml-auto">
            <button
              class="ac-tool ml-2"
              type="button"
              :aria-pressed="themeResolved === 'light' ? 'true' : 'false'"
              :title="themeToggleTitle"
              :aria-label="themeToggleTitle"
              @click="toggleTheme"
            >
              <svg
                class="i-light"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
              >
                <circle cx="12" cy="12" r="4" />
                <path
                  d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"
                />
              </svg>
              <svg
                class="i-dark"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
              >
                <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" />
              </svg>
            </button>
            <button
              id="scanner-preferences-button"
              class="ac-tool ml-2"
              type="button"
              aria-label="Scanner preferences"
              @click="$bvModal.show('scanner-preferences-modal')"
            >
              <b-icon-gear-fill aria-hidden="true" />
            </button>
            <b-tooltip target="scanner-preferences-button">
              Scanner preferences
            </b-tooltip>
            <scanner-credentials />
          </b-navbar-nav>
        </b-container>
      </b-navbar>
      <!-- Energized rail — the powered-bus signature cue, under the masthead. -->
      <b-container>
        <div class="ac-rail" role="presentation"></div>
      </b-container>
    </div>

    <b-modal
      id="scanner-preferences-modal"
      title="Scanner Preferences"
      ok-title="Save"
      @show="loadScannerPreferences"
      @ok="saveScannerPreferences"
    >
      <p class="text-muted">
        Choose which configured scanners are selected by default on the lookup
        page.
      </p>
      <b-alert v-if="scannerPreferencesError" show variant="danger">
        {{ scannerPreferencesError }}
      </b-alert>
      <b-spinner v-if="scannerPreferencesLoading" small type="grow" />
      <b-form-checkbox-group
        v-else
        v-model="preferredScannerNames"
        stacked
      >
        <b-form-checkbox
          v-for="scanner in scannerPreferences"
          :key="scanner.name"
          :value="scanner.name"
          :disabled="!scanner.available"
          class="mb-2"
        >
          <span :class="{ 'text-muted': !scanner.available }">
            {{ getScannerDisplayName(scanner.name) }}
          </span>
          <small v-if="!scanner.available" class="text-muted">
            {{ scannerUnavailableText(scanner.error) }}
          </small>
        </b-form-checkbox>
      </b-form-checkbox-group>

      <hr />

      <b-form-group
        label="SearXNG delay between queries"
        label-for="searxng-delay-ms"
        description="Applied when PhoneInfoga runs SearXNG queries. Increase this if your SearXNG instance or upstream engines need a slower request pace."
      >
        <b-input-group append="ms">
          <b-form-input
            id="searxng-delay-ms"
            v-model.number="searxngDelayMs"
            type="number"
            min="0"
            step="250"
          />
        </b-input-group>
      </b-form-group>
    </b-modal>

    <b-container class="my-md-3">
      <b-row>
        <b-col cols="12">
          <b-alert v-if="isDemo" show variant="warning" fade
            >Welcome to the demo of PhoneInfoga web client.</b-alert
          >
          <b-alert
            v-for="(err, i) in errors"
            v-bind:key="i"
            show
            variant="danger"
            dismissible
            fade
            >{{ err.message }}</b-alert
          >

          <router-view />
        </b-col>
      </b-row>
    </b-container>

    <!-- Footer status strip — uppercase mono telemetry, dot-separated. -->
    <footer class="ac-app-footer">
      <b-container>
        <div class="ac-rail" role="presentation"></div>
        <div class="ac-statusbar mt-3">
          <span>{{ config.appName }}</span>
          <span class="ac-statusbar__sep">·</span>
          <span>{{ themeResolved === "light" ? "Light" : "Dark" }}</span>
          <span class="ac-statusbar__sep">·</span>
          <span>Version</span>
          <a
            class="ac-statusbar__value"
            href="https://github.com/sundowndev/phoneinfoga/releases"
            target="_blank"
            >{{ version || "dev" }}</a
          >
        </div>
      </b-container>
    </footer>
    <script
      v-if="isDemo"
      type="application/javascript"
      defer
      data-domain="demo.phoneinfoga.crvx.fr"
      src="https://analytics.crvx.fr/js/script.js"
    ></script>
  </div>
</template>

<script lang="ts">
import Vue from "vue";
import { mapState } from "vuex";
import config from "@/config";
import axios, { AxiosResponse } from "axios";
import ScannerCredentials from "@/components/ScannerCredentials.vue";
import {
  getDefaultScannerNames,
  getScannerAvailability,
  getScannerDisplayName,
  getSearXNGDelayMs,
  setPreferredScannerNames,
  setSearXNGDelayMs,
  ScannerAvailability,
} from "@/utils";

type HealthResponse = { success: boolean; version: string; demo: boolean };

// Globals exposed by the AC theme module inlined in public/index.html.
type AcThemeWindow = Window & {
  __acGetTheme?: () => string;
  __acGetResolvedTheme?: () => string;
  __acSetTheme?: (mode: string) => void;
};

export default Vue.extend({
  components: { ScannerCredentials },
  data: () => ({
    config,
    version: "",
    isDemo: false,
    scannerPreferences: [] as ScannerAvailability[],
    preferredScannerNames: [] as string[],
    scannerPreferencesLoading: false,
    scannerPreferencesError: "",
    searxngDelayMs: 750,
    themeMode: "dark",
    themeResolved: "dark",
  }),
  computed: {
    ...mapState(["number", "errors"]),
    themeToggleTitle(): string {
      return this.themeResolved === "light"
        ? "Switch to dark theme"
        : "Switch to light theme";
    },
  },
  async created() {
    const res: AxiosResponse<HealthResponse> = await axios.get(config.apiUrl);

    this.version = res.data.version;
    this.isDemo = res.data.demo;
  },
  mounted() {
    this.syncTheme();
    window.addEventListener("ac-theme-changed", this.syncTheme);
  },
  beforeDestroy() {
    window.removeEventListener("ac-theme-changed", this.syncTheme);
  },
  methods: {
    getScannerDisplayName,
    syncTheme(): void {
      const w = window as AcThemeWindow;
      this.themeMode =
        typeof w.__acGetTheme === "function" ? w.__acGetTheme() : "dark";
      this.themeResolved =
        typeof w.__acGetResolvedTheme === "function"
          ? w.__acGetResolvedTheme()
          : "dark";
    },
    toggleTheme(): void {
      const w = window as AcThemeWindow;
      const next = this.themeResolved === "light" ? "dark" : "light";
      if (typeof w.__acSetTheme === "function") {
        w.__acSetTheme(next);
      }
      this.syncTheme();
    },
    scannerUnavailableText(error?: string): string {
      return error ? `Unavailable: ${error}` : "Unavailable";
    },
    async loadScannerPreferences(): Promise<void> {
      this.scannerPreferencesLoading = true;
      this.scannerPreferencesError = "";
      try {
        this.scannerPreferences = await getScannerAvailability();
        this.preferredScannerNames = getDefaultScannerNames(
          this.scannerPreferences
        );
        this.searxngDelayMs = getSearXNGDelayMs();
      } catch (error) {
        this.scannerPreferencesError = String(error);
      }
      this.scannerPreferencesLoading = false;
    },
    saveScannerPreferences(): void {
      const availableNames = this.scannerPreferences
        .filter((scanner) => scanner.available)
        .map((scanner) => scanner.name);
      setPreferredScannerNames(
        this.preferredScannerNames.filter((name) =>
          availableNames.includes(name)
        )
      );
      setSearXNGDelayMs(this.searxngDelayMs);
      window.dispatchEvent(new CustomEvent("scanner-preferences-updated"));
    },
  },
});
</script>

<style scoped>
.ac-app-brand {
  align-items: center;
  display: inline-flex;
  gap: var(--ac-space-4);
  margin-right: var(--ac-space-4);
  padding: 0;
}

.ac-app-brand-text {
  display: flex;
  flex-direction: column;
  line-height: 1.15;
}

.ac-app-brand-desc {
  color: var(--muted);
  font-size: 0.7rem;
  margin-top: 0.2rem;
  white-space: normal;
}

.ac-app-footer {
  margin-top: var(--ac-space-8);
  padding-bottom: 2rem;
}

/* Keep the right-aligned masthead tools (theme, preferences, credentials)
   the same size and vertically centered. */
.navbar-nav.ml-auto {
  align-items: center;
}
</style>
