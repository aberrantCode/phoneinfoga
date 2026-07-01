// Builds a side-by-side comparison of the phone-validation providers
// (Numverify, Veriphone, NumlookupAPI, Abstract, Twilio) plus the offline
// "local" libphonenumber baseline. Each provider returns an overlapping but
// not identical set of fields; this module normalises them into a shared row
// vocabulary so the UI can render a property x provider matrix and flag rows
// where the providers disagree (e.g. one reports a different carrier).
//
// All functions here are pure and side-effect free so they can be unit tested
// without mounting a component.

export interface ComparisonColumn {
  key: string; // scanId, or "local" for the offline baseline
  label: string; // display name
}

export interface ComparisonCell {
  columnKey: string;
  value: string; // display value; "" is rendered as an em dash by the view
  filled: boolean; // value !== "" — convenience flag for the template
}

export interface ComparisonRow {
  key: string;
  label: string;
  cells: ComparisonCell[]; // aligned index-for-index with `columns`
  // True when the filled cells hold more than one distinct value. The view
  // tints the whole row rather than branding any single cell as the outlier —
  // the majority is not necessarily correct.
  disagreement: boolean;
}

export interface Comparison {
  columns: ComparisonColumn[];
  rows: ComparisonRow[];
}

type Raw = Record<string, unknown>;

const BASELINE_KEY = "local";

// Column order for the providers we know about. Unknown providers are appended
// in the order they appear in the results map.
const PROVIDER_ORDER = [
  "numverify",
  "veriphone",
  "numlookupapi",
  "abstract",
  "twilio",
];

const PROVIDER_LABELS: { [key: string]: string } = {
  local: "Local (offline)",
  numverify: "Numverify",
  veriphone: "Veriphone",
  numlookupapi: "NumlookupAPI",
  abstract: "Abstract",
  twilio: "Twilio",
};

// The canonical rows, in display order. A row is dropped entirely if no column
// fills it, so providers that omit a field never create empty clutter.
const ROW_DEFS: { key: string; label: string }[] = [
  { key: "valid", label: "Valid" },
  { key: "international_format", label: "International format" },
  { key: "national_format", label: "National / local" },
  { key: "country_name", label: "Country" },
  { key: "country_code", label: "Country code" },
  { key: "country_prefix", label: "Dial prefix" },
  { key: "location", label: "Location" },
  { key: "carrier", label: "Carrier" },
  { key: "line_type", label: "Line type" },
];

const columnLabel = (key: string): string =>
  PROVIDER_LABELS[key] || key.charAt(0).toUpperCase() + key.slice(1);

const str = (value: unknown): string => {
  if (value === undefined || value === null) {
    return "";
  }
  return String(value).trim();
};

const boolStr = (value: unknown): string => {
  if (value === true) {
    return "Yes";
  }
  if (value === false) {
    return "No";
  }
  return str(value);
};

// The offline baseline stores countryCode as the numeric calling code (e.g. 1),
// so it feeds the dial-prefix row as "+1" rather than the ISO country-code row.
const dialPrefix = (value: unknown): string => {
  const raw = str(value);
  if (raw === "" || raw === "0") {
    return "";
  }
  return raw.startsWith("+") ? raw : `+${raw}`;
};

// Maps a single provider's raw JSON into the shared row vocabulary. Providers
// that share Numverify's field names fall through to the default branch.
const normaliseProvider = (
  columnKey: string,
  data: Raw
): { [k: string]: string } => {
  switch (columnKey) {
    case BASELINE_KEY:
      return {
        valid: boolStr(data.valid),
        international_format: str(data.international),
        national_format: str(data.local),
        country_name: str(data.country),
        country_prefix: dialPrefix(data.countryCode),
        carrier: str(data.carrier),
      };
    case "twilio":
      return {
        valid: boolStr(data.valid),
        national_format: str(data.national_format),
        carrier: str(data.carrier_name),
        line_type: str(data.line_type),
      };
    default:
      // Numverify and the ValidationScannerResponse providers (Veriphone,
      // NumlookupAPI, Abstract) share these JSON field names.
      return {
        valid: boolStr(data.valid),
        international_format: str(data.international_format),
        national_format: str(data.local_format),
        country_name: str(data.country_name),
        country_code: str(data.country_code),
        country_prefix: str(data.country_prefix),
        location: str(data.location),
        carrier: str(data.carrier),
        line_type: str(data.line_type),
      };
  }
};

const hasAnyValue = (normalised: { [k: string]: string }): boolean =>
  Object.keys(normalised).some((key) => normalised[key] !== "");

const compareKey = (value: string): string => value.trim().toLowerCase();

// Orders the provider result keys: known providers first (in PROVIDER_ORDER),
// then any unknown providers in insertion order.
const orderProviders = (keys: string[]): string[] => {
  const known = PROVIDER_ORDER.filter((name) => keys.includes(name));
  const unknown = keys.filter((name) => !PROVIDER_ORDER.includes(name));
  return [...known, ...unknown];
};

/**
 * Builds the comparison matrix.
 *
 * @param baseline the offline `localData` metadata, or null to omit the Local
 *   column entirely.
 * @param results  live scanner results keyed by scanId (e.g. numverify, twilio).
 *   Only providers that returned at least one non-empty field become columns.
 */
export function buildComparison(
  baseline: Raw | null,
  results: { [scanId: string]: Raw }
): Comparison {
  const columns: ComparisonColumn[] = [];
  const normalisedByColumn: { [key: string]: { [k: string]: string } } = {};

  if (baseline) {
    const normalised = normaliseProvider(BASELINE_KEY, baseline);
    if (hasAnyValue(normalised)) {
      columns.push({ key: BASELINE_KEY, label: columnLabel(BASELINE_KEY) });
      normalisedByColumn[BASELINE_KEY] = normalised;
    }
  }

  orderProviders(Object.keys(results || {})).forEach((scanId) => {
    const data = results[scanId];
    if (!data) {
      return;
    }
    const normalised = normaliseProvider(scanId, data as Raw);
    if (hasAnyValue(normalised)) {
      columns.push({ key: scanId, label: columnLabel(scanId) });
      normalisedByColumn[scanId] = normalised;
    }
  });

  const rows: ComparisonRow[] = ROW_DEFS.map((def) => {
    const cells: ComparisonCell[] = columns.map((column) => {
      const value = normalisedByColumn[column.key][def.key] || "";
      return { columnKey: column.key, value, filled: value !== "" };
    });

    const distinct = new Set(
      cells.filter((cell) => cell.filled).map((cell) => compareKey(cell.value))
    );

    return {
      key: def.key,
      label: def.label,
      cells,
      disagreement: distinct.size > 1,
    };
  }).filter((row) => row.cells.some((cell) => cell.filled));

  return { columns, rows };
}

export const __test__ = {
  normaliseProvider,
  orderProviders,
  dialPrefix,
};
