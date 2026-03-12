import { readFile } from "node:fs/promises";

export async function suppressWriteOutput(
  input: { tool_input?: Record<string, unknown> },
  output: { output: string },
): Promise<void> {
  const filePath = (input.tool_input?.path ?? input.tool_input?.file_path) as
    | string
    | undefined;

  let lineCount = 0;
  if (filePath) {
    try {
      const content = await readFile(filePath, "utf-8");
      lineCount = content.split("\n").length;
      if (content.endsWith("\n")) lineCount--;
    } catch {
      lineCount = 0;
    }
  }

  output.output =
    lineCount > 0
      ? `File written successfully. ${lineCount} lines written.`
      : "File written successfully.";
}
