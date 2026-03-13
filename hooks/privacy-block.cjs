'use strict';

/**
 * privacy-block.cjs — PreToolUse hook for Read and Bash
 * Blocks access to sensitive files (.env, credentials, keys, etc.)
 * and prompts the user for approval using the @@PRIVACY_PROMPT@@ protocol.
 */

const SENSITIVE_PATTERNS = [
  /^\.env(\..+)?$/i,           // .env, .env.local, .env.production
  /credentials/i,               // *credentials*
  /secret/i,                    // *secret*
  /\.pem$/i,                    // *.pem
  /\.key$/i,                    // *.key
  /\.p12$/i,                    // *.p12
  /\.pfx$/i,                    // *.pfx
  /id_rsa/i,                    // id_rsa, id_rsa.pub
  /id_ed25519/i,                // id_ed25519
  /\.ssh\//i,                   // .ssh/ directory
  /password/i,                  // *password*
  /api[_-]?key/i,              // api_key, apikey, api-key
  /auth[_-]?token/i,           // auth_token
  /access[_-]?token/i,         // access_token
  /private[_-]?key/i,          // private_key
];

function isSensitivePath(filePath) {
  if (!filePath || typeof filePath !== 'string') return false;
  // Check the basename and full path
  const parts = filePath.split(/[/\\]/);
  const basename = parts[parts.length - 1] || '';
  return SENSITIVE_PATTERNS.some(re => re.test(basename) || re.test(filePath));
}

function extractFilePath(toolName, toolInput) {
  if (toolName === 'Read') {
    return toolInput.file_path || toolInput.path || '';
  }
  if (toolName === 'Bash') {
    const cmd = toolInput.command || '';
    // Extract file path from common patterns: cat .env, source .env, etc.
    const match = cmd.match(/(?:cat|source|\.)\s+([^\s;|&]+)/);
    return match ? match[1] : '';
  }
  return '';
}

function main() {
  let input = '';
  process.stdin.setEncoding('utf8');
  process.stdin.on('data', chunk => { input += chunk; });
  process.stdin.on('end', () => {
    try {
      const event = JSON.parse(input);
      const tool = event.tool_name || '';
      const toolInput = event.tool_input || {};

      const filePath = extractFilePath(tool, toolInput);

      if (filePath && isSensitivePath(filePath)) {
        // Use the @@PRIVACY_PROMPT@@ protocol for user approval
        const promptData = {
          questions: [{
            question: `I need to read "${filePath}" which may contain sensitive data. Do you approve?`,
            header: 'Sensitive File Access',
            options: [
              { label: 'Yes, approve access', description: `Allow reading ${filePath} this time` },
              { label: 'No, skip this file', description: 'Continue without accessing this file' },
            ],
            multiSelect: false,
          }],
        };

        const result = {
          decision: 'block',
          reason: `@@PRIVACY_PROMPT_START@@${JSON.stringify(promptData)}@@PRIVACY_PROMPT_END@@`,
        };
        process.stdout.write(JSON.stringify(result));
      } else {
        process.stdout.write(JSON.stringify({ decision: 'allow' }));
      }
    } catch (e) {
      // Fail open: allow on parse error
      process.stdout.write(JSON.stringify({ decision: 'allow' }));
    }
  });
}

main();
