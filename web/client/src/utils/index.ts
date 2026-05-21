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
    (scanner: ScannerObject) =>
      !hiddenScannerNames.includes(scanner.name)
  );
};

const getScannerDisplayName = (name: string): string => {
  const labels: { [key: string]: string } = {
    googlecse: "Google Custom Search Engine",
    googlesearch: "Google Search",
    numverify: "Numverify",
    searxng: "SearXNG",
  };

  return labels[name] || name.charAt(0).toUpperCase() + name.slice(1);
};

const getPreferredScannerNames = (): string[] | null => {
  try {
    const value = window.localStorage.getItem(SCANNER_PREFERENCES_KEY);
    if (!value) {
      return null;
    }

    const parsed = JSON.parse(value);
    return Array.isArray(parsed) ? parsed.filter((name) => typeof name === "string") : null;
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

const getScannerRunOptions = (scannerName: string): { [key: string]: number } => {
  if (scannerName !== "searxng") {
    return {};
  }

  return {
    SEARXNG_DELAY_MS: getSearXNGDelayMs(),
  };
};

const getDefaultScannerNames = (
  scanners: ScannerAvailability[]
): string[] => {
  const configured = scanners
    .filter((scanner) => scanner.available)
    .map((scanner) => scanner.name);
  const preferred = getPreferredScannerNames();

  if (preferred === null) {
    return configured;
  }

  return preferred.filter((name) => configured.includes(name));
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
  getDefaultScannerNames,
  getPreferredScannerNames,
  setPreferredScannerNames,
  getSearXNGDelayMs,
  setSearXNGDelayMs,
  getScannerRunOptions,
};
export type { ScannerObject, ScannerAvailability };
