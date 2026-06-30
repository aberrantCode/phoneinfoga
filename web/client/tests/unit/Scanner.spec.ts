// Tests reach into the component instance (dynamic vm members and the private
// errorText helper), which legitimately needs `any`. Scope the relaxation here.
/* eslint-disable @typescript-eslint/no-explicit-any */
import { createLocalVue, shallowMount, Wrapper } from "@vue/test-utils";
import { BootstrapVue, BootstrapVueIcons } from "bootstrap-vue";
import Vuex from "vuex";

// Auto-mock axios so mounting Scanner (which probes /dryrun in mounted()) never
// performs a real request.
jest.mock("axios");
import axios from "axios";

import Scanner from "@/components/Scanner.vue";

const mockedAxios = axios as jest.Mocked<typeof axios>;

function mountScanner(
  props: Record<string, unknown> = {}
): Wrapper<Vue & Record<string, any>> {
  const localVue = createLocalVue();
  localVue.use(BootstrapVue);
  localVue.use(BootstrapVueIcons);
  localVue.use(Vuex);

  const store = new Vuex.Store({
    state: { number: "+12025550100" },
    mutations: { pushError: () => undefined },
  });

  return shallowMount(Scanner, {
    localVue,
    store,
    propsData: { scanId: "numverify", name: "Numverify", ...props },
  }) as Wrapper<Vue & Record<string, any>>;
}

describe("Scanner.vue", () => {
  beforeEach(() => {
    mockedAxios.post.mockReset();
    // dryRun() in mounted() expects a resolved axios response shape.
    mockedAxios.post.mockResolvedValue({ data: { success: true } } as never);
  });

  it("mounts the single-file component", () => {
    const wrapper = mountScanner();
    expect(wrapper.exists()).toBe(true);
  });

  describe("errorText", () => {
    it("returns a string error verbatim", () => {
      const wrapper = mountScanner();
      expect((wrapper.vm as any).errorText("plain failure")).toBe(
        "plain failure"
      );
    });

    it("uses the message of an Error instance", () => {
      const wrapper = mountScanner();
      expect((wrapper.vm as any).errorText(new Error("boom"))).toBe("boom");
    });

    it("stringifies any other value", () => {
      const wrapper = mountScanner();
      expect((wrapper.vm as any).errorText({ code: 500 })).toBe(
        "[object Object]"
      );
      expect((wrapper.vm as any).errorText(42)).toBe("42");
    });
  });

  describe("emitStatus", () => {
    it("emits a status event with the full payload including error text", () => {
      const wrapper = mountScanner();
      wrapper.vm.emitStatus("error", "Numverify failed", undefined, "401");

      const events = wrapper.emitted("status");
      expect(events).toBeTruthy();
      expect(events?.[events.length - 1][0]).toEqual({
        scanId: "numverify",
        scanner: "Numverify",
        status: "error",
        message: "Numverify failed",
        etaMs: undefined,
        error: "401",
      });
    });

    it("omits error/eta for a plain running status", () => {
      const wrapper = mountScanner();
      wrapper.vm.emitStatus("running", "Running Numverify", 5000);

      const events = wrapper.emitted("status");
      const payload = events?.[events.length - 1][0];
      expect(payload).toMatchObject({
        scanId: "numverify",
        status: "running",
        message: "Running Numverify",
        etaMs: 5000,
        error: undefined,
      });
    });
  });

  describe("display getters", () => {
    it("derives a stable collapse id from the scanId", () => {
      const wrapper = mountScanner();
      expect(wrapper.vm.collapseId).toBe("scanner-collapse-numverify");
    });

    it("uses friendly names for the search scanners and the prop otherwise", () => {
      expect(mountScanner({ scanId: "googlesearch" }).vm.displayName).toBe(
        "Google search"
      );
      expect(mountScanner({ scanId: "searxng" }).vm.displayName).toBe(
        "SearXNG search"
      );
      expect(
        mountScanner({ scanId: "numverify", name: "Numverify" }).vm.displayName
      ).toBe("Numverify");
    });
  });
});
