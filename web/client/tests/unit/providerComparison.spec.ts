import { buildComparison, __test__ } from "../../src/utils/providerComparison";
import { canonicalCountry } from "../../src/utils/countries";

// Realistic-ish fixtures modelled on the actual scanner JSON shapes.
const numverify = {
  valid: true,
  number: "12028675309",
  local_format: "(202) 867-5309",
  international_format: "+1 202 867 5309",
  country_prefix: "+1",
  country_code: "US",
  country_name: "United States",
  location: "New York",
  carrier: "T-Mobile USA, Inc.",
  line_type: "mobile",
};

const veriphone = {
  provider: "veriphone",
  valid: true,
  local_format: "(202) 867-5309",
  international_format: "+1 202 867 5309",
  country_name: "United States",
  country_prefix: "+1",
  location: "New York",
  carrier: "T-Mobile USA, Inc.",
  line_type: "mobile",
};

const numlookupapi = {
  provider: "numlookupapi",
  valid: true,
  local_format: "(202) 867-5309",
  international_format: "+1 202 867 5309",
  country_name: "United States",
  country_prefix: "+1",
  location: "New York",
  carrier: "T-Mobile USA, Inc.",
  line_type: "mobile",
};

// Twilio reports the *correct* carrier here, disagreeing with the majority.
const twilio = {
  valid: true,
  national_format: "(202) 867-5309",
  line_type: "mobile",
  carrier_name: "Verizon Wireless",
};

const baseline = {
  valid: true,
  local: "(202) 867-5309",
  international: "+1 202 867 5309",
  e164: "+12028675309",
  countryCode: 1,
  country: "United States",
  carrier: "",
};

const rowByKey = (
  comparison: ReturnType<typeof buildComparison>,
  key: string
) => comparison.rows.find((row) => row.key === key);

describe("utils/providerComparison", () => {
  describe("#buildComparison", () => {
    it("flags the carrier row when providers disagree", () => {
      const comparison = buildComparison(null, {
        numverify,
        veriphone,
        numlookupapi,
        twilio,
      });

      const carrier = rowByKey(comparison, "carrier");
      expect(carrier).toBeDefined();
      expect(carrier?.disagreement).toBe(true);
    });

    it("does not flag rows where every provider agrees", () => {
      const comparison = buildComparison(null, {
        numverify,
        veriphone,
        numlookupapi,
        twilio,
      });

      expect(rowByKey(comparison, "valid")?.disagreement).toBe(false);
      expect(rowByKey(comparison, "line_type")?.disagreement).toBe(false);
    });

    it("does not flag number-format rows that differ only in punctuation", () => {
      const comparison = buildComparison(null, {
        numverify: {
          valid: true,
          international_format: "+1 202 867 5309",
          local_format: "(202) 867-5309",
          country_prefix: "+1",
        },
        veriphone: {
          valid: true,
          international_format: "+12028675309",
          local_format: "202-867-5309",
          country_prefix: "1",
        },
      });

      expect(rowByKey(comparison, "international_format")?.disagreement).toBe(
        false
      );
      expect(rowByKey(comparison, "national_format")?.disagreement).toBe(false);
      expect(rowByKey(comparison, "country_prefix")?.disagreement).toBe(false);
    });

    it("treats country name, ISO code and long form as the same country", () => {
      const comparison = buildComparison(null, {
        numverify: { valid: true, country_name: "United States" },
        veriphone: { valid: true, country_name: "United States of America" },
        numlookupapi: { valid: true, country_name: "USA" },
      });

      expect(rowByKey(comparison, "country_name")?.disagreement).toBe(false);
    });

    it("still flags genuinely different countries", () => {
      const comparison = buildComparison(null, {
        numverify: { valid: true, country_name: "United States" },
        veriphone: { valid: true, country_name: "Canada" },
      });

      expect(rowByKey(comparison, "country_name")?.disagreement).toBe(true);
    });

    it("does not flag a format row when a provider omits it but the rest agree", () => {
      const comparison = buildComparison(null, {
        numverify: { valid: true, international_format: "+1 202 867 5309" },
        veriphone: { valid: true, international_format: "+12028675309" },
        // Twilio returns no international format — its empty cell must not flag.
        twilio: { valid: true, national_format: "(202) 867-5309" },
      });

      expect(rowByKey(comparison, "international_format")?.disagreement).toBe(
        false
      );
    });

    it("treats agreement case-insensitively and ignores surrounding space", () => {
      const comparison = buildComparison(null, {
        numverify: { valid: true, carrier: "AT&T Mobility" },
        veriphone: { valid: true, carrier: "  at&t mobility " },
      });

      expect(rowByKey(comparison, "carrier")?.disagreement).toBe(false);
    });

    it("builds one column per provider that returned data, in a stable order", () => {
      const comparison = buildComparison(baseline, {
        twilio,
        numverify,
      });

      expect(comparison.columns.map((c) => c.key)).toEqual([
        "local",
        "numverify",
        "twilio",
      ]);
      expect(comparison.columns[0].label).toBe("Local (offline)");
    });

    it("omits providers that returned no usable fields", () => {
      const comparison = buildComparison(null, {
        numverify,
        abstract: {},
      });

      expect(comparison.columns.map((c) => c.key)).toEqual(["numverify"]);
    });

    it("omits the baseline column when no baseline is supplied", () => {
      const comparison = buildComparison(null, { numverify });
      expect(comparison.columns.map((c) => c.key)).toEqual(["numverify"]);
    });

    it("drops rows that no provider fills", () => {
      const comparison = buildComparison(null, { twilio });
      // Twilio never returns location, so the location row should not appear.
      expect(rowByKey(comparison, "location")).toBeUndefined();
    });

    it("renders empty cells as unfilled rather than dropping the column", () => {
      const comparison = buildComparison(null, { numverify, twilio });
      const country = rowByKey(comparison, "country_name");
      // Twilio has no country_name; its cell is present but unfilled.
      const twilioCell = country?.cells.find((c) => c.columnKey === "twilio");
      expect(twilioCell?.filled).toBe(false);
      expect(twilioCell?.value).toBe("");
    });
  });

  describe("baseline normalisation", () => {
    it("maps the numeric calling code onto the dial-prefix row, not country code", () => {
      const comparison = buildComparison(baseline, { numverify });

      const prefixCell = rowByKey(comparison, "country_prefix")?.cells.find(
        (c) => c.columnKey === "local"
      );
      expect(prefixCell?.value).toBe("+1");

      // The baseline must not populate the ISO country-code row (that would
      // fabricate a disagreement with Numverify's "US").
      const isoCell = rowByKey(comparison, "country_code")?.cells.find(
        (c) => c.columnKey === "local"
      );
      expect(isoCell?.filled).toBe(false);
    });

    it("keeps the local baseline aligned with providers on shared fields", () => {
      const comparison = buildComparison(baseline, {
        numverify,
        veriphone,
        numlookupapi,
      });
      // Baseline carrier is empty, providers agree on T-Mobile — the single
      // provider value set is size 1, so no disagreement.
      expect(rowByKey(comparison, "country_prefix")?.disagreement).toBe(false);
    });
  });

  describe("helpers", () => {
    it("orders known providers first, unknown providers last", () => {
      expect(
        __test__.orderProviders(["twilio", "mystery", "numverify"])
      ).toEqual(["numverify", "twilio", "mystery"]);
    });

    it("normalises the dial prefix with a leading plus", () => {
      expect(__test__.dialPrefix(1)).toBe("+1");
      expect(__test__.dialPrefix("+44")).toBe("+44");
      expect(__test__.dialPrefix(0)).toBe("");
      expect(__test__.dialPrefix(null)).toBe("");
    });

    it("reduces number-format rows to digits and leaves other rows folded", () => {
      expect(__test__.rowCompareKey("international_format", "+1 (202) 8")).toBe(
        "12028"
      );
      expect(__test__.rowCompareKey("country_prefix", "+1")).toBe("1");
      expect(__test__.rowCompareKey("carrier", "  Verizon Wireless ")).toBe(
        "verizon wireless"
      );
    });
  });

  describe("canonicalCountry", () => {
    it("maps codes, short and long forms to one canonical key", () => {
      const us = canonicalCountry("United States");
      expect(canonicalCountry("USA")).toBe(us);
      expect(canonicalCountry("US")).toBe(us);
      expect(canonicalCountry("United States of America")).toBe(us);
      expect(canonicalCountry("u.s.a.")).toBe(us);
    });

    it("distinguishes different countries", () => {
      expect(canonicalCountry("United Kingdom")).not.toBe(
        canonicalCountry("United States")
      );
    });

    it("falls back to a normalised form for unknown countries", () => {
      expect(canonicalCountry("  Freedonia ")).toBe(
        canonicalCountry("freedonia")
      );
    });

    it("returns empty string for blank input", () => {
      expect(canonicalCountry("")).toBe("");
      expect(canonicalCountry("   ")).toBe("");
    });
  });
});
