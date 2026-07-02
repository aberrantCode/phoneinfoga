// Tests reach into the component instance (dynamic vm members), which
// legitimately needs `any`. Scope the relaxation to this test file.
/* eslint-disable @typescript-eslint/no-explicit-any */
import { createLocalVue, shallowMount, Wrapper } from "@vue/test-utils";
import { BootstrapVue, BootstrapVueIcons } from "bootstrap-vue";
import Vuex, { Store } from "vuex";

// Keep the pure helpers (display names, descriptions, default selection) real,
// but stub the functions that hit the network so mounting/orchestration does not
// fire real requests.
jest.mock("@/utils", () => {
  const actual = jest.requireActual("@/utils");
  return {
    __esModule: true,
    ...actual,
    getScannerAvailability: jest.fn().mockResolvedValue([]),
    createLookup: jest.fn(),
    closeLookup: jest.fn(),
    getLatestLookup: jest.fn().mockResolvedValue(null),
    listLookups: jest.fn().mockResolvedValue([]),
    getLookup: jest.fn(),
  };
});
jest.mock("axios");

import axios from "axios";
import Scan from "@/views/Scan.vue";
import VuePhoneNumberInput from "vue-phone-number-input";
import {
  getScannerAvailability,
  getScannerDescription,
  createLookup,
  closeLookup,
  getLatestLookup,
  listLookups,
  getLookup,
  ScannerAvailability,
} from "@/utils";

const mockedAxios = axios as jest.Mocked<typeof axios>;
const mockedCreateLookup = createLookup as jest.Mock;
const mockedCloseLookup = closeLookup as jest.Mock;
const mockedGetLatestLookup = getLatestLookup as jest.Mock;
const mockedListLookups = listLookups as jest.Mock;
const mockedGetLookup = getLookup as jest.Mock;

const mockedGetAvailability = getScannerAvailability as jest.Mock;

const scanner = (
  name: string,
  available: boolean,
  error?: string
): ScannerAvailability => ({
  name,
  description: "",
  available,
  ...(error !== undefined ? { error } : {}),
});

function mountScan(): {
  wrapper: Wrapper<Vue & Record<string, any>>;
  store: Store<{ number: string }>;
  commit: jest.Mock;
} {
  const localVue = createLocalVue();
  localVue.use(BootstrapVue);
  localVue.use(BootstrapVueIcons);
  localVue.use(Vuex);

  const commit = jest.fn();
  const store = new Vuex.Store<{ number: string }>({
    state: { number: "" },
    mutations: {
      setNumber: () => undefined,
      pushError: () => undefined,
      resetState: () => undefined,
    },
  });
  store.commit = (commit as unknown) as typeof store.commit;

  const wrapper = shallowMount(Scan, { localVue, store }) as Wrapper<
    Vue & Record<string, any>
  >;
  return { wrapper, store, commit };
}

describe("Scan.vue", () => {
  beforeEach(() => {
    window.localStorage.clear();
    mockedGetAvailability.mockReset();
    mockedGetAvailability.mockResolvedValue([]);
  });

  describe("selectableScanners", () => {
    it("lists available scanners first and unavailable ones last", async () => {
      const { wrapper } = mountScan();
      await wrapper.setData({
        scannerAvailability: [
          scanner("searxng", false, "no instance"),
          scanner("numverify", true),
        ],
        scannerFailures: {},
      });

      const names = wrapper.vm.selectableScanners.map(
        (s: ScannerAvailability) => s.name
      );
      expect(names).toEqual(["numverify", "searxng"]);
    });

    it("demotes an available scanner that has a remembered run failure", async () => {
      const { wrapper } = mountScan();
      await wrapper.setData({
        scannerAvailability: [scanner("numverify", true)],
        scannerFailures: { numverify: "401 Unauthorized" },
      });

      const [decorated] = wrapper.vm.selectableScanners;
      expect(decorated.name).toBe("numverify");
      expect(decorated.available).toBe(false);
      expect(decorated.error).toBe("401 Unauthorized");
    });
  });

  describe("scannerToggleTitle", () => {
    it("returns the scanner description when available", () => {
      const { wrapper } = mountScan();
      expect(wrapper.vm.scannerToggleTitle(scanner("numverify", true))).toBe(
        getScannerDescription("numverify")
      );
    });

    it("surfaces the specific error when unavailable with a reason", () => {
      const { wrapper } = mountScan();
      expect(
        wrapper.vm.scannerToggleTitle(
          scanner("numverify", false, "missing key")
        )
      ).toBe("Unavailable — missing key");
    });

    it("falls back to a generic message when unavailable without a reason", () => {
      const { wrapper } = mountScan();
      expect(
        wrapper.vm.scannerToggleTitle(scanner("numverify", false))
      ).toContain("Unavailable");
    });
  });

  describe("updateScannerStatus", () => {
    it("appends a new status and replaces an existing one by scanId", async () => {
      const { wrapper } = mountScan();
      wrapper.vm.updateScannerStatus({
        scanId: "numverify",
        scanner: "Numverify",
        status: "queued",
        message: "Waiting",
      });
      expect(wrapper.vm.scannerStatuses).toHaveLength(1);

      wrapper.vm.updateScannerStatus({
        scanId: "numverify",
        scanner: "Numverify",
        status: "running",
        message: "Running",
      });
      expect(wrapper.vm.scannerStatuses).toHaveLength(1);
      expect(wrapper.vm.scannerStatuses[0].status).toBe("running");
    });

    it("remembers a failure on an error status (preferring error over message)", () => {
      const { wrapper } = mountScan();
      wrapper.vm.updateScannerStatus({
        scanId: "numverify",
        scanner: "Numverify",
        status: "error",
        message: "Numverify failed",
        error: "401 Unauthorized",
      });
      expect(wrapper.vm.scannerFailures.numverify).toBe("401 Unauthorized");
    });

    it("falls back to the message when an error status carries no error text", () => {
      const { wrapper } = mountScan();
      wrapper.vm.updateScannerStatus({
        scanId: "numverify",
        scanner: "Numverify",
        status: "error",
        message: "Numverify failed",
      });
      expect(wrapper.vm.scannerFailures.numverify).toBe("Numverify failed");
    });

    it("clears a remembered failure when the scanner later completes", async () => {
      const { wrapper } = mountScan();
      await wrapper.setData({ scannerFailures: { numverify: "boom" } });
      wrapper.vm.updateScannerStatus({
        scanId: "numverify",
        scanner: "Numverify",
        status: "complete",
        message: "Numverify finished",
      });
      expect(wrapper.vm.scannerFailures.numverify).toBeUndefined();
    });
  });

  describe("loadScannerAvailability", () => {
    it("populates availability, default selection and resets failures", async () => {
      // Persistent (not Once): the component's mounted() hook calls this loader
      // before the test's explicit call, so both calls must return the list.
      mockedGetAvailability.mockResolvedValue([
        scanner("numverify", true),
        scanner("searxng", false, "no instance"),
      ]);
      const { wrapper } = mountScan();
      await wrapper.setData({ scannerFailures: { stale: "old error" } });

      await wrapper.vm.loadScannerAvailability();

      expect(wrapper.vm.scannerAvailability).toHaveLength(2);
      expect(wrapper.vm.selectedScannerNames).toEqual(["numverify"]);
      expect(wrapper.vm.scannerFailures).toEqual({});
      expect(wrapper.vm.scannerError).toBe("");
    });

    it("records a scannerError when the probe rejects", async () => {
      mockedGetAvailability.mockRejectedValue(new Error("network down"));
      const { wrapper } = mountScan();

      await wrapper.vm.loadScannerAvailability();

      expect(wrapper.vm.scannerError).toContain("network down");
    });
  });

  describe("getScanners", () => {
    it("runs only available, non-failed, selected scanners", async () => {
      const { wrapper } = mountScan();
      await wrapper.setData({
        scannerAvailability: [
          scanner("numverify", true),
          scanner("searxng", true),
          scanner("abstract", false),
        ],
        selectedScannerNames: ["numverify", "searxng", "abstract"],
        scannerFailures: { searxng: "rate limited" },
      });

      await wrapper.vm.getScanners();

      expect(
        wrapper.vm.scanners.map((s: ScannerAvailability) => s.name)
      ).toEqual(["numverify"]);
      expect(wrapper.vm.scannerStatuses).toHaveLength(1);
      expect(wrapper.vm.scannerStatuses[0]).toMatchObject({
        scanId: "numverify",
        status: "queued",
      });
    });
  });

  describe("presentation helpers", () => {
    it("maps statuses to bootstrap variants", () => {
      const { wrapper } = mountScan();
      expect(wrapper.vm.statusVariant("running")).toBe("primary");
      expect(wrapper.vm.statusVariant("complete")).toBe("success");
      expect(wrapper.vm.statusVariant("error")).toBe("danger");
      expect(wrapper.vm.statusVariant("canceled")).toBe("warning");
      expect(wrapper.vm.statusVariant("queued")).toBe("secondary");
    });

    it("capitalises the status label", () => {
      const { wrapper } = mountScan();
      expect(wrapper.vm.statusLabel("running")).toBe("Running");
    });

    it("formats durations under and over a minute", () => {
      const { wrapper } = mountScan();
      expect(wrapper.vm.formatDuration(15000)).toBe("15s");
      expect(wrapper.vm.formatDuration(90000)).toBe("1m 30s");
      expect(wrapper.vm.formatDuration(-1000)).toBe("0s");
    });

    it("counts active and finished scanners", async () => {
      const { wrapper } = mountScan();
      await wrapper.setData({
        scannerStatuses: [
          { scanId: "a", scanner: "A", status: "running", message: "" },
          { scanId: "b", scanner: "B", status: "complete", message: "" },
          { scanId: "c", scanner: "C", status: "error", message: "" },
        ],
      });
      expect(wrapper.vm.activeScannerCount).toBe(1);
      expect(wrapper.vm.finishedScannerCount).toBe(2);
    });

    it("filters empty/zero and duplicate metadata items", async () => {
      const { wrapper } = mountScan();
      await wrapper.setData({
        localData: {
          valid: true,
          raw_local: "",
          local: "",
          e164: "+12025550100",
          international: "",
          countryCode: 0,
          country: "United States",
          carrier: "",
        },
      });
      const labels = wrapper.vm.metadataItems.map(
        (i: { label: string }) => i.label
      );
      expect(labels).toContain("Valid");
      expect(labels).toContain("E.164");
      expect(labels).toContain("Country");
      expect(labels).not.toContain("Calling code"); // value was 0
      expect(labels).not.toContain("Local"); // value was ""
    });
  });

  describe("viewState model", () => {
    it("starts in the entry state with no active lookup", () => {
      const { wrapper } = mountScan();
      expect(wrapper.vm.viewState).toMatchObject({
        state: "entry",
        source: "fresh",
        activeLookup: null,
      });
      expect(wrapper.vm.isEntryState).toBe(true);
      expect(wrapper.vm.isResultsState).toBe(false);
      expect(wrapper.vm.isReplay).toBe(false);
    });

    it("enterResults('fresh') moves to a fresh results state", () => {
      const { wrapper } = mountScan();
      wrapper.vm.enterResults("fresh");
      expect(wrapper.vm.isResultsState).toBe(true);
      expect(wrapper.vm.isReplay).toBe(false);
    });

    it("enterResults('replay', lookup) records the active lookup and is a replay", () => {
      const { wrapper } = mountScan();
      const lookup = { id: "lk-1", results: [] };
      wrapper.vm.enterResults("replay", lookup);
      expect(wrapper.vm.isResultsState).toBe(true);
      expect(wrapper.vm.isReplay).toBe(true);
      expect(wrapper.vm.viewState.activeLookup).toBe(lookup);
    });

    it("enterEntry() (and clearData) return to the entry state", () => {
      const { wrapper } = mountScan();
      wrapper.vm.enterResults("replay", { id: "lk-1", results: [] });
      wrapper.vm.enterEntry();
      expect(wrapper.vm.isEntryState).toBe(true);
      expect(wrapper.vm.viewState.activeLookup).toBeNull();

      wrapper.vm.enterResults("fresh");
      wrapper.vm.clearData();
      expect(wrapper.vm.isEntryState).toBe(true);
    });
  });

  describe("fresh-lookup orchestration (AC1-AC3)", () => {
    const flush = (): Promise<void> =>
      new Promise((resolve) => setTimeout(resolve, 0));
    const validNumber = {
      valid: true,
      e164: "+14152229670",
      raw_local: "",
      local: "",
      international: "",
      countryCode: 1,
      country: "US",
      carrier: "",
    };

    beforeEach(() => {
      mockedCreateLookup.mockReset();
      mockedCloseLookup.mockReset();
      mockedAxios.post.mockReset();
      // Default: no prior lookup, so runScans takes the fresh path.
      mockedGetLatestLookup.mockReset();
      mockedGetLatestLookup.mockResolvedValue(null);
    });

    // Drain the mounted() availability loader first so it can't overwrite the
    // scanner selection we set up here mid-runScans.
    const arrangeFreshLookup = async (
      wrapper: Wrapper<Vue & Record<string, any>>
    ): Promise<void> => {
      await flush();
      wrapper.vm.inputNumber = "14152229670";
      wrapper.vm.scannerAvailability = [scanner("local", true)];
      wrapper.vm.selectedScannerNames = ["local"];
    };

    it("creates a lookup and enters the fresh results state", async () => {
      mockedCreateLookup.mockResolvedValue({
        id: "lk-9",
        createdAt: "2026-07-01T10:00:00Z",
        clientIp: "1.2.3.4",
        scannersRequested: ["local"],
        status: "pending",
      });
      mockedAxios.post.mockResolvedValue({ data: validNumber } as never);

      const { wrapper } = mountScan();
      await arrangeFreshLookup(wrapper);
      await wrapper.vm.runScans();
      await flush();

      expect(mockedCreateLookup).toHaveBeenCalledTimes(1);
      expect(mockedCreateLookup.mock.calls[0][1]).toEqual(["local"]);
      expect(wrapper.vm.activeLookupId).toBe("lk-9");
      expect(wrapper.vm.isResultsState).toBe(true);
      expect(wrapper.vm.viewState.source).toBe("fresh");
    });

    it("closes the lookup once every scanner settles", async () => {
      mockedCreateLookup.mockResolvedValue({
        id: "lk-9",
        createdAt: "",
        clientIp: "",
        scannersRequested: ["local"],
        status: "pending",
      });
      mockedCloseLookup.mockResolvedValue({
        id: "lk-9",
        status: "complete",
        completedAt: "2026-07-01T10:01:00Z",
        scannersRequested: ["local"],
        createdAt: "",
      });
      mockedAxios.post.mockResolvedValue({ data: validNumber } as never);

      const { wrapper } = mountScan();
      await arrangeFreshLookup(wrapper);
      await wrapper.vm.runScans();
      await flush();

      wrapper.vm.updateScannerStatus({
        scanId: "local",
        scanner: "Local",
        status: "complete",
        message: "done",
      });
      await flush();

      expect(mockedCloseLookup).toHaveBeenCalledWith("lk-9");
    });

    it("still shows results when createLookup fails (persistence is non-fatal)", async () => {
      mockedCreateLookup.mockRejectedValue(new Error("db down"));
      mockedAxios.post.mockResolvedValue({ data: validNumber } as never);

      const { wrapper } = mountScan();
      await arrangeFreshLookup(wrapper);
      await wrapper.vm.runScans();
      await flush();

      expect(wrapper.vm.isResultsState).toBe(true);
      expect(wrapper.vm.activeLookupId).toBe("");
    });

    it("replays the latest lookup without scanning on 200 (AC7)", async () => {
      mockedGetLatestLookup.mockResolvedValue({
        id: "lk-old",
        status: "complete",
        createdAt: "2026-07-01T09:00:00Z",
        completedAt: "2026-07-01T09:01:00Z",
        clientIp: "203.0.113.7",
        userAgent: "UA",
        scannersRequested: ["local"],
        number: {
          valid: true,
          e164: "+14152229670",
          rawLocal: "4152229670",
          local: "",
          international: "",
          countryCode: 1,
          country: "US",
          carrier: "",
        },
        results: [
          {
            scanner: "local",
            status: "success",
            raw: { e164: "+14152229670" },
            durationMs: 1,
            startedAt: "2026-07-01T09:00:00Z",
            finishedAt: "2026-07-01T09:00:00Z",
          },
        ],
      });

      const { wrapper } = mountScan();
      await arrangeFreshLookup(wrapper);
      await wrapper.vm.runScans();
      await flush();

      expect(mockedGetLatestLookup).toHaveBeenCalled();
      // Replay: no fresh persistence and no /run or /v2/numbers requests (AC7).
      expect(mockedCreateLookup).not.toHaveBeenCalled();
      expect(mockedAxios.post).not.toHaveBeenCalled();
      expect(wrapper.vm.isReplay).toBe(true);
      expect(wrapper.vm.viewState.activeLookup.id).toBe("lk-old");
      expect(wrapper.vm.localData.e164).toBe("+14152229670");
    });

    it("runs a fresh lookup when there is no prior lookup (404)", async () => {
      mockedGetLatestLookup.mockResolvedValue(null);
      mockedCreateLookup.mockResolvedValue({
        id: "lk-new",
        createdAt: "",
        clientIp: "",
        scannersRequested: ["local"],
        status: "pending",
      });
      mockedAxios.post.mockResolvedValue({ data: validNumber } as never);

      const { wrapper } = mountScan();
      await arrangeFreshLookup(wrapper);
      await wrapper.vm.runScans();
      await flush();

      expect(mockedCreateLookup).toHaveBeenCalled();
      expect(wrapper.vm.viewState.source).toBe("fresh");
    });

    it("Run new lookup forces a fresh scan from replay (AC8)", async () => {
      mockedGetLatestLookup.mockResolvedValue({
        id: "lk-old",
        status: "complete",
        createdAt: "2026-07-01T09:00:00Z",
        completedAt: "2026-07-01T09:01:00Z",
        clientIp: "203.0.113.7",
        userAgent: "UA",
        scannersRequested: ["local"],
        number: {
          valid: true,
          e164: "+14152229670",
          rawLocal: "",
          local: "",
          international: "",
          countryCode: 1,
          country: "US",
          carrier: "",
        },
        results: [
          {
            scanner: "local",
            status: "success",
            raw: {},
            durationMs: 1,
            startedAt: "",
            finishedAt: "",
          },
        ],
      });
      mockedCreateLookup.mockResolvedValue({
        id: "lk-new",
        createdAt: "",
        clientIp: "",
        scannersRequested: ["local"],
        status: "pending",
      });
      mockedAxios.post.mockResolvedValue({ data: validNumber } as never);

      const { wrapper } = mountScan();
      await arrangeFreshLookup(wrapper);
      await wrapper.vm.runScans();
      await flush();
      expect(wrapper.vm.isReplay).toBe(true);

      await wrapper.vm.runNewLookup();
      await flush();

      // A brand-new lookup was created and we are now in the fresh results state.
      expect(mockedCreateLookup).toHaveBeenCalled();
      expect(wrapper.vm.viewState.source).toBe("fresh");
      expect(wrapper.vm.activeLookupId).toBe("lk-new");
    });
  });

  describe("previous lookups dropdown (AC9)", () => {
    const flush = (): Promise<void> =>
      new Promise((resolve) => setTimeout(resolve, 0));

    beforeEach(() => {
      mockedListLookups.mockReset();
      mockedGetLookup.mockReset();
    });

    it("loads the number's lookups on open", async () => {
      mockedListLookups.mockResolvedValue([
        {
          id: "a",
          e164: "+14152229670",
          status: "complete",
          scannersRequested: ["local"],
          createdAt: "2026-07-01T09:00:00Z",
          completedAt: null,
        },
      ]);

      const { wrapper } = mountScan();
      await flush();
      await wrapper.vm.loadPreviousLookups();

      expect(mockedListLookups).toHaveBeenCalled();
      expect(wrapper.vm.previousLookups).toHaveLength(1);
    });

    it("opens a selected lookup and renders it in replay mode without scanning", async () => {
      mockedGetLookup.mockResolvedValue({
        id: "a",
        status: "complete",
        createdAt: "2026-07-01T09:00:00Z",
        completedAt: "2026-07-01T09:01:00Z",
        clientIp: "203.0.113.7",
        userAgent: "UA",
        scannersRequested: ["local"],
        number: {
          valid: true,
          e164: "+14152229670",
          rawLocal: "",
          local: "",
          international: "",
          countryCode: 1,
          country: "US",
          carrier: "",
        },
        results: [
          {
            scanner: "local",
            status: "success",
            raw: {},
            durationMs: 1,
            startedAt: "",
            finishedAt: "",
          },
        ],
      });
      mockedAxios.post.mockReset();

      const { wrapper } = mountScan();
      await flush();
      await wrapper.vm.openPreviousLookup("a");
      await flush();

      expect(mockedGetLookup).toHaveBeenCalledWith("a");
      expect(wrapper.vm.isReplay).toBe(true);
      expect(wrapper.vm.viewState.activeLookup.id).toBe("a");
      // No scanning triggered by opening a historical lookup.
      expect(mockedAxios.post).not.toHaveBeenCalled();
    });
  });

  describe("replay banner", () => {
    it("shows the replay banner with the lookup time in replay mode", async () => {
      const { wrapper } = mountScan();
      wrapper.vm.enterResults("replay", {
        id: "lk-1",
        status: "complete",
        createdAt: "2026-07-01T10:00:00Z",
        completedAt: null,
        clientIp: "",
        userAgent: "",
        scannersRequested: [],
        number: {
          valid: true,
          e164: "+14152229670",
          rawLocal: "",
          local: "",
          international: "",
          countryCode: 1,
          country: "US",
          carrier: "",
        },
        results: [],
      });
      await wrapper.vm.$nextTick();

      expect(wrapper.vm.replayBannerText).toContain(
        "Showing your most recent lookup from"
      );
      expect(wrapper.text()).toContain("Showing your most recent lookup from");
    });

    it("has no banner in the fresh results state", () => {
      const { wrapper } = mountScan();
      wrapper.vm.enterResults("fresh");
      expect(wrapper.vm.replayBannerText).toBe("");
    });
  });

  describe("request record in metadata panel (AC5)", () => {
    it("derives the lookup record fields from the active lookup", () => {
      const { wrapper } = mountScan();
      wrapper.vm.enterResults("replay", {
        id: "lk-1",
        status: "complete",
        createdAt: "2026-07-01T10:00:00Z",
        clientIp: "203.0.113.7",
        scannersRequested: ["local", "numverify"],
        results: [],
      });

      const record = wrapper.vm.lookupRecordItems as Array<{
        label: string;
        value: string;
      }>;
      const byLabel = Object.fromEntries(record.map((i) => [i.label, i.value]));

      expect(Object.keys(byLabel)).toEqual(
        expect.arrayContaining([
          "Lookup time",
          "Client IP",
          "Scanners requested",
          "Status",
        ])
      );
      expect(byLabel["Client IP"]).toBe("203.0.113.7");
      expect(byLabel["Scanners requested"]).toBe("local, numverify");
      expect(byLabel["Status"]).toBe("Complete");
      expect(byLabel["Lookup time"]).toBeTruthy();
    });

    it("has no record items in the entry state", () => {
      const { wrapper } = mountScan();
      expect(wrapper.vm.lookupRecordItems).toEqual([]);
    });
  });

  describe("results-state input hiding (AC6)", () => {
    it("shows the phone input in entry state and hides it in results state", async () => {
      const { wrapper } = mountScan();
      expect(wrapper.findComponent(VuePhoneNumberInput).exists()).toBe(true);

      wrapper.vm.enterResults("fresh");
      await wrapper.vm.$nextTick();

      expect(wrapper.findComponent(VuePhoneNumberInput).exists()).toBe(false);
      expect(wrapper.text()).toContain("Start over");
    });

    it("startOver returns to entry and clears the input field", async () => {
      const { wrapper } = mountScan();
      wrapper.vm.enterResults("fresh");
      wrapper.vm.inputNumber = "14152229670";
      wrapper.vm.inputNumberVal = "14152229670";
      wrapper.vm.inputNumberValid = true;

      wrapper.vm.startOver();
      await wrapper.vm.$nextTick();

      expect(wrapper.vm.isEntryState).toBe(true);
      expect(wrapper.vm.inputNumber).toBe("");
      expect(wrapper.vm.inputNumberVal).toBe("");
      expect(wrapper.vm.inputNumberValid).toBe(false);
      expect(wrapper.findComponent(VuePhoneNumberInput).exists()).toBe(true);
    });
  });

  describe("results-state rendering (AC5 + AC6)", () => {
    it("renders the request record fields, hides the input and shows Start over", async () => {
      const { wrapper } = mountScan();
      wrapper.vm.enterResults("replay", {
        id: "lk-1",
        status: "complete",
        createdAt: "2026-07-01T10:00:00Z",
        completedAt: "2026-07-01T10:01:00Z",
        clientIp: "203.0.113.7",
        userAgent: "UA",
        scannersRequested: ["local", "numverify"],
        number: {
          valid: true,
          e164: "+14152229670",
          rawLocal: "",
          local: "",
          international: "",
          countryCode: 1,
          country: "US",
          carrier: "",
        },
        results: [],
      });
      await wrapper.vm.$nextTick();

      const text = wrapper.text();
      // Request record fields (AC5).
      expect(text).toContain("Request record");
      expect(text).toContain("203.0.113.7");
      expect(text).toContain("Complete");
      expect(text).toContain("local, numverify");

      // Input hidden, Start over shown (AC6).
      expect(wrapper.findComponent(VuePhoneNumberInput).exists()).toBe(false);
      expect(text).toContain("Start over");
    });
  });
});
