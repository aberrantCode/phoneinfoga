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
          v-b-tooltip.hover
          :title="scannerDescription"
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
            v-if="isGoogleSearch && hasData && !searchLauncherRunning"
            @click.stop="openAllSearches"
            variant="outline-primary"
            size="sm"
            class="mr-2"
            >Open All</b-button
          >
          <b-button
            v-if="isSearXNGSearch && hasData && !searchLauncherRunning"
            @click.stop="openMatchedSearches"
            variant="outline-primary"
            size="sm"
            class="mr-2"
            >Open Matches</b-button
          >
          <b-button
            v-if="(isGoogleSearch || isSearXNGSearch) && searchLauncherRunning"
            @click.stop="stopSearches"
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

        <SearchActionGroups
          v-else-if="
            (isGoogleSearch || isSearXNGSearch || isSerpApi) && hasData
          "
          ref="searchGroups"
          :mode="searchGroupsMode"
          :data="data"
          :id-prefix="collapseId"
          @launcher-change="searchLauncherRunning = $event"
        />

        <ScannerSummary
          v-else-if="isValidationLike && hasData"
          :badge="validationBadge"
          :headline="validationHeadline"
          :subtext="validationSubtext"
          :groups="validationGroups"
          :col-md="4"
        />

        <div v-else-if="isTwilio && hasData">
          <ScannerSummary
            :badge="twilioBadge"
            :headline="twilioHeadline"
            :subtext="twilioSubtext"
            :groups="twilioGroups"
            :col-md="4"
          />
          <b-alert
            v-for="(note, index) in twilioNotes"
            :key="index"
            show
            variant="light"
            class="twilio-note mb-0 mt-2"
          >
            {{ note }}
          </b-alert>
        </div>

        <ScannerSummary
          v-else-if="isOvh && hasData"
          :badge="ovhBadge"
          :groups="ovhGroups"
          :col-md="6"
          empty-text="No additional location details were returned."
        />

        <ScannerSummary
          v-else-if="isLocal && hasData && hasMetadata && !localHasChanges"
          :badge="localBadge"
        >
          <b-alert show variant="light" class="local-banner mb-0">
            No new information &mdash; these values match the Metadata panel
            above.
          </b-alert>
        </ScannerSummary>

        <ScannerSummary
          v-else-if="isLocal && hasData"
          :badge="localBadge"
          :groups="localGroups"
          :col-md="6"
        />

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
import ScannerSummary, {
  SummaryBadge,
  SummaryGroup,
} from "./ScannerSummary.vue";
import SearchActionGroups from "./SearchActionGroups.vue";
import { getScannerDescription } from "@/utils";
import config from "@/config";

// Capitalises a provider enum like "mobile" or "voip" for display; blank in,
// blank out.
const titleCase = (value: string | undefined): string => {
  if (!value) {
    return "";
  }
  return value.charAt(0).toUpperCase() + value.slice(1);
};

// Shared by Numverify and the ValidationScannerResponse providers (Veriphone,
// NumlookupAPI, Abstract), which all return these field names.
interface ValidationResult {
  valid: boolean;
  number?: string;
  local_format?: string;
  international_format?: string;
  country_prefix?: string;
  country_code?: string;
  country_name?: string;
  location?: string;
  carrier?: string;
  line_type?: string;
}

interface TwilioResult {
  valid: boolean;
  national_format?: string;
  line_type?: string;
  carrier_name?: string;
  mobile_country_code?: string;
  mobile_network_code?: string;
  caller_name?: string;
  caller_type?: string;
  sim_swap_last_date?: string;
  sim_swap_in_period?: boolean;
  call_forwarding_status?: string;
  notes?: string[];
}

interface NumverifyRow {
  label: string;
  value: string;
}

interface NumverifyGroup {
  key: string;
  label: string;
  rows: NumverifyRow[];
}

interface OVHResult {
  found: boolean;
  number_range?: string;
  city?: string;
  zip_code?: string;
}

interface LocalScannerData {
  raw_local?: string;
  local?: string;
  e164?: string;
  international?: string;
  country_code?: number;
  country?: string;
  carrier?: string;
}

// Shape returned by /v2/numbers (Scan.vue's localData), used to detect
// whether the local scanner adds anything beyond the Metadata panel.
interface LocalMetadata {
  valid?: boolean;
  rawLocal?: string;
  local?: string;
  e164?: string;
  international?: string;
  countryCode?: number;
  country?: string;
  carrier?: string;
}

interface LocalRow {
  label: string;
  value: string;
  changed: boolean;
}

@Component({
  components: {
    JsonViewer,
    ScannerSummary,
    SearchActionGroups,
  },
})
export default class Scanner extends Vue {
  data: unknown = null;
  loading = false;
  dryrunError = false;
  error: unknown = null;
  expanded = false;
  searchLauncherRunning = false;
  cancelSource: CancelTokenSource | null = null;
  scanStartedAt = 0;
  etaTimer: number | null = null;
  computed = {
    ...mapState(["number"]),
    ...mapMutations(["pushError"]),
  };

  @Prop() scanId!: string;
  @Prop() name!: string;
  @Prop({ default: false }) autoRun!: boolean;
  // Comparison providers surface their fields in the shared matrix, so their
  // own panel stays collapsed on success instead of auto-expanding.
  @Prop({ default: true }) autoExpandOnData!: boolean;
  @Prop({ default: () => ({}) }) scanOptions!: { [key: string]: number };
  @Prop({ default: () => ({}) }) metadata!: LocalMetadata;
  // When set, the scan result is persisted against this lookup record. Empty (the default)
  // keeps the original request body and behavior unchanged.
  @Prop({ default: "" }) lookupId!: string;

  get collapseId(): string {
    return `scanner-collapse-${this.scanId}`;
  }

  get scannerDescription(): string {
    return getScannerDescription(this.scanId);
  }

  get displayName(): string {
    if (this.scanId === "googlesearch") {
      return "Google search";
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

  get isSerpApi(): boolean {
    return this.scanId === "serpapi";
  }

  // Routes the three footprint scanners through the shared SearchActionGroups
  // component. googlesearch shows query links; searxng and serpapi show
  // per-query match counts with inline results.
  get searchGroupsMode(): "google" | "searxng" | "serpapi" {
    if (this.isGoogleSearch) {
      return "google";
    }
    if (this.isSerpApi) {
      return "serpapi";
    }
    return "searxng";
  }

  get isNumverify(): boolean {
    return this.scanId === "numverify";
  }

  get isValidationProvider(): boolean {
    return ["veriphone", "numlookupapi", "abstract"].includes(this.scanId);
  }

  // Numverify and the validation providers share one response shape and one
  // summary rendering.
  get isValidationLike(): boolean {
    return this.isNumverify || this.isValidationProvider;
  }

  get validationResult(): ValidationResult {
    return (this.data || {}) as ValidationResult;
  }

  get validationHeadline(): string {
    return (
      this.validationResult.international_format ||
      this.validationResult.number ||
      ""
    );
  }

  get validationLineType(): string {
    return titleCase(this.validationResult.line_type);
  }

  get validationBadge(): SummaryBadge {
    return this.validationResult.valid
      ? { variant: "success", label: "Valid number", icon: "check-circle-fill" }
      : {
          variant: "danger",
          label: "Invalid number",
          icon: "exclamation-circle-fill",
        };
  }

  get validationSubtext(): string {
    return this.validationLineType ? `${this.validationLineType} line` : "";
  }

  get validationGroups(): NumverifyGroup[] {
    const result = this.validationResult;
    const definitions: {
      key: string;
      label: string;
      rows: NumverifyRow[];
    }[] = [
      {
        key: "formats",
        label: "Number formats",
        rows: [
          { label: "International", value: result.international_format || "" },
          { label: "Local", value: result.local_format || "" },
          { label: "Raw", value: result.number || "" },
        ],
      },
      {
        key: "location",
        label: "Country & location",
        rows: [
          { label: "Country", value: result.country_name || "" },
          { label: "Country code", value: result.country_code || "" },
          { label: "Dial prefix", value: result.country_prefix || "" },
          { label: "Location", value: result.location || "" },
        ],
      },
      {
        key: "carrier",
        label: "Carrier & line",
        rows: [
          { label: "Carrier", value: result.carrier || "" },
          { label: "Line type", value: this.validationLineType },
        ],
      },
    ];

    return definitions
      .map((group) => ({
        ...group,
        rows: group.rows.filter((row) => row.value !== ""),
      }))
      .filter((group) => group.rows.length > 0);
  }

  get isTwilio(): boolean {
    return this.scanId === "twilio";
  }

  get twilio(): TwilioResult {
    return (this.data || {}) as TwilioResult;
  }

  get twilioHeadline(): string {
    return this.twilio.national_format || "";
  }

  get twilioBadge(): SummaryBadge {
    return this.twilio.valid
      ? { variant: "success", label: "Valid number", icon: "check-circle-fill" }
      : {
          variant: "danger",
          label: "Invalid number",
          icon: "exclamation-circle-fill",
        };
  }

  get twilioSubtext(): string {
    const lineType = titleCase(this.twilio.line_type);
    return lineType ? `${lineType} line` : "";
  }

  get twilioGroups(): NumverifyGroup[] {
    const result = this.twilio;
    const simSwap =
      result.sim_swap_in_period === undefined
        ? ""
        : result.sim_swap_in_period
        ? "Yes"
        : "No";
    const definitions: {
      key: string;
      label: string;
      rows: NumverifyRow[];
    }[] = [
      {
        key: "carrier",
        label: "Carrier & line",
        rows: [
          { label: "Carrier", value: result.carrier_name || "" },
          { label: "Line type", value: titleCase(result.line_type) },
          { label: "National format", value: result.national_format || "" },
          { label: "MCC", value: result.mobile_country_code || "" },
          { label: "MNC", value: result.mobile_network_code || "" },
        ],
      },
      {
        key: "caller",
        label: "Caller (CNAM)",
        rows: [
          { label: "Caller name", value: result.caller_name || "" },
          { label: "Caller type", value: titleCase(result.caller_type) },
        ],
      },
      {
        key: "signals",
        label: "Line signals",
        rows: [
          { label: "SIM swapped in period", value: simSwap },
          { label: "Last SIM swap", value: result.sim_swap_last_date || "" },
          {
            label: "Call forwarding",
            value: titleCase(result.call_forwarding_status),
          },
        ],
      },
    ];

    return definitions
      .map((group) => ({
        ...group,
        rows: group.rows.filter((row) => row.value !== ""),
      }))
      .filter((group) => group.rows.length > 0);
  }

  get twilioNotes(): string[] {
    return this.twilio.notes || [];
  }

  get isOvh(): boolean {
    return this.scanId === "ovh";
  }

  get ovh(): OVHResult {
    return (this.data || {}) as OVHResult;
  }

  get ovhGroups(): NumverifyGroup[] {
    const result = this.ovh;
    const rows: NumverifyRow[] = [
      { label: "Number range", value: result.number_range || "" },
      { label: "City", value: result.city || "" },
      { label: "Zip code", value: result.zip_code || "" },
    ].filter((row) => row.value !== "");

    if (rows.length === 0) {
      return [];
    }

    return [{ key: "location", label: "Location", rows }];
  }

  get ovhBadge(): SummaryBadge {
    return this.ovh.found
      ? { variant: "success", label: "Found", icon: "check-circle-fill" }
      : { variant: "secondary", label: "Not found", icon: "dash-circle-fill" };
  }

  get isLocal(): boolean {
    return this.scanId === "local";
  }

  get local(): LocalScannerData {
    return (this.data || {}) as LocalScannerData;
  }

  get hasMetadata(): boolean {
    const m = this.metadata || {};
    return Boolean(m.e164 || m.international || m.local);
  }

  get localRows(): LocalRow[] {
    const data = this.local;
    const meta = this.metadata || {};
    const toText = (value: string | number | undefined): string =>
      value === undefined || value === null ? "" : String(value);

    const definitions = [
      { label: "Raw local", value: data.raw_local, meta: meta.rawLocal },
      { label: "Local", value: data.local, meta: meta.local },
      { label: "E.164", value: data.e164, meta: meta.e164 },
      {
        label: "International",
        value: data.international,
        meta: meta.international,
      },
      {
        label: "Country code",
        value: data.country_code,
        meta: meta.countryCode,
      },
      { label: "Country", value: data.country, meta: meta.country },
      { label: "Carrier", value: data.carrier, meta: meta.carrier },
    ];

    return definitions
      .map((def) => {
        const value = toText(def.value);
        const metaValue = toText(def.meta);
        return {
          label: def.label,
          value,
          changed: this.hasMetadata && value !== metaValue,
        };
      })
      .filter((row) => row.value !== "");
  }

  get localHasChanges(): boolean {
    return this.localRows.some((row) => row.changed);
  }

  get localBadge(): SummaryBadge {
    if (!this.hasMetadata) {
      return {
        variant: "secondary",
        label: "Offline details",
        icon: "info-circle-fill",
      };
    }
    if (this.localHasChanges) {
      return {
        variant: "warning",
        label: "Differs from metadata",
        icon: "exclamation-circle-fill",
      };
    }
    return {
      variant: "success",
      label: "Matches metadata",
      icon: "check-circle-fill",
    };
  }

  get localGroups(): SummaryGroup[] {
    return [{ key: "offline", label: "Offline details", rows: this.localRows }];
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
      const body: {
        number: string;
        options: { [key: string]: number };
        lookupId?: string;
      } = {
        number: this.$store.state.number,
        options: this.scanOptions,
      };
      if (this.lookupId) {
        body.lookupId = this.lookupId;
      }

      const res = await axios.post(
        `${config.apiUrl}/v2/scanners/${this.scanId}/run`,
        body,
        {
          cancelToken: this.cancelSource.token,
          validateStatus: () => true,
        }
      );

      if (!res.data.success && res.data.error) {
        throw res.data.error;
      }
      this.data = res.data.result;
      this.$emit("result", { scanId: this.scanId, data: this.data });
      if (this.autoExpandOnData) {
        this.expanded = true;
      }
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
        this.emitStatus(
          "error",
          `${this.displayName} failed`,
          undefined,
          this.errorText(error)
        );
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

    return Math.max(
      0,
      this.estimatedDurationMs - (Date.now() - this.scanStartedAt)
    );
  }

  emitStatus(
    status: string,
    message: string,
    etaMs?: number,
    error?: string
  ): void {
    this.$emit("status", {
      scanId: this.scanId,
      scanner: this.displayName,
      status,
      message,
      etaMs,
      error,
    });
  }

  private errorText(error: unknown): string {
    if (typeof error === "string") {
      return error;
    }
    if (error instanceof Error) {
      return error.message;
    }
    return String(error);
  }

  private get searchGroupsRef():
    | (Vue & { openAll(): void; openMatches(): void; stop(): void })
    | undefined {
    return this.$refs.searchGroups as
      | (Vue & { openAll(): void; openMatches(): void; stop(): void })
      | undefined;
  }

  openAllSearches(): void {
    this.searchGroupsRef?.openAll();
  }

  openMatchedSearches(): void {
    this.searchGroupsRef?.openMatches();
  }

  stopSearches(): void {
    this.searchGroupsRef?.stop();
  }
}
</script>

<style scoped>
.scanner-panel-header {
  cursor: pointer;
}

.scanner-panel-toggle {
  align-items: center;
  background: transparent;
  border: 0;
  color: var(--ink);
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

.local-banner {
  border: 1px solid var(--rule-soft);
}

.twilio-note {
  border: 1px solid var(--rule-soft);
  font-size: 0.85rem;
}
</style>
