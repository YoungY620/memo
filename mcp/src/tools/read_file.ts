/**
 * read_file tool implementation
 * Reads text file content with line-based pagination
 */

import * as fs from 'fs';
import { validatePath } from '../path.js';

export interface ReadFileResult {
  content: string;
  totalLines: number;
}

// Maximum file size: 10MB
const MAX_FILE_SIZE = 10 * 1024 * 1024;

// Maximum lines per request
const MAX_LINES = 1000;

/**
 * Reads file content with line-based pagination
 * @param rootPath - The root path for validation
 * @param userPath - User-provided file path (relative to root)
 * @param offset - Starting line number (0-based)
 * @param limit - Number of lines to read (max 1000)
 */
export function readFile(
  rootPath: string,
  userPath: string,
  offset: number = 0,
  limit: number = 500
): ReadFileResult {
  const filePath = validatePath(userPath, rootPath);
  
  const stat = fs.statSync(filePath);
  
  if (!stat.isFile()) {
    throw new Error('Not a file');
  }
  
  if (stat.size > MAX_FILE_SIZE) {
    throw new Error(`File too large: ${stat.size} bytes (max ${MAX_FILE_SIZE})`);
  }
  
  const content = fs.readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  
  // Clamp parameters
  offset = Math.max(0, offset);
  limit = Math.max(1, Math.min(limit, MAX_LINES));
  
  const selected = lines.slice(offset, offset + limit);
  
  return {
    content: selected.join('\n'),
    totalLines: lines.length
  };
}
