// Tests reach into the component instance (dynamic vm members), which
// legitimately needs `any`. Scope the relaxation to this test file.
/* eslint-disable @typescript-eslint/no-explicit-any */
import { createLocalVue, shallowMount, Wrapper } from "@vue/test-utils";
import { BootstrapVue, BootstrapVueIcons } from "bootstrap-vue";
import Vuex, { Store } from "vuex";

// Keep the pure helpers (display names, descriptions, default selection) real,
// but stub the one function that hits the network so mounting the view does not
// fire a real dryrun probe.
jest.mock("@/utils", () => {
  const actual = jest.requireActual("@/utils");
  return {
    __esModule: true,
    ...actual,
    getScannerAvailability: jest.fn().mockResolvedValue([]),
  };
});

import Scan from "@/views/Scan.vue";
import {
  getScannerAvailability,
  getScannerDescription,
  ScannerAvailability,
} from "@/utils";

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
});
