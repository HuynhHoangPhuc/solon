import { z } from "zod";
import { readFileWithHashlines } from "../utils/file-reader.js";

export const solonReadInputSchema = z.object({
  file_path: z.string().describe("Absolute path to the file to read"),
  offset: z
    .number()
    .optional()
    .describe("Line number to start reading from (1-based)"),
  limit: z.number().optional().describe("Maximum number of lines to read"),
});

export type SolonReadInput = z.infer<typeof solonReadInputSchema>;

export const SOLON_READ_DESCRIPTION =
  "Read a file with hashline-annotated line references (N#HH|content format). " +
  "Use these LINE#ID refs in Edit tool's old_string for precise, token-efficient edits. " +
  "Prefer this over the native Read tool when you plan to edit the file.";

export async function solonReadHandler(
  input: SolonReadInput,
): Promise<{ content: Array<{ type: "text"; text: string }> }> {
  // Basic security: require absolute path
  if (!input.file_path.startsWith("/")) {
    throw new Error("file_path must be an absolute path");
  }

  const opts: import("../utils/file-reader.js").FileReadOptions = {};
  if (input.offset !== undefined) opts.offset = input.offset;
  if (input.limit !== undefined) opts.limit = input.limit;

  const result = await readFileWithHashlines(input.file_path, opts);

  return {
    content: [{ type: "text" as const, text: result.content }],
  };
}
