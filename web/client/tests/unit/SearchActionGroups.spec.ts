/* eslint-disable @typescript-eslint/no-explicit-any */
import { createLocalVue, mount, Wrapper } from "@vue/test-utils";
import { BootstrapVue, BootstrapVueIcons } from "bootstrap-vue";

import SearchActionGroups from "@/components/SearchActionGroups.vue";

// SerpAPI returns per-query result counts and inline hits (like SearXNG) but no
// per-query search URL, carrying instead the engine that produced each result.
const serpapiData = {
  general: [
    {
      number: "+12025550100",
      dork: 'intext:"+12025550100"',
      engine: "google",
      result_count: 2,
      results: [
        { title: "A hit", url: "https://example.com/a", engine: "google" },
        { title: "B hit", url: "https://example.com/b", engine: "google" },
      ],
    },
    {
      number: "+12025550100",
      dork: 'site:bing intext:"+12025550100"',
      engine: "bing",
      result_count: 0,
    },
  ],
  social_media: [],
};

// SearXNG carries a per-query `url` used to reopen the search in a browser.
const searxngData = {
  general: [
    {
      number: "+12025550100",
      dork: 'intext:"+12025550100"',
      url: "https://searx.example.com/search?q=foo",
      result_count: 1,
      results: [{ title: "A hit", url: "https://example.com/a" }],
    },
  ],
};

function mountGroups(
  props: Record<string, unknown>
): Wrapper<Vue & Record<string, any>> {
  const localVue = createLocalVue();
  localVue.use(BootstrapVue);
  localVue.use(BootstrapVueIcons);

  return mount(SearchActionGroups, {
    localVue,
    propsData: props,
  }) as Wrapper<Vue & Record<string, any>>;
}

describe("SearchActionGroups.vue serpapi mode", () => {
  it("groups results by investigation intent, dropping empty categories", () => {
    const wrapper = mountGroups({ mode: "serpapi", data: serpapiData });
    const groups = wrapper.vm.resultsGroups;
    expect(groups).toHaveLength(1);
    expect(groups[0].key).toBe("general");
    expect(groups[0].items).toHaveLength(2);
  });

  it("counts only non-error queries that returned results as matches", () => {
    const wrapper = mountGroups({ mode: "serpapi", data: serpapiData });
    expect(wrapper.vm.groupMatchCount(serpapiData.general)).toBe(1);
  });

  it("disables the batch launcher and uses SerpApi-specific copy", () => {
    const wrapper = mountGroups({ mode: "serpapi", data: serpapiData });
    expect(wrapper.vm.supportsLauncher).toBe(false);
    expect(wrapper.vm.resultsIntro).toContain("SerpApi");
  });

  it("renders the engine badge and omits url-only controls", () => {
    const wrapper = mountGroups({ mode: "serpapi", data: serpapiData });
    const html = wrapper.html();
    expect(html).toContain("google");
    expect(html).toContain("bing");
    // No per-query URL to open, so the open/copy-url controls are absent.
    expect(wrapper.find('[aria-label="Open query in SearXNG"]').exists()).toBe(
      false
    );
    expect(
      wrapper.find('[aria-label="Copy SearXNG search URL"]').exists()
    ).toBe(false);
    // The copy-query control is still available.
    expect(wrapper.find('[aria-label="Copy query text"]').exists()).toBe(true);
  });

  it("keeps the launcher and open controls for searxng mode", () => {
    const wrapper = mountGroups({ mode: "searxng", data: searxngData });
    expect(wrapper.vm.supportsLauncher).toBe(true);
    expect(wrapper.vm.resultsIntro).toContain("SearXNG");
    expect(wrapper.find('[aria-label="Open query in SearXNG"]').exists()).toBe(
      true
    );
  });
});
