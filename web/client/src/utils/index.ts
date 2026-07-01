import axios from "axios";
import config from "../config/index";
interface ScannerObject {
  name: string;
  description: string;
}

interface ScannerAvailability extends ScannerObject {
  available: boolean;
  error?: string;
}

const SCANNER_PREFERENCES_KEY = "phoneinfoga.defaultScanners";
const SEARXNG_DELAY_KEY = "phoneinfoga.searxngDelayMs";
const scannerProbeNumber = "12022942000";
const hiddenScannerNames = ["local", "ovh"];

const formatNumber = (number: string): string => {
  return number.replace(/[_\W]+/g, "");
};

const isValid = (number: string): boolean => {
  const formatted = formatNumber(number);

  return formatted.match(/^[0-9]+$/) !== null && formatted.length > 2;
};

const formatString = (string: string): string => {
  return string.replace(/([A-Z])/g, " $1").trim();
};

const getScanners = async (): Promise<ScannerObject[]> => {
  const res = await axios.get(`${config.apiUrl}/v2/scanners`);

  // Local is already shown in the information panel. OVH is intentionally
  // hidden from the web scanners panel because the public lookup is low-signal.
  return res.data.scanners.filter(
    (scanner: ScannerObject) => !hiddenScannerNames.includes(scanner.name)
  );
};

const getScannerDisplayName = (name: string): string => {
  const labels: { [key: string]: string } = {
    googlesearch: "Google Search",
    numverify: "Numverify",
    searxng: "SearXNG",
  };

  return labels[name] || name.charAt(0).toUpperCase() + name.slice(1);
};

const getScannerDescription = (name: string): string => {
  const descriptions: { [key: string]: string } = {
    local:
      "Offline libphonenumber analysis: formatting, country, line type and carrier — no external calls.",
    numverify:
      "Numverify API: real-time validation, carrier, line type and location.",
    veriphone: "Veriphone API: validation, carrier, line type and country.",
    abstract: "Abstract API: phone validation, carrier and line type.",
    numlookupapi: "NumLookupAPI: validation, carrier and location lookup.",
    ovh: "Looks up OVH Telecom's public number ranges (French numbers).",
    googlesearch:
      "Generates ready-to-run Google dork queries (footprints) across social, docs and reputation sites.",
    searxng:
      "Runs the Google dork footprints through a SearXNG instance and returns matches.",
    hlr: "HLR lookup: live network/roaming status and ported-number detection.",
    ipqualityscore:
      "IPQualityScore: fraud/risk score, active-line and line-type checks.",
    nanpa:
      "North American Numbering Plan: validates US/Canada area codes and exchanges.",
    serpapi: "SerpApi: search-engine results mentioning the number.",
    twilio: "Twilio Lookup: carrier, line type and caller-name (CNAM).",
    breach: "Checks data-breach datasets for records linked to the number.",
    validation: "Validates the number's format and basic metadata.",
  };

  return descriptions[name] || `${getScannerDisplayName(name)} scanner.`;
};

const getPreferredScannerNames = (): string[] | null => {
  try {
    const value = window.localStorage.getItem(SCANNER_PREFERENCES_KEY);
    if (!value) {
      return null;
    }

    const parsed = JSON.parse(value);
    return Array.isArray(parsed)
      ? parsed.filter((name) => typeof name === "string")
      : null;
  } catch {
    return null;
  }
};

const setPreferredScannerNames = (scannerNames: string[]): void => {
  window.localStorage.setItem(
    SCANNER_PREFERENCES_KEY,
    JSON.stringify(scannerNames)
  );
};

const getSearXNGDelayMs = (): number => {
  const raw = window.localStorage.getItem(SEARXNG_DELAY_KEY);
  const value = raw ? Number(raw) : 750;
  if (!Number.isFinite(value) || value < 0) {
    return 750;
  }
  return Math.floor(value);
};

const setSearXNGDelayMs = (delayMs: number): void => {
  const value = Number.isFinite(delayMs) && delayMs >= 0 ? delayMs : 750;
  window.localStorage.setItem(SEARXNG_DELAY_KEY, String(Math.floor(value)));
};

const getScannerRunOptions = (
  scannerName: string
): { [key: string]: number } => {
  if (scannerName !== "searxng") {
    return {};
  }

  return {
    SEARXNG_DELAY_MS: getSearXNGDelayMs(),
  };
};

const getDefaultScannerNames = (scanners: ScannerAvailability[]): string[] => {
  const configured = scanners
    .filter((scanner) => scanner.available)
    .map((scanner) => scanner.name);
  const preferred = getPreferredScannerNames();

  if (preferred === null) {
    return configured;
  }

  return preferred.filter((name) => configured.includes(name));
};

interface NumberMetadata {
  valid: boolean;
  rawLocal: string;
  local: string;
  e164: string;
  international: string;
  countryCode: number;
  country: string;
  carrier: string;
}

interface LookupResult {
  scanner: string;
  status: string;
  errorMessage?: string;
  raw: unknown;
  durationMs: number;
  startedAt: string;
  finishedAt: string;
}

interface LookupDetail {
  id: string;
  status: string;
  createdAt: string;
  completedAt: string | null;
  clientIp: string;
  userAgent: string;
  number: NumberMetadata;
  scannersRequested: string[];
  results: LookupResult[];
}

interface LookupSummary {
  id: string;
  e164: string;
  status: string;
  scannersRequested: string[];
  createdAt: string;
  completedAt: string | null;
}

interface CreateLookupResult {
  id: string;
  number: NumberMetadata;
  scannersRequested: string[];
  clientIp: string;
  createdAt: string;
  status: string;
}

interface CloseLookupResult {
  id: string;
  status: string;
  scannersRequested: string[];
  createdAt: string;
  completedAt: string | null;
}

// createLookup records a new lookup request (before any scanner runs) and returns
// its id and captured metadata.
const createLookup = async (
  number: string,
  scanners: string[]
): Promise<CreateLookupResult> => {
  const res = await axios.post(`${config.apiUrl}/v2/lookups`, {
    number,
    scanners,
  });

  return res.data;
};

// closeLookup finalizes a lookup, computing its complete/partial status.
const closeLookup = async (id: string): Promise<CloseLookupResult> => {
  const res = await axios.post(`${config.apiUrl}/v2/lookups/${id}/close`);

  return res.data;
};

// getLookup returns a lookup's full detail (metadata + all scanner results).
const getLookup = async (id: string): Promise<LookupDetail> => {
  const res = await axios.get(`${config.apiUrl}/v2/lookups/${id}`);

  return res.data;
};

// getLatestLookup returns the most recent lookup for a number, or null when none
// exists (HTTP 404) so callers can fall back to running a fresh lookup.
const getLatestLookup = async (
  number: string
): Promise<LookupDetail | null> => {
  const res = await axios.get(`${config.apiUrl}/v2/lookups/latest`, {
    params: { number },
    validateStatus: (status: number) => status === 200 || status === 404,
  });

  return res.status === 200 ? res.data : null;
};

// listLookups returns a number's lookup summaries, newest first.
const listLookups = async (
  number: string,
  limit?: number
): Promise<LookupSummary[]> => {
  const params: { number: string; limit?: number } = { number };
  if (typeof limit === "number") {
    params.limit = limit;
  }

  const res = await axios.get(`${config.apiUrl}/v2/lookups`, { params });

  return res.data.lookups;
};

const getScannerAvailability = async (): Promise<ScannerAvailability[]> => {
  const scanners = await getScanners();

  return Promise.all(
    scanners.map(async (scanner) => {
      try {
        const res = await axios.post(
          `${config.apiUrl}/v2/scanners/${scanner.name}/dryrun`,
          {
            number: scannerProbeNumber,
          },
          {
            validateStatus: () => true,
          }
        );

        return {
          ...scanner,
          available: res.data.success === true,
          error: res.data.error,
        };
      } catch (error) {
        return {
          ...scanner,
          available: false,
          error: String(error),
        };
      }
    })
  );
};

export {
  formatNumber,
  isValid,
  formatString,
  getScanners,
  getScannerAvailability,
  getScannerDisplayName,
  getScannerDescription,
  getDefaultScannerNames,
  getPreferredScannerNames,
  setPreferredScannerNames,
  getSearXNGDelayMs,
  setSearXNGDelayMs,
  getScannerRunOptions,
  createLookup,
  closeLookup,
  getLookup,
  getLatestLookup,
  listLookups,
};
export type {
  ScannerObject,
  ScannerAvailability,
  NumberMetadata,
  LookupResult,
  LookupDetail,
  LookupSummary,
  CreateLookupResult,
  CloseLookupResult,
};
