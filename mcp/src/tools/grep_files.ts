/**
 * grep_files tool implementation
 * Searches for patterns in files
 */

import * as fs from 'fs';
import * as path from 'path';
import { minimatch } from 'minimatch';
import { validatePath } from '../path.js';

export interface GrepMatch {
  file: string;
  line: number;
  content: string;
}

export interface GrepFilesResult {
  matches: GrepMatch[];
}

// Maximum matches to return
const MAX_MATCHES = 100;

// Maximum content length per match
const MAX_CONTENT_LENGTH = 200;

/**
 * Searches for pattern in files
 * @param rootPath - The root path for validation
 * @param pattern - Search string
 * @param userPath - File or directory path (relative to root)
 * @param glob - File name filter (only for directory mode)
 */
export function grepFiles(
  rootPath: string,
  pattern: string,
  userPath: string = '.',
  glob: string = '*'
): GrepFilesResult {
  const targetPath = validatePath(userPath, rootPath);
  const stat = fs.statSync(targetPath);
  const matches: GrepMatch[] = [];
  
  if (stat.isFile()) {
    searchFile(targetPath, pattern, rootPath, matches);
  } else if (stat.isDirectory()) {
    searchDirectory(targetPath, pattern, glob, rootPath, matches);
  } else {
    throw new Error('Path is neither a file nor a directory');
  }
  
  return { matches: matches.slice(0, MAX_MATCHES) };
}

function searchFile(
  filePath: string,
  pattern: string,
  rootPath: string,
  matches: GrepMatch[]
): void {
  if (matches.length >= MAX_MATCHES) return;
  
  try {
    const stat = fs.statSync(filePath);
    // Skip large files
    if (stat.size > 1024 * 1024) return;
    
    const content = fs.readFileSync(filePath, 'utf-8');
    const lines = content.split('\n');
    
    lines.forEach((line, i) => {
      if (matches.length >= MAX_MATCHES) return;
      
      if (line.includes(pattern)) {
        matches.push({
          file: path.relative(rootPath, filePath),
          line: i + 1,
          content: line.slice(0, MAX_CONTENT_LENGTH)
        });
      }
    });
  } catch {
    // Ignore unreadable files
  }
}

function searchDirectory(
  dirPath: string,
  pattern: string,
  glob: string,
  rootPath: string,
  matches: GrepMatch[]
): void {
  if (matches.length >= MAX_MATCHES) return;
  
  try {
    const entries = fs.readdirSync(dirPath, { withFileTypes: true });
    
    for (const entry of entries) {
      if (matches.length >= MAX_MATCHES) break;
      
      const fullPath = path.join(dirPath, entry.name);
      
      if (entry.isDirectory()) {
        // Skip common non-content directories
        if (['node_modules', '.git', 'dist', '__pycache__'].includes(entry.name)) {
          continue;
        }
        searchDirectory(fullPath, pattern, glob, rootPath, matches);
      } else if (entry.isFile()) {
        // Check glob pattern
        if (matchGlob(entry.name, glob)) {
          searchFile(fullPath, pattern, rootPath, matches);
        }
      }
    }
  } catch {
    // Ignore inaccessible directories
  }
}

function matchGlob(filename: string, glob: string): boolean {
  if (glob === '*') return true;
  return minimatch(filename, glob);
}
