import { createLocalVue, shallowMount } from "@vue/test-utils";
import { BootstrapVue } from "bootstrap-vue";
import ScannerSummary from "@/components/ScannerSummary.vue";

// Smoke test proving the SFC toolchain (vue-jest + vue-template-compiler)
// can compile and mount a single-file component. This is the test that was
// blocked while vue-template-compiler (2.6.12) was out of sync with vue (2.7.x).
//
// BootstrapVue is registered on a localVue so the b-* elements are known
// components; shallowMount then stubs them, keeping the test isolated and
// free of "Unknown custom element" warnings.
const localVue = createLocalVue();
localVue.use(BootstrapVue);

describe("ScannerSummary.vue", () => {
  it("compiles and mounts the single-file component", () => {
    const wrapper = shallowMount(ScannerSummary, {
      localVue,
      propsData: {
        badge: { variant: "success", label: "Completed" },
        headline: "Scan finished",
      },
    });

    expect(wrapper.exists()).toBe(true);
    expect(wrapper.find(".scanner-summary").exists()).toBe(true);
  });

  it("renders the headline and subtext props", () => {
    const wrapper = shallowMount(ScannerSummary, {
      localVue,
      propsData: {
        badge: { variant: "info", label: "Running" },
        headline: "In progress",
        subtext: "Querying providers",
      },
    });

    expect(wrapper.find(".scanner-summary-headline").text()).toBe(
      "In progress"
    );
    expect(wrapper.find(".scanner-summary-subtext").text()).toBe(
      "Querying providers"
    );
  });

  it("shows the empty text when no groups are provided", () => {
    const wrapper = shallowMount(ScannerSummary, {
      localVue,
      propsData: {
        badge: { variant: "secondary", label: "Idle" },
        emptyText: "No results yet",
      },
    });

    expect(wrapper.text()).toContain("No results yet");
  });
});
