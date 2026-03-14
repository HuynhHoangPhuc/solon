import { describe, it, expect } from "bun:test";
import {
  isPrivacySensitive, isSafeFile, hasApprovalPrefix, stripApprovalPrefix,
  extractPaths, checkPrivacy, APPROVED_PREFIX,
} from "../../lib/privacy-checker.ts";

describe("isSafeFile", () => {
  it("allows .env.example", () => expect(isSafeFile(".env.example")).toBe(true));
  it("allows .env.sample", () => expect(isSafeFile(".env.sample")).toBe(true));
  it("allows .env.template", () => expect(isSafeFile(".env.template")).toBe(true));
  it("allows path/to/.env.example", () => expect(isSafeFile("path/to/.env.example")).toBe(true));
  it("blocks .env (not safe)", () => expect(isSafeFile(".env")).toBe(false));
  it("blocks credentials.json (not safe)", () => expect(isSafeFile("credentials.json")).toBe(false));
});

describe("isPrivacySensitive", () => {
  it("detects .env", () => expect(isPrivacySensitive(".env")).toBe(true));
  it("detects .env.local", () => expect(isPrivacySensitive(".env.local")).toBe(true));
  it("detects .env.production", () => expect(isPrivacySensitive(".env.production")).toBe(true));
  it("detects credentials.json", () => expect(isPrivacySensitive("credentials.json")).toBe(true));
  it("detects secrets.yaml", () => expect(isPrivacySensitive("secrets.yaml")).toBe(true));
  it("detects private.pem", () => expect(isPrivacySensitive("private.pem")).toBe(true));
  it("detects id_rsa", () => expect(isPrivacySensitive("id_rsa")).toBe(true));
  it("detects id_ed25519", () => expect(isPrivacySensitive("id_ed25519")).toBe(true));
  it("allows .env.example (safe file)", () => expect(isPrivacySensitive(".env.example")).toBe(false));
  it("allows src/main.ts", () => expect(isPrivacySensitive("src/main.ts")).toBe(false));
  it("allows package.json", () => expect(isPrivacySensitive("package.json")).toBe(false));
  it("decodes URI-encoded paths (.env%2Elocal → .env.local → sensitive)", () => {
    // %2E decodes to '.' so .env%2Elocal → .env.local which IS sensitive
    expect(isPrivacySensitive(".env%2Elocal")).toBe(true);
  });
});

describe("hasApprovalPrefix / stripApprovalPrefix", () => {
  it("detects APPROVED: prefix", () => expect(hasApprovalPrefix("APPROVED:.env")).toBe(true));
  it("rejects missing prefix", () => expect(hasApprovalPrefix(".env")).toBe(false));
  it("strips prefix", () => expect(stripApprovalPrefix("APPROVED:.env")).toBe(".env"));
  it("returns unchanged if no prefix", () => expect(stripApprovalPrefix(".env")).toBe(".env"));
});

describe("extractPaths", () => {
  it("extracts file_path", () => {
    const paths = extractPaths({ file_path: ".env" });
    expect(paths.some((p) => p.value === ".env")).toBe(true);
  });

  it("extracts path field", () => {
    const paths = extractPaths({ path: "credentials.json" });
    expect(paths.some((p) => p.value === "credentials.json")).toBe(true);
  });

  it("extracts .env from bash command", () => {
    const paths = extractPaths({ command: "cat .env" });
    expect(paths.some((p) => p.value === ".env")).toBe(true);
  });

  it("extracts APPROVED: path from bash command", () => {
    const paths = extractPaths({ command: "cat APPROVED:.env" });
    expect(paths.some((p) => p.value.startsWith("APPROVED:"))).toBe(true);
  });

  it("extracts env var assignment", () => {
    const paths = extractPaths({ command: "FILE=.env.local cat $FILE" });
    expect(paths.some((p) => p.value === ".env.local")).toBe(true);
  });

  it("returns empty for no sensitive paths", () => {
    const paths = extractPaths({ file_path: "src/main.ts" });
    expect(paths).toHaveLength(1); // file_path is always extracted
  });
});

describe("checkPrivacy", () => {
  it("blocks .env read without approval", () => {
    const result = checkPrivacy({ toolName: "Read", toolInput: { file_path: ".env" } });
    expect(result.blocked).toBe(true);
    expect(result.promptData).toBeTruthy();
  });

  it("allows APPROVED:.env", () => {
    const result = checkPrivacy({ toolName: "Read", toolInput: { file_path: `${APPROVED_PREFIX}.env` } });
    expect(result.blocked).toBe(false);
    expect(result.approved).toBe(true);
  });

  it("warns but does not block Bash accessing .env", () => {
    const result = checkPrivacy({ toolName: "Bash", toolInput: { command: "cat .env" } });
    expect(result.blocked).toBe(false);
    expect(result.isBash).toBe(true);
  });

  it("allows .env.example", () => {
    const result = checkPrivacy({ toolName: "Read", toolInput: { file_path: ".env.example" } });
    expect(result.blocked).toBe(false);
  });

  it("allows normal file", () => {
    const result = checkPrivacy({ toolName: "Read", toolInput: { file_path: "src/main.ts" } });
    expect(result.blocked).toBe(false);
  });

  it("skips all checks when disabled option set", () => {
    const result = checkPrivacy({
      toolName: "Read",
      toolInput: { file_path: ".env" },
      options: { disabled: true },
    });
    expect(result.blocked).toBe(false);
  });
});
