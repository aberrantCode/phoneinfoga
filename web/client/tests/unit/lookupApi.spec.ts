// Auto-mock axios so the lookup API client never performs a real request.
jest.mock("axios");
import axios from "axios";

import {
  createLookup,
  closeLookup,
  getLookup,
  getLatestLookup,
  listLookups,
} from "../../src/utils";

const mockedAxios = axios as jest.Mocked<typeof axios>;

describe("src/utils lookup API", () => {
  beforeEach(() => {
    mockedAxios.get.mockReset();
    mockedAxios.post.mockReset();
  });

  describe("#createLookup", () => {
    it("POSTs number and scanners and returns the record", async () => {
      mockedAxios.post.mockResolvedValue({
        data: { id: "abc", status: "pending", scannersRequested: ["local"] },
      });

      const res = await createLookup("14152229670", ["local"]);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        expect.stringContaining("/v2/lookups"),
        { number: "14152229670", scanners: ["local"] }
      );
      expect(res.id).toBe("abc");
      expect(res.status).toBe("pending");
    });
  });

  describe("#closeLookup", () => {
    it("POSTs to the close endpoint for the id", async () => {
      mockedAxios.post.mockResolvedValue({
        data: { id: "abc", status: "complete" },
      });

      const res = await closeLookup("abc");

      expect(mockedAxios.post).toHaveBeenCalledWith(
        expect.stringContaining("/v2/lookups/abc/close")
      );
      expect(res.status).toBe("complete");
    });
  });

  describe("#getLookup", () => {
    it("GETs the detail for the id", async () => {
      mockedAxios.get.mockResolvedValue({
        status: 200,
        data: { id: "abc", results: [] },
      });

      const res = await getLookup("abc");

      expect(mockedAxios.get).toHaveBeenCalledWith(
        expect.stringContaining("/v2/lookups/abc")
      );
      expect(res.id).toBe("abc");
    });
  });

  describe("#getLatestLookup", () => {
    it("returns the detail on 200", async () => {
      mockedAxios.get.mockResolvedValue({
        status: 200,
        data: { id: "abc", results: [] },
      });

      const res = await getLatestLookup("14152229670");

      expect(mockedAxios.get).toHaveBeenCalledWith(
        expect.stringContaining("/v2/lookups/latest"),
        expect.objectContaining({ params: { number: "14152229670" } })
      );
      expect(res).not.toBeNull();
      expect(res?.id).toBe("abc");
    });

    it("returns null on 404 (no prior lookup)", async () => {
      mockedAxios.get.mockResolvedValue({ status: 404, data: {} });

      const res = await getLatestLookup("14152229670");

      expect(res).toBeNull();
    });
  });

  describe("#listLookups", () => {
    it("GETs summaries scoped to the number", async () => {
      mockedAxios.get.mockResolvedValue({
        status: 200,
        data: { lookups: [{ id: "a" }, { id: "b" }] },
      });

      const res = await listLookups("14152229670", 20);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        expect.stringContaining("/v2/lookups"),
        expect.objectContaining({
          params: { number: "14152229670", limit: 20 },
        })
      );
      expect(res).toHaveLength(2);
    });
  });
});
