// Shared TypeScript interfaces for all hook I/O
// Use `as const` objects instead of enum (Node --experimental-strip-types compat)

export const PermissionMode = {
  DEFAULT: "default",
  PLAN: "plan",
  ACCEPT_EDITS: "acceptEdits",
  DONT_ASK: "dontAsk",
  BYPASS: "bypassPermissions",
} as const;
export type PermissionMode = (typeof PermissionMode)[keyof typeof PermissionMode];

export const PermissionDecision = {
  ALLOW: "allow",
  DENY: "deny",
  ASK: "ask",
} as const;
export type PermissionDecision = (typeof PermissionDecision)[keyof typeof PermissionDecision];

// Base hook input (common fields from stdin JSON)
export interface HookInput {
  session_id: string;
  transcript_path: string;
  cwd: string;
  permission_mode: PermissionMode;
  hook_event_name: string;
  agent_id?: string;
  agent_type?: string;
}

// Event-specific inputs
export interface SessionStartInput extends HookInput {
  source: "startup" | "resume" | "clear" | "compact";
  model: string;
}

export interface SubagentStartInput extends HookInput {
  agent_type: string;
  agent_id: string;
}

export interface SubagentStopInput extends HookInput {
  agent_type: string;
}

export interface UserPromptSubmitInput extends HookInput {
  prompt: string;
}

export interface PreToolUseInput extends HookInput {
  tool_name: string;
  tool_input: Record<string, unknown>;
}

export interface PostToolUseInput extends HookInput {
  tool_name: string;
  tool_input: Record<string, unknown>;
  tool_output?: string;
}

export interface TaskCompletedInput extends HookInput {
  task_id: string;
  task_subject: string;
  task_description?: string;
  teammate_name: string;
  team_name: string;
}

export interface TeammateIdleInput extends HookInput {
  teammate_name: string;
  team_name: string;
}

export interface StopInput extends HookInput {
  hook_event_name: "Stop";
}

// Hook output types
export interface HookOutput {
  continue?: boolean;
  additionalContext?: string;
  hookSpecificOutput?: HookSpecificOutput;
}

export interface PreToolUseOutput {
  hookEventName: "PreToolUse";
  permissionDecision: PermissionDecision;
  permissionDecisionReason?: string;
  additionalContext?: string;
}

export interface SessionStartOutput {
  hookEventName: "SessionStart";
  additionalContext?: string;
}

export interface SubagentStartOutput {
  hookEventName: "SubagentStart";
  additionalContext?: string;
}

export interface TaskCompletedOutput {
  hookEventName: "TaskCompleted";
  additionalContext?: string;
}

export interface TeammateIdleOutput {
  hookEventName: "TeammateIdle";
  additionalContext?: string;
}

export type HookSpecificOutput =
  | PreToolUseOutput
  | SessionStartOutput
  | SubagentStartOutput
  | TaskCompletedOutput
  | TeammateIdleOutput;

// Config types
export interface SLConfig {
  plan: PlanConfig;
  paths: PathsConfig;
  docs: DocsConfig;
  locale: LocaleConfig;
  trust: TrustConfig;
  project: ProjectConfig;
  skills: SkillsConfig;
  hooks: Record<string, boolean>;
  assertions: string[];
  codingLevel: number;
  statusline: string;
  subagent?: SubagentConfig;
}

export interface PlanConfig {
  namingFormat: string;
  dateFormat: string;
  issuePrefix: string | null;
  reportsDir: string;
  resolution: {
    order: string[];
    branchPattern: string;
  };
  validation: {
    mode: string;
    minQuestions: number;
    maxQuestions: number;
    focusAreas: string[];
  };
}

export interface PathsConfig {
  docs: string;
  plans: string;
}

export interface DocsConfig {
  maxLoc: number;
}

export interface LocaleConfig {
  thinkingLanguage: string | null;
  responseLanguage: string | null;
}

export interface TrustConfig {
  passphrase: string | null;
  enabled: boolean;
}

export interface ProjectConfig {
  type: string;
  packageManager: string;
  framework: string;
}

export interface SkillsConfig {
  research: { useGemini: boolean };
}

export interface SubagentConfig {
  agents?: Record<string, { contextPrefix?: string }>;
}

// Plan resolution result
export interface PlanResolution {
  path: string | null;
  resolvedBy: "session" | "branch" | null;
}

// Session state
export interface SessionState {
  sessionOrigin: string;
  activePlan: string | null;
  suggestedPlan: string | null;
  timestamp: number;
  source: string;
}

// Project detection result
export interface ProjectInfo {
  type: string;
  packageManager: string | null;
  framework: string | null;
  pythonVersion: string | null;
  nodeVersion: string;
  gitBranch: string | null;
  gitRoot: string | null;
  gitUrl: string | null;
  osPlatform: string;
  user: string;
  locale: string;
  timezone: string;
}

// Scout block result
export interface ScoutBlockResult {
  blocked: boolean;
  path?: string;
  pattern?: string;
  reason?: string;
  isBroadPattern?: boolean;
  suggestions?: string[];
  isAllowedCommand?: boolean;
}

// Privacy check result
export interface PrivacyCheckResult {
  blocked: boolean;
  filePath?: string;
  reason?: string;
  approved?: boolean;
  isBash?: boolean;
  suspicious?: boolean;
  promptData?: Record<string, unknown>;
}

// Notification provider interface
export interface NotificationProvider {
  name: string;
  isEnabled: (env: Record<string, string>) => boolean;
  send: (input: HookInput, env: Record<string, string>) => Promise<NotificationResult>;
}

export interface NotificationResult {
  success: boolean;
  error?: string;
  throttled?: boolean;
}
