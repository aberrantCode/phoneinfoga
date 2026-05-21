<template>
  <div id="app" style="padding-bottom: 50px">
    <div>
      <b-navbar toggleable="lg" type="dark" variant="dark">
        <b-container>
          <b-navbar-brand to="/">
            <img
              src="@/assets/logo.svg"
              class="d-inline-block align-top"
              width="30"
              height="30"
              alt="logo"
            />
            {{ config.appName }}
          </b-navbar-brand>

          <b-collapse id="nav-text-collapse" is-nav>
            <b-navbar-nav>
              <b-nav-text>{{ config.appDescription }}</b-nav-text>
            </b-navbar-nav>
          </b-collapse>

          <b-navbar-nav class="ml-auto">
            <b-collapse id="nav-collapse" is-nav>
              <b-navbar-nav>
                <b-nav-item
                  href="https://github.com/sundowndev/phoneinfoga"
                  target="_blank"
                  >GitHub</b-nav-item
                >
                <b-nav-item
                  href="https://sundowndev.github.io/phoneinfoga/resources/"
                  target="_blank"
                  >Resources</b-nav-item
                >
                <b-nav-item
                  href="https://sundowndev.github.io/phoneinfoga/"
                  target="_blank"
                  >Documentation</b-nav-item
                >
              </b-navbar-nav>
            </b-collapse>
            <b-button
              id="scanner-preferences-button"
              variant="outline-light"
              size="sm"
              class="ml-2"
              aria-label="Scanner preferences"
              @click="$bvModal.show('scanner-preferences-modal')"
            >
              <b-icon-gear-fill aria-hidden="true" />
            </b-button>
            <b-tooltip target="scanner-preferences-button">
              Scanner preferences
            </b-tooltip>
          </b-navbar-nav>
        </b-container>
      </b-navbar>
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

    <b-navbar
      toggleable="lg"
      type="light"
      variant="light"
      fixed="bottom"
      v-if="version !== ''"
    >
      <b-container>
        <b-navbar-nav class="ml-auto">
          <b-collapse id="nav-collapse" is-nav>
            <b-navbar-nav>
              <b-nav-item
                href="https://github.com/sundowndev/phoneinfoga/releases"
                target="_blank"
                >{{ config.appName }} {{ version }}</b-nav-item
              >
            </b-navbar-nav>
          </b-collapse>
        </b-navbar-nav>
      </b-container>
    </b-navbar>
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

export default Vue.extend({
  data: () => ({
    config,
    version: "",
    isDemo: false,
    scannerPreferences: [] as ScannerAvailability[],
    preferredScannerNames: [] as string[],
    scannerPreferencesLoading: false,
    scannerPreferencesError: "",
    searxngDelayMs: 750,
  }),
  computed: {
    ...mapState(["number", "errors"]),
  },
  async created() {
    const res: AxiosResponse<HealthResponse> = await axios.get(config.apiUrl);

    this.version = res.data.version;
    this.isDemo = res.data.demo;
  },
  methods: {
    getScannerDisplayName,
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
