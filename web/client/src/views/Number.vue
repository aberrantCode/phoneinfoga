<template>
  <div>
    <b-card v-if="isLookup || showInformations" no-body class="mb-3 mt-3">
      <b-card-header
        class="bg-white results-section-header"
        @click="informationExpanded = !informationExpanded"
      >
        <button
          class="results-section-toggle"
          type="button"
          :aria-expanded="informationExpanded ? 'true' : 'false'"
          aria-controls="number-information-collapse"
          @click.stop="informationExpanded = !informationExpanded"
        >
          <span aria-hidden="true">{{ informationExpanded ? "-" : "+" }}</span>
          <span>Metadata</span>
        </button>
      </b-card-header>
      <b-collapse
        id="number-information-collapse"
        v-model="informationExpanded"
      >
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
          aria-controls="number-scanners-collapse"
          @click.stop="scannersExpanded = !scannersExpanded"
        >
          <span aria-hidden="true">{{ scannersExpanded ? "-" : "+" }}</span>
          <span>Scanners</span>
        </button>
      </b-card-header>
      <b-collapse id="number-scanners-collapse" v-model="scannersExpanded">
        <b-card-body>
          <Scanner
            v-for="(scanner, index) in scanners"
            ref="scannerRefs"
            :key="index"
            :name="getScannerDisplayName(scanner.name)"
            :scanId="scanner.name"
            :autoRun="true"
            :scan-options="getScannerRunOptions(scanner.name)"
            @status="updateScannerStatus"
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
  getScannerRunOptions,
  ScannerAvailability,
} from "../utils";
import Scanner from "../components/Scanner.vue";
import axios, { AxiosResponse } from "axios";
import config from "@/config";

interface Data {
  loading: boolean;
  isLookup: boolean;
  showInformations: boolean;
  informationExpanded: boolean;
  scannersExpanded: boolean;
  scanners: ScannerAvailability[];
  scannerStatuses: ScannerStatus[];
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
}

export type ScanResponse<T> = AxiosResponse<{
  success: boolean;
  result: T;
  error: string;
}>;

export default Vue.extend({
  components: { Scanner },
  computed: {
    ...mapState(["number"]),
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
    activeScannerCount(): number {
      return this.scannerStatuses.filter((item) => item.status === "running")
        .length;
    },
    finishedScannerCount(): number {
      return this.scannerStatuses.filter((item) =>
        ["complete", "canceled", "error"].includes(item.status)
      ).length;
    },
  },
  data(): Data {
    return {
      loading: false,
      isLookup: false,
      showInformations: false,
      informationExpanded: true,
      scannersExpanded: true,
      scanners: [],
      scannerStatuses: [],
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
    this.runScans();
  },
  methods: {
    getScannerDisplayName,
    getScannerRunOptions,
    async getScanners() {
      try {
        const scannerAvailability = await getScannerAvailability();
        const defaultScannerNames = getDefaultScannerNames(scannerAvailability);
        this.scanners = scannerAvailability.filter(
          (scanner) =>
            scanner.available && defaultScannerNames.includes(scanner.name)
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
    async runScans(): Promise<void> {
      if (!isValid(this.$route.params.number)) {
        this.$store.commit("pushError", { message: "Number is not valid." });
        return;
      }

      this.loading = true;

      this.$store.commit("setNumber", formatNumber(this.$route.params.number));

      try {
        const res = await axios.post(`${config.apiUrl}/v2/numbers`, {
          number: this.$store.state.number,
        });

        this.localData = res.data;

        if (this.localData.valid) {
          this.getScanners();
          this.isLookup = true;
        } else {
          this.showInformations = true;
        }
      } catch (error) {
        this.$store.commit("pushError", { message: error });
      }

      this.loading = false;
    },
  },
});
</script>

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
