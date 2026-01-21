/**
 * list_files tool implementation
 * Lists files and directories with configurable depth
 */

import * as fs from 'fs';
import * as path from 'path';
import { validatePath } from '../path.js';

export interface FileEntry {
  name: string;
  type: 'file' | 'directory';
  children?: FileEntry[];
}

export interface ListFilesResult {
  entries: FileEntry[];
}

/**
 * Lists files and directories
 * @param rootPath - The root path for validation
 * @param userPath - User-provided path (relative to root)
 * @param depth - Depth to expand (1 = current only, max 5)
 */
export function listFiles(
  rootPath: string,
  userPath: string = '.',
  depth: number = 1
): ListFilesResult {
  const dirPath = validatePath(userPath, rootPath);
  
  // Clamp depth to valid range
  depth = Math.max(1, Math.min(depth, 5));
  
  const stat = fs.statSync(dirPath);
  if (!stat.isDirectory()) {
    throw new Error('Path is not a directory');
  }
  
  return { entries: listDirectory(dirPath, depth) };
}

function listDirectory(dir: string, depth: number): FileEntry[] {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  
  return entries.map(entry => {
    const isDir = entry.isDirectory();
    const result: FileEntry = {
      name: entry.name,
      type: isDir ? 'directory' : 'file'
    };
    
    if (isDir && depth > 1) {
      try {
        result.children = listDirectory(path.join(dir, entry.name), depth - 1);
      } catch {
        // Ignore inaccessible directories
        result.children = [];
      }
    }
    
    return result;
  });
}
