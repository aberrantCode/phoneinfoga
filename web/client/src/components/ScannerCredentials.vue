<template>
  <span>
    <button
      id="scanner-credentials-button"
      class="ac-tool ml-2"
      type="button"
      aria-label="Scanner credentials"
      @click="$bvModal.show('scanner-credentials-modal')"
    >
      <b-icon-key-fill aria-hidden="true" />
    </button>
    <b-tooltip target="scanner-credentials-button">
      Scanner credentials
    </b-tooltip>

    <b-modal
      id="scanner-credentials-modal"
      title="Scanner Credentials"
      ok-title="Save"
      :ok-disabled="loading || saving"
      @show="loadConfig"
      @ok.prevent="saveConfig"
    >
      <p class="text-muted">
        Enter API credentials to enable the Twilio and breach scanners. Values
        are applied to the running server immediately — no restart needed.
        Secrets are stored only in the server process and are never returned in
        full.
      </p>

      <b-alert v-if="error" show variant="danger">{{ error }}</b-alert>
      <b-alert v-if="saved" show variant="success" dismissible>
        Credentials saved and applied.
      </b-alert>

      <div v-if="loading" class="text-center my-3">
        <b-spinner small type="grow" /> Loading current configuration…
      </div>

      <div v-else>
        <h6>Twilio Lookup v2</h6>
        <b-form-group
          v-for="field in twilioSecrets"
          :key="field.key"
          :label="field.label"
          :label-for="field.key"
          :description="hintFor(field.key)"
        >
          <b-form-input
            :id="field.key"
            v-model="values[field.key]"
            type="password"
            autocomplete="off"
            :placeholder="placeholderFor(field.key)"
          />
        </b-form-group>

        <hr />

        <h6>Breach / leak (Dehashed)</h6>
        <b-form-checkbox v-model="breachEnabled" class="mb-2">
          Enable breach scanner
          <small class="text-muted"
            >(opt-in; required to run breach lookups)</small
          >
        </b-form-checkbox>
        <b-form-group
          v-for="field in dehashedSecrets"
          :key="field.key"
          :label="field.label"
          :label-for="field.key"
          :description="hintFor(field.key)"
        >
          <b-form-input
            :id="field.key"
            v-model="values[field.key]"
            type="password"
            autocomplete="off"
            :placeholder="placeholderFor(field.key)"
          />
        </b-form-group>
      </div>
    </b-modal>
  </span>
</template>

<script lang="ts">
import Vue from "vue";
import axios, { AxiosResponse } from "axios";
import config from "@/config";

type ConfigField = {
  key: string;
  secret: boolean;
  configured: boolean;
  value?: string;
};

type ConfigResponse = {
  success: boolean;
  fields: ConfigField[];
};

const TWILIO_SECRETS = [
  { key: "TWILIO_ACCOUNT_SID", label: "Account SID" },
  { key: "TWILIO_AUTH_TOKEN", label: "Auth token" },
];

const DEHASHED_SECRETS = [
  { key: "DEHASHED_EMAIL", label: "Dehashed email" },
  { key: "DEHASHED_API_KEY", label: "Dehashed API key" },
];

export default Vue.extend({
  name: "ScannerCredentials",
  data: () => ({
    twilioSecrets: TWILIO_SECRETS,
    dehashedSecrets: DEHASHED_SECRETS,
    fields: [] as ConfigField[],
    values: {} as Record<string, string>,
    breachEnabled: false,
    loading: false,
    saving: false,
    saved: false,
    error: "",
  }),
  methods: {
    fieldByKey(key: string): ConfigField | undefined {
      return this.fields.find((f) => f.key === key);
    },
    hintFor(key: string): string {
      const field = this.fieldByKey(key);
      if (field && field.configured) {
        return "Currently set. Leave blank to keep the existing value.";
      }
      return "Not set.";
    },
    placeholderFor(key: string): string {
      const field = this.fieldByKey(key);
      if (field && field.configured && field.value) {
        return `Set (${field.value})`;
      }
      return "Not set";
    },
    async loadConfig(): Promise<void> {
      this.loading = true;
      this.error = "";
      this.saved = false;
      this.values = {};
      try {
        const res: AxiosResponse<ConfigResponse> = await axios.get(
          `${config.apiUrl}/config`
        );
        this.fields = res.data.fields;
        const enabled = this.fieldByKey("BREACH_SCANNER_ENABLED");
        this.breachEnabled = !!enabled && enabled.value === "true";
      } catch (err) {
        this.error = `Failed to load configuration: ${err}`;
      }
      this.loading = false;
    },
    async saveConfig(): Promise<void> {
      this.saving = true;
      this.error = "";
      this.saved = false;

      const payload: Record<string, string> = {
        BREACH_SCANNER_ENABLED: this.breachEnabled ? "true" : "false",
      };
      // Only send secrets the user actually typed, so blank inputs preserve
      // the existing server-side value.
      for (const key of Object.keys(this.values)) {
        const value = this.values[key];
        if (value && value.length > 0) {
          payload[key] = value;
        }
      }

      try {
        const res: AxiosResponse<ConfigResponse> = await axios.post(
          `${config.apiUrl}/config`,
          payload
        );
        this.fields = res.data.fields;
        this.values = {};
        this.saved = true;
        // Let the rest of the app re-check scanner availability.
        window.dispatchEvent(new CustomEvent("scanner-preferences-updated"));
      } catch (err) {
        this.error = `Failed to save configuration: ${err}`;
      }
      this.saving = false;
    },
  },
});
</script>
