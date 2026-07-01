<template>
  <div>
    <!-- Entry state: phone-number field, country selector and Lookup button. In the
         results state these are hidden and replaced by the Start over control (AC6). -->
    <b-form
      v-if="!isResultsState"
      @submit="onSubmit"
      class="d-flex justify-content-center mt-5"
    >
      <b-form-group id="input-group-1" label-for="input-1">
        <b-input-group>
          <VuePhoneNumberInput
            v-model="inputNumberVal"
            :disabled="loading"
            @update="updateInputNumber"
          />
          <b-button
            size="sm"
            variant="dark"
            v-on:click="runScans"
            :disabled="loading"
            class="mr-2 ml-2"
          >
            <b-icon-play-fill></b-icon-play-fill>
            Lookup
          </b-button>

          <b-button
            variant="danger"
            size="sm"
            v-on:click="clearData"
            v-show="number"
            :disabled="loading"
            >Reset
          </b-button>
        </b-input-group>
      </b-form-group>
    </b-form>

    <div
      v-if="isResultsState"
      class="results-controls d-flex justify-content-center mt-5"
    >
      <b-button variant="outline-secondary" size="sm" @click="startOver">
        Start over
      </b-button>
    </div>

    <div v-if="!isLookup && inputNumberValid" class="scanner-selector">
      <div class="scanner-selector-toolbar">
        <span class="scanner-selector-label">Scanners:</span>
        <b-spinner v-if="scannerLoading" small type="grow" />

        <b-form-checkbox-group
          v-model="selectedScannerNames"
          class="scanner-toggle-list"
        >
          <label
            v-for="scanner in selectableScanners"
            :key="scanner.name"
            v-b-tooltip.hover
            :title="scannerToggleTitle(scanner)"
            class="scanner-toggle"
            :class="{
              'scanner-toggle-active':
                scanner.available &&
                selectedScannerNames.includes(scanner.name),
              'scanner-toggle-error': !scanner.available,
            }"
          >
            <input
              v-model="selectedScannerNames"
              type="checkbox"
              :value="scanner.name"
              :disabled="!scanner.available"
            />
            <span class="scanner-toggle-check" aria-hidden="true">
              <b-icon-exclamation-circle-fill v-if="!scanner.available" />
              <b-icon-check v-else />
            </span>
            <span>{{ getScannerDisplayName(scanner.name) }}</span>
          </label>
        </b-form-checkbox-group>
      </div>
      <b-alert v-if="scannerError" show variant="danger" class="mt-2 mb-0">
        {{ scannerError }}
      </b-alert>
      <p
        v-if="!scannerLoading && selectableScanners.length === 0"
        class="mb-0 text-muted"
      >
        No configured scanner plugins are currently available.
      </p>
    </div>

    <b-card
      v-if="isLookup || showInformations || isResultsState"
      no-body
      class="mb-3 mt-3"
    >
      <b-card-header
        class="bg-white results-section-header"
        @click="informationExpanded = !informationExpanded"
      >
        <button
          class="results-section-toggle"
          type="button"
          :aria-expanded="informationExpanded ? 'true' : 'false'"
          aria-controls="scan-information-collapse"
          @click.stop="informationExpanded = !informationExpanded"
        >
          <span aria-hidden="true">{{ informationExpanded ? "-" : "+" }}</span>
          <span>Metadata</span>
        </button>
      </b-card-header>
      <b-collapse id="scan-information-collapse" v-model="informationExpanded">
        <b-card-body>
          <div class="metadata-grid">
            <div
              v-for="item in metadataItems"
              :key="item.label"
              class="metadata-item text-left"
            >
              <span class="metadata-label">{{ item.label }}</span>
              <span class="metadata-value">{{ item.value }}</span>
            </div>
          </div>

          <!-- Request/results record (AC5): time, client IP, scanners, status. -->
          <div v-if="lookupRecordItems.length" class="metadata-record mt-3">
            <h3 class="metadata-record-title text-left">Request record</h3>
            <div class="metadata-grid">
              <div
                v-for="item in lookupRecordItems"
                :key="item.label"
                class="metadata-item text-left"
              >
                <span class="metadata-label">{{ item.label }}</span>
                <span class="metadata-value">{{ item.value }}</span>
              </div>
            </div>
          </div>
        </b-card-body>
      </b-collapse>
    </b-card>

    <b-card v-if="isLookup" no-body class="mb-3">
      <b-card-header
        class="bg-white results-section-header"
        @click="comparisonExpanded = !comparisonExpanded"
      >
        <button
          class="results-section-toggle"
          type="button"
          :aria-expanded="comparisonExpanded ? 'true' : 'false'"
          aria-controls="scan-comparison-collapse"
          @click.stop="comparisonExpanded = !comparisonExpanded"
        >
          <span aria-hidden="true">{{ comparisonExpanded ? "-" : "+" }}</span>
          <span>Provider comparison</span>
        </button>
      </b-card-header>
      <b-collapse id="scan-comparison-collapse" v-model="comparisonExpanded">
        <b-card-body>
          <ProviderComparison
            :baseline="localData"
            :results="comparisonResults"
          />
        </b-card-body>
      </b-collapse>
    </b-card>

    <b-card v-if="isLookup" no-body class="mb-3 scan-status-panel">
      <b-card-header
        class="bg-white d-flex align-items-center justify-content-between"
      >
        <div class="text-left">
          <h2 class="h6 mb-0">Scan status</h2>
          <p class="text-muted mb-0">
            {{ activeScannerCount }} running,
            {{ finishedScannerCount }} finished
          </p>
        </div>
        <b-button
          v-if="activeScannerCount > 0"
          variant="outline-danger"
          size="sm"
          @click="cancelRunningScanners"
        >
          Cancel running
        </b-button>
      </b-card-header>
      <b-card-body>
        <div class="scan-status-list">
          <div
            v-for="status in scannerStatuses"
            :key="status.scanId"
            class="scan-status-item"
          >
            <b-badge :variant="statusVariant(status.status)" class="mr-2">
              {{ statusLabel(status.status) }}
            </b-badge>
            <span class="scan-status-name">{{ status.scanner }}</span>
            <span class="text-muted">{{ status.message }}</span>
            <span
              v-if="status.status === 'running' && status.etaMs !== undefined"
              class="text-muted"
            >
              ETA {{ formatDuration(status.etaMs) }}
            </span>
          </div>
        </div>
      </b-card-body>
    </b-card>

    <b-card v-if="isLookup" no-body class="text-center">
      <b-card-header
        class="bg-white results-section-header"
        @click="scannersExpanded = !scannersExpanded"
      >
        <button
          class="results-section-toggle"
          type="button"
          :aria-expanded="scannersExpanded ? 'true' : 'false'"
          aria-controls="scan-scanners-collapse"
          @click.stop="scannersExpanded = !scannersExpanded"
        >
          <span aria-hidden="true">{{ scannersExpanded ? "-" : "+" }}</span>
          <span>Scanners</span>
        </button>
      </b-card-header>
      <b-collapse id="scan-scanners-collapse" v-model="scannersExpanded">
        <b-card-body>
          <Scanner
            v-for="(scanner, index) in scanners"
            ref="scannerRefs"
            :key="index"
            :name="getScannerDisplayName(scanner.name)"
            :scanId="scanner.name"
            :autoRun="true"
            :auto-expand-on-data="!isComparisonProvider(scanner.name)"
            :scan-options="getScannerRunOptions(scanner.name)"
            :metadata="localData"
            :lookup-id="activeLookupId"
            @status="updateScannerStatus"
            @result="captureScannerResult"
          />
        </b-card-body>
      </b-collapse>
    </b-card>
  </div>
</template>

<script lang="ts">
import Vue from "vue";
import { mapState } from "vuex";
import {
  formatNumber,
  isValid,
  getDefaultScannerNames,
  getScannerAvailability,
  getScannerDisplayName,
  getScannerDescription,
  getScannerRunOptions,
  createLookup,
  closeLookup,
  ScannerAvailability,
  LookupDetail,
  CreateLookupResult,
} from "../utils";
import VuePhoneNumberInput from "vue-phone-number-input";
import Scanner from "../components/Scanner.vue";
import ProviderComparison from "../components/ProviderComparison.vue";
import axios, { AxiosResponse } from "axios";
import config from "@/config";

// Renders an RFC3339 timestamp in the viewer's locale, falling back to the raw
// value if it can't be parsed.
const formatTimestamp = (iso: string): string => {
  if (!iso) {
    return "";
  }
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleString();
};

const titleCase = (value: string): string =>
  value ? value.charAt(0).toUpperCase() + value.slice(1) : "";

// Scanners whose results share the validation-field vocabulary and therefore
// belong in the provider-comparison matrix.
const COMPARISON_PROVIDERS = [
  "numverify",
  "veriphone",
  "numlookupapi",
  "abstract",
  "twilio",
];

interface InputNumberObject {
  countryCallingCode: string;
  countryCode: string;
  e164: string;
  formatInternational: string;
  formatNational: string;
  formattedNumber: string;
  isValid: boolean;
  nationalNumber: string;
  phoneNumber: string;
  type: string;
  uri: string;
}

interface Data {
  loading: boolean;
  scannerLoading: boolean;
  scannerError: string;
  viewState: ViewState;
  isLookup: boolean;
  showInformations: boolean;
  informationExpanded: boolean;
  scannersExpanded: boolean;
  comparisonExpanded: boolean;
  inputNumber: string;
  inputNumberVal: string;
  inputNumberValid: boolean;
  scanEvent: Vue;
  scannerAvailability: ScannerAvailability[];
  selectedScannerNames: string[];
  scanners: ScannerAvailability[];
  scannerStatuses: ScannerStatus[];
  scannerFailures: { [name: string]: string };
  scannerResults: { [name: string]: unknown };
  activeLookupId: string;
  lookupClosed: boolean;
  localData: {
    valid: boolean;
    raw_local: string;
    local: string;
    e164: string;
    international: string;
    countryCode: number;
    country: string;
    carrier: string;
  };
}

interface ScannerStatus {
  scanId: string;
  scanner: string;
  status: string;
  message: string;
  etaMs?: number;
  error?: string;
}

// The page is a two-state flow: `entry` (number field + scanner picker) and
// `results` (rendered lookup). Results has two sources: `fresh` (just ran) or
// `replay` (loaded from history). activeLookup holds the replayed record, if any.
type ViewStateName = "entry" | "results";
type ViewStateSource = "fresh" | "replay";

// A fresh lookup starts from a CreateLookupResult (record only); a replayed lookup
// carries the full LookupDetail. Both expose the fields the record panel needs.
type ActiveLookup = LookupDetail | CreateLookupResult;

interface ViewState {
  state: ViewStateName;
  source: ViewStateSource;
  activeLookup: ActiveLookup | null;
}

export type ScanResponse<T> = AxiosResponse<{
  success: boolean;
  result: T;
  error: string;
}>;

export default Vue.extend({
  components: { Scanner, ProviderComparison, VuePhoneNumberInput },
  data(): Data {
    return {
      loading: false,
      scannerLoading: false,
      scannerError: "",
      viewState: { state: "entry", source: "fresh", activeLookup: null },
      isLookup: false,
      showInformations: false,
      informationExpanded: true,
      scannersExpanded: true,
      comparisonExpanded: true,
      inputNumber: "",
      inputNumberVal: "",
      inputNumberValid: false,
      scanEvent: new Vue(),
      scannerAvailability: [],
      selectedScannerNames: [],
      scanners: [],
      scannerStatuses: [],
      scannerFailures: {},
      scannerResults: {},
      activeLookupId: "",
      lookupClosed: false,
      localData: {
        valid: false,
        raw_local: "",
        local: "",
        e164: "",
        international: "",
        countryCode: 33,
        country: "",
        carrier: "",
      },
    };
  },
  mounted() {
    this.loadScannerAvailability();
    window.addEventListener(
      "scanner-preferences-updated",
      this.loadScannerAvailability
    );
  },
  beforeDestroy() {
    window.removeEventListener(
      "scanner-preferences-updated",
      this.loadScannerAvailability
    );
  },
  computed: {
    ...mapState(["number"]),
    isEntryState(): boolean {
      return this.viewState.state === "entry";
    },
    isResultsState(): boolean {
      return this.viewState.state === "results";
    },
    isReplay(): boolean {
      return (
        this.viewState.state === "results" && this.viewState.source === "replay"
      );
    },
    // The request/results record shown alongside the number metadata (AC5): when a
    // lookup is active, surface its time, client IP, requested scanners and status.
    lookupRecordItems(): Array<{ label: string; value: string }> {
      const lookup = this.viewState.activeLookup;
      if (!lookup) {
        return [];
      }

      const items = [
        { label: "Lookup time", value: formatTimestamp(lookup.createdAt) },
        { label: "Client IP", value: lookup.clientIp || "" },
        {
          label: "Scanners requested",
          value: (lookup.scannersRequested || []).join(", "),
        },
        { label: "Status", value: titleCase(lookup.status) },
      ];

      return items.filter((item) => item.value !== "");
    },
    metadataItems(): Array<{ label: string; value: string }> {
      const items = [
        { label: "Valid", value: this.localData.valid ? "Yes" : "No" },
        { label: "E.164", value: this.localData.e164 },
        { label: "International", value: this.localData.international },
        { label: "Local", value: this.localData.local },
        { label: "Country", value: this.localData.country },
        {
          label: "Calling code",
          value: String(this.localData.countryCode || ""),
        },
        { label: "Carrier", value: this.localData.carrier },
      ];

      const seen: { [key: string]: boolean } = {};
      return items.filter((item) => {
        if (item.value === "" || item.value === "0") {
          return false;
        }
        const key = `${item.label}:${item.value}`;
        if (seen[key]) {
          return false;
        }
        seen[key] = true;
        return true;
      });
    },
    selectableScanners(): ScannerAvailability[] {
      // Show every configured scanner. A scanner is unavailable if the dryrun
      // probe failed, OR if a previous real run surfaced an error (credentials
      // / exception we couldn't know until it actually ran). Group the ones
      // that can't run last so they read as disabled extras.
      const decorated = this.scannerAvailability.map((scanner) => {
        const runError = this.scannerFailures[scanner.name];
        if (scanner.available && runError) {
          return { ...scanner, available: false, error: runError };
        }
        return scanner;
      });
      const available = decorated.filter((s) => s.available);
      const unavailable = decorated.filter((s) => !s.available);
      return [...available, ...unavailable];
    },
    activeScannerCount(): number {
      return this.scannerStatuses.filter((item) => item.status === "running")
        .length;
    },
    finishedScannerCount(): number {
      return this.scannerStatuses.filter((item) =>
        ["complete", "canceled", "error"].includes(item.status)
      ).length;
    },
    comparisonResults(): { [name: string]: unknown } {
      return COMPARISON_PROVIDERS.reduce(
        (acc: { [name: string]: unknown }, name) => {
          if (this.scannerResults[name] !== undefined) {
            acc[name] = this.scannerResults[name];
          }
          return acc;
        },
        {}
      );
    },
  },
  methods: {
    getScannerDisplayName,
    getScannerDescription,
    getScannerRunOptions,
    isComparisonProvider(name: string): boolean {
      return COMPARISON_PROVIDERS.includes(name);
    },
    captureScannerResult(payload: { scanId: string; data: unknown }): void {
      // Immutable update so the comparisonResults computed re-evaluates.
      this.scannerResults = {
        ...this.scannerResults,
        [payload.scanId]: payload.data,
      };
    },
    scannerToggleTitle(scanner: ScannerAvailability): string {
      if (!scanner.available) {
        return scanner.error
          ? `Unavailable — ${scanner.error}`
          : "Unavailable — this scanner can't run (missing credentials or configuration).";
      }
      return getScannerDescription(scanner.name);
    },
    // enterResults / enterEntry are the two view-state transitions. Phases 6-7 use
    // them to drive the fresh-lookup and replay flows; clearData resets to entry.
    enterResults(
      source: ViewStateSource,
      lookup: ActiveLookup | null = null
    ): void {
      this.viewState = { state: "results", source, activeLookup: lookup };
    },
    enterEntry(): void {
      this.viewState = { state: "entry", source: "fresh", activeLookup: null };
    },
    // startOver resets the whole page back to the entry state and clears the number
    // field so the user can look up a different number (AC6).
    startOver(): void {
      this.clearData();
      this.inputNumber = "";
      this.inputNumberVal = "";
      this.inputNumberValid = false;
    },
    clearData() {
      this.enterEntry();
      this.isLookup = false;
      this.showInformations = false;
      this.informationExpanded = true;
      this.scannersExpanded = true;
      this.comparisonExpanded = true;
      this.scannerStatuses = [];
      this.scannerResults = {};
      this.activeLookupId = "";
      this.lookupClosed = false;
      this.$store.commit("resetState");
    },
    async loadScannerAvailability(): Promise<void> {
      this.scannerLoading = true;
      this.scannerError = "";
      // Re-probing (mount, or after credentials/preferences change) clears any
      // remembered run failures so a fixed scanner can recover.
      this.scannerFailures = {};
      try {
        this.scannerAvailability = await getScannerAvailability();
        this.selectedScannerNames = getDefaultScannerNames(
          this.scannerAvailability
        );
      } catch (error) {
        this.scannerError = String(error);
      }
      this.scannerLoading = false;
    },
    async runScans(): Promise<void> {
      this.clearData();
      if (!isValid(this.inputNumber)) {
        this.$store.commit("pushError", { message: "Number is not valid." });
        return;
      }

      this.loading = true;

      this.$store.commit("setNumber", formatNumber(this.inputNumber));

      try {
        const res = await axios.post(`${config.apiUrl}/v2/numbers`, {
          number: this.$store.state.number,
        });

        this.localData = res.data;

        if (this.localData.valid) {
          await this.startFreshLookup();
        } else {
          this.showInformations = true;
        }
      } catch (error) {
        this.$store.commit("pushError", { message: error });
      }

      this.loading = false;
    },
    // startFreshLookup records the request (createLookup) BEFORE the scanners mount,
    // so each per-scanner /run carries the lookupId and its result is persisted (AC1/AC2).
    // Persistence is best-effort: a createLookup failure still renders live results.
    async startFreshLookup(): Promise<void> {
      await this.getScanners();
      const scannerNames = this.scanners.map((scanner) => scanner.name);

      try {
        const record = await createLookup(
          this.$store.state.number,
          scannerNames
        );
        this.activeLookupId = record.id;
        this.enterResults("fresh", record);
      } catch (error) {
        this.activeLookupId = "";
        this.enterResults("fresh", null);
      }

      // Rendering the scanners (autoRun) starts the per-scanner runs with the lookupId.
      this.isLookup = true;
    },
    // maybeCloseLookup finalizes the lookup exactly once, after every scanner has
    // reached a terminal state (AC3). Closing is best-effort.
    maybeCloseLookup(): void {
      if (
        !this.activeLookupId ||
        this.lookupClosed ||
        this.scanners.length === 0
      ) {
        return;
      }

      const settled = this.scannerStatuses.filter((status) =>
        ["complete", "error", "canceled"].includes(status.status)
      ).length;
      if (settled < this.scanners.length) {
        return;
      }

      this.lookupClosed = true;
      closeLookup(this.activeLookupId)
        .then((summary) => {
          const current = this.viewState.activeLookup;
          if (current) {
            this.viewState = {
              ...this.viewState,
              activeLookup: { ...current, status: summary.status },
            };
          }
        })
        .catch(() => {
          // Availability over durability: a failed close must not break the UI.
        });
    },
    onSubmit(evt: Event) {
      evt.preventDefault();
    },
    updateInputNumber(val: InputNumberObject) {
      this.inputNumber = val.e164;
      this.inputNumberValid = Boolean(val.isValid);
    },
    async getScanners() {
      try {
        if (this.scannerAvailability.length === 0) {
          await this.loadScannerAvailability();
        }

        this.scanners = this.scannerAvailability.filter(
          (scanner) =>
            scanner.available &&
            !this.scannerFailures[scanner.name] &&
            this.selectedScannerNames.includes(scanner.name)
        );
        this.scannerStatuses = this.scanners.map((scanner) => ({
          scanId: scanner.name,
          scanner: getScannerDisplayName(scanner.name),
          status: "queued",
          message: "Waiting to start",
        }));
      } catch (error) {
        this.$store.commit("pushError", { message: error });
      }
    },
    updateScannerStatus(status: ScannerStatus): void {
      const index = this.scannerStatuses.findIndex(
        (item) => item.scanId === status.scanId
      );
      if (index >= 0) {
        this.$set(this.scannerStatuses, index, status);
      } else {
        this.scannerStatuses.push(status);
      }

      // Remember scanners that fail a real run so the selection panel can
      // disable them with the specific failure message. A later success clears
      // the record (e.g. a retry succeeded).
      if (status.status === "error") {
        this.scannerFailures = {
          ...this.scannerFailures,
          [status.scanId]: status.error || status.message,
        };
      } else if (
        status.status === "complete" &&
        this.scannerFailures[status.scanId]
      ) {
        const rest = { ...this.scannerFailures };
        delete rest[status.scanId];
        this.scannerFailures = rest;
      }

      this.maybeCloseLookup();
    },
    cancelRunningScanners(): void {
      const refs = this.$refs.scannerRefs as Vue[] | Vue | undefined;
      const scanners = Array.isArray(refs) ? refs : refs ? [refs] : [];
      scanners.forEach((scanner) => {
        const maybeScanner = scanner as Vue & { cancelScan?: () => void };
        if (typeof maybeScanner.cancelScan === "function") {
          maybeScanner.cancelScan();
        }
      });
    },
    statusVariant(status: string): string {
      if (status === "running") {
        return "primary";
      }
      if (status === "complete") {
        return "success";
      }
      if (status === "error") {
        return "danger";
      }
      if (status === "canceled") {
        return "warning";
      }
      return "secondary";
    },
    statusLabel(status: string): string {
      return status.charAt(0).toUpperCase() + status.slice(1);
    },
    formatDuration(ms: number): string {
      const seconds = Math.max(0, Math.ceil(ms / 1000));
      const minutes = Math.floor(seconds / 60);
      const remainder = seconds % 60;

      if (minutes === 0) {
        return `${remainder}s`;
      }

      return `${minutes}m ${remainder}s`;
    },
  },
});
</script>

<style src="vue-phone-number-input/dist/vue-phone-number-input.css"></style>
<style scoped>
.results-section-header {
  cursor: pointer;
}

.results-section-toggle {
  align-items: center;
  background: transparent;
  border: 0;
  color: var(--ink);
  display: inline-flex;
  font-size: 1.15rem;
  font-weight: 600;
  gap: 0.5rem;
  line-height: 1.2;
  padding: 0;
}

.scanner-selector {
  margin-left: auto;
  margin-right: auto;
  margin-top: 2rem;
  max-width: 46rem;
}

.scanner-selector-toolbar {
  align-items: center;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  justify-content: center;
}

.scanner-selector-label {
  color: var(--ink-soft);
  font-weight: 600;
  letter-spacing: 0.04em;
}

.scanner-toggle-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  justify-content: center;
  width: 100%;
}

/* A scanner toggle reads as a recessed well, like the phone-number field. */
.scanner-toggle {
  align-items: center;
  background: color-mix(in oklch, var(--bg) 60%, var(--surface-1));
  border: 1px solid var(--rule);
  border-radius: var(--ac-radius-md);
  color: var(--ink);
  cursor: pointer;
  display: inline-flex;
  font-weight: 500;
  gap: 0.45rem;
  line-height: 1;
  margin: 0;
  min-height: 2rem;
  padding: 0.45rem 0.75rem;
  transition: background-color 0.12s ease, border-color 0.12s ease,
    color 0.12s ease;
  user-select: none;
}

.scanner-toggle:hover {
  border-color: var(--accent);
}

.scanner-toggle input {
  height: 1px;
  opacity: 0;
  position: absolute;
  width: 1px;
}

.scanner-toggle-check {
  align-items: center;
  border: 1px solid var(--rule);
  border-radius: 0.2rem;
  color: transparent;
  display: inline-flex;
  height: 1rem;
  justify-content: center;
  width: 1rem;
}

/* Selected scanner — keeps the exact same border as the phone-number field and
   Lookup button (--rule at rest, --accent on hover from the base rules); the
   selected state is conveyed by the checkbox tick, not a special border. */
.scanner-toggle-active {
  background: color-mix(in oklch, var(--bg) 60%, var(--surface-1));
  color: var(--ink);
}

/* The checked box matches the phone-number field: a neutral recessed well with
   a hairline --rule border. The tick stays a quiet de-energised phosphor so the
   selected state is still legible without out-shouting the field. */
.scanner-toggle-active .scanner-toggle-check {
  background: color-mix(in oklch, var(--bg) 60%, var(--surface-1));
  border-color: var(--rule);
  color: var(--accent-dim);
}

/* A scanner that can't run (no credentials / dryrun failure): disabled, with an
   error-tinted border and a muted label. The tooltip carries the failure message. */
.scanner-toggle-error,
.scanner-toggle-error:hover {
  background: color-mix(in oklch, var(--led-down) 8%, var(--surface-1));
  border-color: color-mix(in oklch, var(--led-down) 60%, transparent);
  color: var(--muted);
  cursor: not-allowed;
}

.scanner-toggle-error .scanner-toggle-check {
  background: transparent;
  border-color: color-mix(in oklch, var(--led-down) 55%, transparent);
  color: var(--led-down);
}

.metadata-grid {
  display: grid;
  gap: 0.75rem 1rem;
  grid-template-columns: repeat(auto-fit, minmax(13rem, 1fr));
}

.metadata-item {
  border-bottom: 1px solid var(--rule-soft);
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding-bottom: 0.5rem;
}

.metadata-label {
  color: var(--muted);
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.metadata-value {
  overflow-wrap: anywhere;
}

.metadata-record-title {
  color: var(--ink-soft);
  font-size: 0.9rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  margin-bottom: 0.5rem;
}

.scan-status-list {
  display: grid;
  gap: 0.5rem;
}

.scan-status-item {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  text-align: left;
}

.scan-status-name {
  font-weight: 600;
}
</style>
