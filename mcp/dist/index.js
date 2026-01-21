#!/usr/bin/env node

// src/index.ts
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  ErrorCode,
  McpError
} from "@modelcontextprotocol/sdk/types.js";

// src/tools/list_files.ts
import * as fs2 from "fs";
import * as path2 from "path";

// src/path.ts
import * as fs from "fs";
import * as path from "path";
function validatePath(userPath, rootPath2) {
  if (userPath.includes("\0")) {
    throw new Error("Invalid path: contains null bytes");
  }
  if (path.isAbsolute(userPath)) {
    throw new Error("Absolute paths are not allowed");
  }
  const resolved = path.resolve(rootPath2, userPath);
  let real;
  let realRoot;
  try {
    realRoot = fs.realpathSync(rootPath2);
  } catch (err) {
    throw new Error(`Root path does not exist: ${rootPath2}`);
  }
  try {
    real = fs.realpathSync(resolved);
  } catch (err) {
    const parent = path.dirname(resolved);
    try {
      const realParent = fs.realpathSync(parent);
      if (!realParent.startsWith(realRoot + path.sep) && realParent !== realRoot) {
        throw new Error("Path outside root directory");
      }
      return resolved;
    } catch {
      throw new Error(`Path does not exist: ${userPath}`);
    }
  }
  if (!real.startsWith(realRoot + path.sep) && real !== realRoot) {
    throw new Error("Path outside root directory");
  }
  return real;
}

// src/tools/list_files.ts
function listFiles(rootPath2, userPath = ".", depth = 1) {
  const dirPath = validatePath(userPath, rootPath2);
  depth = Math.max(1, Math.min(depth, 5));
  const stat = fs2.statSync(dirPath);
  if (!stat.isDirectory()) {
    throw new Error("Path is not a directory");
  }
  return { entries: listDirectory(dirPath, depth) };
}
function listDirectory(dir, depth) {
  const entries = fs2.readdirSync(dir, { withFileTypes: true });
  return entries.map((entry) => {
    const isDir = entry.isDirectory();
    const result = {
      name: entry.name,
      type: isDir ? "directory" : "file"
    };
    if (isDir && depth > 1) {
      try {
        result.children = listDirectory(path2.join(dir, entry.name), depth - 1);
      } catch {
        result.children = [];
      }
    }
    return result;
  });
}

// src/tools/read_file.ts
import * as fs3 from "fs";
var MAX_FILE_SIZE = 10 * 1024 * 1024;
var MAX_LINES = 1e3;
function readFile(rootPath2, userPath, offset = 0, limit = 500) {
  const filePath = validatePath(userPath, rootPath2);
  const stat = fs3.statSync(filePath);
  if (!stat.isFile()) {
    throw new Error("Not a file");
  }
  if (stat.size > MAX_FILE_SIZE) {
    throw new Error(`File too large: ${stat.size} bytes (max ${MAX_FILE_SIZE})`);
  }
  const content = fs3.readFileSync(filePath, "utf-8");
  const lines = content.split("\n");
  offset = Math.max(0, offset);
  limit = Math.max(1, Math.min(limit, MAX_LINES));
  const selected = lines.slice(offset, offset + limit);
  return {
    content: selected.join("\n"),
    totalLines: lines.length
  };
}

// src/tools/grep_files.ts
import * as fs4 from "fs";
import * as path3 from "path";
import { minimatch } from "minimatch";
var MAX_MATCHES = 100;
var MAX_CONTENT_LENGTH = 200;
function grepFiles(rootPath2, pattern, userPath = ".", glob = "*") {
  const targetPath = validatePath(userPath, rootPath2);
  const stat = fs4.statSync(targetPath);
  const matches = [];
  if (stat.isFile()) {
    searchFile(targetPath, pattern, rootPath2, matches);
  } else if (stat.isDirectory()) {
    searchDirectory(targetPath, pattern, glob, rootPath2, matches);
  } else {
    throw new Error("Path is neither a file nor a directory");
  }
  return { matches: matches.slice(0, MAX_MATCHES) };
}
function searchFile(filePath, pattern, rootPath2, matches) {
  if (matches.length >= MAX_MATCHES) return;
  try {
    const stat = fs4.statSync(filePath);
    if (stat.size > 1024 * 1024) return;
    const content = fs4.readFileSync(filePath, "utf-8");
    const lines = content.split("\n");
    lines.forEach((line, i) => {
      if (matches.length >= MAX_MATCHES) return;
      if (line.includes(pattern)) {
        matches.push({
          file: path3.relative(rootPath2, filePath),
          line: i + 1,
          content: line.slice(0, MAX_CONTENT_LENGTH)
        });
      }
    });
  } catch {
  }
}
function searchDirectory(dirPath, pattern, glob, rootPath2, matches) {
  if (matches.length >= MAX_MATCHES) return;
  try {
    const entries = fs4.readdirSync(dirPath, { withFileTypes: true });
    for (const entry of entries) {
      if (matches.length >= MAX_MATCHES) break;
      const fullPath = path3.join(dirPath, entry.name);
      if (entry.isDirectory()) {
        if (["node_modules", ".git", "dist", "__pycache__"].includes(entry.name)) {
          continue;
        }
        searchDirectory(fullPath, pattern, glob, rootPath2, matches);
      } else if (entry.isFile()) {
        if (matchGlob(entry.name, glob)) {
          searchFile(fullPath, pattern, rootPath2, matches);
        }
      }
    }
  } catch {
  }
}
function matchGlob(filename, glob) {
  if (glob === "*") return true;
  return minimatch(filename, glob);
}

// src/index.ts
var args = process.argv.slice(2);
var rootPath = process.cwd();
for (let i = 0; i < args.length; i++) {
  if (args[i] === "--root-path" && args[i + 1]) {
    rootPath = args[i + 1];
    i++;
  }
}
var server = new Server(
  {
    name: "kimi-indexer-mcp",
    version: "1.0.0"
  },
  {
    capabilities: {
      tools: {}
    }
  }
);
server.setRequestHandler(ListToolsRequestSchema, async () => {
  return {
    tools: [
      {
        name: "list_files",
        description: "List files and directories. Use this to explore the file structure.",
        inputSchema: {
          type: "object",
          properties: {
            path: {
              type: "string",
              description: "Directory path (relative to root)",
              default: "."
            },
            depth: {
              type: "number",
              description: "Depth to expand (1=current only, max 5)",
              default: 1
            }
          }
        }
      },
      {
        name: "read_file",
        description: "Read text file content. Binary files are not supported.",
        inputSchema: {
          type: "object",
          properties: {
            path: {
              type: "string",
              description: "File path (relative to root)"
            },
            offset: {
              type: "number",
              description: "Starting line number (0-based)",
              default: 0
            },
            limit: {
              type: "number",
              description: "Number of lines to read (max 1000)",
              default: 500
            }
          },
          required: ["path"]
        }
      },
      {
        name: "grep_files",
        description: "Search for pattern in files. Returns matching lines. Supports both file and directory.",
        inputSchema: {
          type: "object",
          properties: {
            pattern: {
              type: "string",
              description: "Search string"
            },
            path: {
              type: "string",
              description: "File or directory path (relative to root)",
              default: "."
            },
            glob: {
              type: "string",
              description: "File name filter (only for directory mode)",
              default: "*"
            }
          },
          required: ["pattern"]
        }
      }
    ]
  };
});
server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args2 } = request.params;
  try {
    switch (name) {
      case "list_files": {
        const path4 = args2?.path || ".";
        const depth = args2?.depth || 1;
        const result = listFiles(rootPath, path4, depth);
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(result, null, 2)
            }
          ]
        };
      }
      case "read_file": {
        const path4 = args2?.path;
        if (!path4) {
          throw new McpError(ErrorCode.InvalidParams, "path is required");
        }
        const offset = args2?.offset || 0;
        const limit = args2?.limit || 500;
        const result = readFile(rootPath, path4, offset, limit);
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(result, null, 2)
            }
          ]
        };
      }
      case "grep_files": {
        const pattern = args2?.pattern;
        if (!pattern) {
          throw new McpError(ErrorCode.InvalidParams, "pattern is required");
        }
        const path4 = args2?.path || ".";
        const glob = args2?.glob || "*";
        const result = grepFiles(rootPath, pattern, path4, glob);
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(result, null, 2)
            }
          ]
        };
      }
      default:
        throw new McpError(ErrorCode.MethodNotFound, `Unknown tool: ${name}`);
    }
  } catch (error) {
    if (error instanceof McpError) {
      throw error;
    }
    const message = error instanceof Error ? error.message : String(error);
    throw new McpError(ErrorCode.InternalError, message);
  }
});
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error(`MCP server started with root: ${rootPath}`);
}
main().catch((error) => {
  console.error("Server error:", error);
  process.exit(1);
});
