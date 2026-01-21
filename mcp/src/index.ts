#!/usr/bin/env node
/**
 * MCP Server for kimi-indexer
 * Provides read-only file access for AI agents
 */

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  ErrorCode,
  McpError,
} from '@modelcontextprotocol/sdk/types.js';

import { listFiles } from './tools/list_files.js';
import { readFile } from './tools/read_file.js';
import { grepFiles } from './tools/grep_files.js';

// Parse command line arguments
const args = process.argv.slice(2);
let rootPath = process.cwd();

for (let i = 0; i < args.length; i++) {
  if (args[i] === '--root-path' && args[i + 1]) {
    rootPath = args[i + 1];
    i++;
  }
}

// Create MCP server
const server = new Server(
  {
    name: 'kimi-indexer-mcp',
    version: '1.0.0',
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// Register tools list handler
server.setRequestHandler(ListToolsRequestSchema, async () => {
  return {
    tools: [
      {
        name: 'list_files',
        description: 'List files and directories. Use this to explore the file structure.',
        inputSchema: {
          type: 'object',
          properties: {
            path: {
              type: 'string',
              description: 'Directory path (relative to root)',
              default: '.',
            },
            depth: {
              type: 'number',
              description: 'Depth to expand (1=current only, max 5)',
              default: 1,
            },
          },
        },
      },
      {
        name: 'read_file',
        description: 'Read text file content. Binary files are not supported.',
        inputSchema: {
          type: 'object',
          properties: {
            path: {
              type: 'string',
              description: 'File path (relative to root)',
            },
            offset: {
              type: 'number',
              description: 'Starting line number (0-based)',
              default: 0,
            },
            limit: {
              type: 'number',
              description: 'Number of lines to read (max 1000)',
              default: 500,
            },
          },
          required: ['path'],
        },
      },
      {
        name: 'grep_files',
        description: 'Search for pattern in files. Returns matching lines. Supports both file and directory.',
        inputSchema: {
          type: 'object',
          properties: {
            pattern: {
              type: 'string',
              description: 'Search string',
            },
            path: {
              type: 'string',
              description: 'File or directory path (relative to root)',
              default: '.',
            },
            glob: {
              type: 'string',
              description: 'File name filter (only for directory mode)',
              default: '*',
            },
          },
          required: ['pattern'],
        },
      },
    ],
  };
});

// Register tool call handler
server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;

  try {
    switch (name) {
      case 'list_files': {
        const path = (args?.path as string) || '.';
        const depth = (args?.depth as number) || 1;
        const result = listFiles(rootPath, path, depth);
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(result, null, 2),
            },
          ],
        };
      }

      case 'read_file': {
        const path = args?.path as string;
        if (!path) {
          throw new McpError(ErrorCode.InvalidParams, 'path is required');
        }
        const offset = (args?.offset as number) || 0;
        const limit = (args?.limit as number) || 500;
        const result = readFile(rootPath, path, offset, limit);
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(result, null, 2),
            },
          ],
        };
      }

      case 'grep_files': {
        const pattern = args?.pattern as string;
        if (!pattern) {
          throw new McpError(ErrorCode.InvalidParams, 'pattern is required');
        }
        const path = (args?.path as string) || '.';
        const glob = (args?.glob as string) || '*';
        const result = grepFiles(rootPath, pattern, path, glob);
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(result, null, 2),
            },
          ],
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

// Start server
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error(`MCP server started with root: ${rootPath}`);
}

main().catch((error) => {
  console.error('Server error:', error);
  process.exit(1);
});
