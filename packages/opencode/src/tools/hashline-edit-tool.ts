import { readFile, writeFile } from "node:fs/promises";
import { validateLineRefs } from "@solon/core";
import { z } from "zod";

const EditOperationSchema = z.object({
  op: z.enum(["replace", "append", "prepend"]),
  pos: z.string().optional().describe("LINE#ID anchor position"),
  end: z.string().optional().describe("LINE#ID end of range for replace"),
  lines: z
    .union([z.string(), z.array(z.string()), z.null()])
    .optional()
    .describe("Replacement text lines (null to delete)"),
});

export const HashlineEditInputSchema = z.object({
  filePath: z.string().describe("Absolute path to file"),
  edits: z
    .array(EditOperationSchema)
    .describe("Edit operations using LINE#ID refs"),
});

export type HashlineEditInput = z.infer<typeof HashlineEditInputSchema>;

interface EditOperation {
  op: "replace" | "append" | "prepend";
  lineNumber: number;
  endLineNumber?: number | undefined;
  newLines: string[];
}

export async function hashlineEditHandler(
  input: HashlineEditInput,
): Promise<string> {
  const content = await readFile(input.filePath, "utf-8");
  const fileLines = content.split("\n");

  // Remove trailing empty line from split
  if (fileLines[fileLines.length - 1] === "") fileLines.pop();

  // Collect all refs for validation
  const allRefs: string[] = [];
  for (const edit of input.edits) {
    if (edit.pos) allRefs.push(edit.pos);
    if (edit.end) allRefs.push(edit.end);
  }

  // Validate all refs against current file
  await validateLineRefs(allRefs, fileLines);

  // Resolve and sort edits bottom-up (descending line number)
  const resolved: EditOperation[] = [];
  for (const edit of input.edits) {
    const posLine = edit.pos
      ? Number.parseInt(edit.pos.split("#")[0] ?? "0", 10)
      : 0;
    const endLine = edit.end
      ? Number.parseInt(edit.end.split("#")[0] ?? "0", 10)
      : undefined;
    const newLines =
      edit.lines == null
        ? []
        : Array.isArray(edit.lines)
          ? edit.lines
          : [edit.lines];
    resolved.push({
      op: edit.op,
      lineNumber: posLine,
      endLineNumber: endLine,
      newLines,
    });
  }

  // Sort bottom-up so later edits don't shift earlier line numbers
  resolved.sort((a, b) => b.lineNumber - a.lineNumber);

  // Apply edits
  let editCount = 0;
  for (const op of resolved) {
    const idx = op.lineNumber > 0 ? op.lineNumber - 1 : 0;
    const endIdx = op.endLineNumber ? op.endLineNumber - 1 : idx;

    if (op.op === "replace") {
      fileLines.splice(idx, endIdx - idx + 1, ...op.newLines);
      editCount++;
    } else if (op.op === "append") {
      fileLines.splice(
        op.lineNumber > 0 ? idx + 1 : fileLines.length,
        0,
        ...op.newLines,
      );
      editCount++;
    } else if (op.op === "prepend") {
      fileLines.splice(idx, 0, ...op.newLines);
      editCount++;
    }
  }

  await writeFile(input.filePath, `${fileLines.join("\n")}\n`, "utf-8");
  return `Applied ${editCount} edit(s) to ${input.filePath}`;
}

// OpenCode tool definition shape (without SDK dependency)
export const hashlineEditToolDef = {
  description: `Edit files using hashline LINE#ID references. Read a file first with solon_read or Read to get LINE#ID tags (e.g., 42#WN|content), then reference them as "42#WN" for precise edits. Edits applied bottom-up, safe for multi-line changes.`,
  args: HashlineEditInputSchema,
  execute: hashlineEditHandler,
};
